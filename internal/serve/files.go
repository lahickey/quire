package serve

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var variantName = regexp.MustCompile(`^[0-9a-f]{32}\.[a-z0-9]{3,4}$`)

var copyBufs = sync.Pool{New: func() any { b := make([]byte, 256<<10); return &b }}

// Filer is a source whose files can be fetched: a bundle is one, a baker is not.
type Filer interface {
	fs.FS
	Served(file string) (mediaType string, size int64, ok bool)
}

type files struct {
	from        Filer
	dev         bool
	memoryLimit int
	cache       *cache[[]byte]
}

// writerOnly hides the response's ReadFrom: net's fallback allocates a buffer
// per response (golang/go#38389), so anything sendfile can't take goes through a
// pooled one.
type writerOnly struct{ io.Writer }

func stream(w io.Writer, f fs.File) {
	if _, ok := f.(syscall.Conn); ok {
		_, _ = io.Copy(w, f)
		return
	}
	buf := copyBufs.Get().(*[]byte)
	defer copyBufs.Put(buf)
	_, _ = io.CopyBuffer(writerOnly{w}, f, *buf)
}

func (s *files) headers(h http.Header, mediaType, file string) {
	h.Set("Content-Type", mediaType)
	h.Set("ETag", `"`+file+`"`)
	if !s.dev {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	}
}

func Files(from Filer, memory, memoryLimit int, dev bool) http.Handler {
	s := &files{from: from, dev: dev, memoryLimit: memoryLimit}
	if memory > 0 && memoryLimit > 0 {
		s.cache = newCache(memory, func(b []byte) int { return len(b) })
	}
	return s
}

func (s *files) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	if !variantName.MatchString(file) {
		http.NotFound(w, r)
		return
	}
	mediaType, size, ok := s.from.Served(file)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var body []byte
	var f fs.File
	hit := false
	if s.cache != nil {
		body, hit = s.cache.get(file)
	}
	if !hit {
		var err error
		if f, err = s.from.Open(file); err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = f.Close() }()
	}
	h := w.Header()
	if r.Header.Get("Range") != "" || r.Header.Get("If-None-Match") != "" || r.Header.Get("If-Range") != "" {
		s.headers(h, mediaType, file)
		var rs io.ReadSeeker = bytes.NewReader(body)
		if f != nil {
			if rs, ok = f.(io.ReadSeeker); !ok {
				http.Error(w, "quire: bundle file is not seekable", http.StatusInternalServerError)
				return
			}
		}
		http.ServeContent(w, r, file, time.Time{}, rs)
		return
	}
	if r.Method == http.MethodHead {
		s.headers(h, mediaType, file)
		h.Set("Content-Length", strconv.FormatInt(size, 10))
		w.WriteHeader(http.StatusOK)
		return
	}

	if !hit && s.cache != nil && size <= int64(s.memoryLimit) {
		body = make([]byte, size)
		if _, err := io.ReadFull(f, body); err != nil {
			http.NotFound(w, r)
			return
		}
		s.cache.put(file, body)
		hit = true
	}
	s.headers(h, mediaType, file)
	h.Set("Content-Length", strconv.FormatInt(size, 10))
	w.WriteHeader(http.StatusOK)
	if hit {
		_, _ = w.Write(body)
		return
	}
	stream(w, f)
}
