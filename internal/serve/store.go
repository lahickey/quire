package serve

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lahickey/quire/internal/journal"
)

var variantName = regexp.MustCompile(`^[0-9a-f]{32}\.[a-z]{3,4}$`)

var copyBufs = sync.Pool{New: func() any { b := make([]byte, 256<<10); return &b }}

type variant struct {
	size   int64
	format journal.Format
}

type Store struct {
	records     map[string]journal.Record
	fsys        fs.FS
	fontID      string
	dev         bool
	memoryLimit int
	cache       *cache[[]byte]
	byKey       map[string]variant
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

func (s *Store) headers(h http.Header, v variant, file string) {
	h.Set("Content-Type", v.format.Type)
	h.Set("ETag", `"`+file+`"`)
	if !s.dev {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	}
}

func NewStore(memory, memoryLimit int, dev bool, fontID string) *Store {
	s := &Store{
		fontID:      fontID,
		dev:         dev,
		memoryLimit: memoryLimit,
		byKey:       map[string]variant{},
	}
	if memory > 0 && memoryLimit > 0 {
		s.cache = newCache(memory, func(b []byte) int { return len(b) })
	}
	return s
}

func (s *Store) Load(fsys fs.FS) error {
	recs, err := journal.LoadJournal(fsys)
	if err != nil {
		return err
	}
	s.records, s.fsys = recs, fsys
	for _, rec := range recs {
		if rec.Kind != journal.KindPicture {
			continue
		}
		for _, vs := range rec.Formats {
			for _, v := range vs.Out {
				s.byKey[v.Name] = variant{format: vs.Format(), size: v.Size}
			}
		}
	}
	return nil
}

func (s *Store) Record(name string) (journal.Record, bool) {
	r, ok := s.records[name]
	return r, ok
}

func (s *Store) File(name string) ([]byte, bool, error) {
	r, ok := s.records[name]
	if !ok {
		return nil, false, nil
	}
	file, ok := r.File()
	if !ok {
		return nil, true, fmt.Errorf("%s: the bundle holds it as more than one file", name)
	}
	b, err := fs.ReadFile(s.fsys, file)
	return b, true, err
}

func (s *Store) Master(name string) (journal.Record, error) {
	r, ok := s.Record(name)
	if !ok {
		return journal.Record{}, fmt.Errorf("quire: image %s is in no journal: run the site's bake, `go run . bake`", name)
	}
	return r, nil
}

func (s *Store) Diagram(src string) (string, error) {
	svg, ok, err := s.File(journal.DiagramName(src, s.fontID))
	if !ok {
		first, _, _ := strings.Cut(strings.TrimSpace(src), "\n")
		return "", fmt.Errorf("diagram %q is not baked in these fonts; run the site's bake, `go run . bake`", first)
	}
	return string(svg), err
}

func (s *Store) Documents(pattern string, _ []string) ([]journal.Document, error) {
	var out []journal.Document
	for _, name := range slices.Sorted(maps.Keys(s.records)) {
		ok, err := path.Match(pattern, name)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", pattern, err)
		}
		if !ok || s.records[name].Kind != journal.KindDocument {
			continue
		}
		body, _, err := s.File(name)
		if err != nil {
			return nil, err
		}
		d, err := journal.DecodeDocument(body)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	if !variantName.MatchString(file) {
		http.NotFound(w, r)
		return
	}
	v, ok := s.byKey[file]
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
		if f, err = s.fsys.Open(file); err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = f.Close() }()
	}
	h := w.Header()
	if r.Header.Get("Range") != "" || r.Header.Get("If-None-Match") != "" || r.Header.Get("If-Range") != "" {
		s.headers(h, v, file)
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
		s.headers(h, v, file)
		h.Set("Content-Length", strconv.FormatInt(v.size, 10))
		w.WriteHeader(http.StatusOK)
		return
	}

	if !hit && s.cache != nil && v.size <= int64(s.memoryLimit) {
		body = make([]byte, v.size)
		if _, err := io.ReadFull(f, body); err != nil {
			http.NotFound(w, r)
			return
		}
		s.cache.put(file, body)
		hit = true
	}
	s.headers(h, v, file)
	h.Set("Content-Length", strconv.FormatInt(v.size, 10))
	w.WriteHeader(http.StatusOK)
	if hit {
		_, _ = w.Write(body)
		return
	}
	stream(w, f)
}
