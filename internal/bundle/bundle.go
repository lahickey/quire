package bundle

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
	"github.com/lahickey/quire/internal/lru"
	"github.com/lahickey/quire/model"
)

type loaded struct {
	records map[string]journal.Record
	fsys    fs.FS
}

type variant struct {
	fsys   fs.FS
	size   int64
	format journal.Format
}

type Store struct {
	journals    []loaded
	fontID      string
	dev         bool
	memoryLimit int
	cache       *lru.Cache[[]byte]

	mu     sync.Mutex
	byKey  map[string]variant
	byName map[string]model.Image
	maxOf  map[string]model.Slot
}

func New(memory, memoryLimit int, dev bool, fontID string) *Store {
	s := &Store{
		fontID:      fontID,
		dev:         dev,
		memoryLimit: memoryLimit,
		byKey:       map[string]variant{},
		byName:      map[string]model.Image{},
		maxOf:       map[string]model.Slot{},
	}
	if memory > 0 && memoryLimit > 0 {
		s.cache = lru.New(memory, func(b []byte) int { return len(b) })
	}
	return s
}

func (s *Store) add(j loaded) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.journals = append(s.journals, j)
	for _, rec := range j.records {
		for _, vs := range rec.Formats {
			f := vs.Format()
			if !strings.HasPrefix(f.Type, "image/") {
				continue
			}
			for _, v := range vs.Out {
				if _, taken := s.byKey[v.Name]; taken {
					continue
				}
				s.byKey[v.Name] = variant{format: f, fsys: j.fsys, size: v.Size}
			}
		}
	}
}

func (s *Store) AddBundle(fsys fs.FS) error {
	recs, err := journal.LoadJournal(fsys)
	if err != nil {
		return err
	}
	s.add(loaded{records: recs, fsys: fsys})
	return nil
}

func (s *Store) Record(name string) (journal.Record, bool) {
	for _, j := range s.journals {
		if r, ok := j.records[name]; ok {
			return r, true
		}
	}
	return journal.Record{}, false
}

func (s *Store) File(name string) ([]byte, bool, error) {
	for _, j := range s.journals {
		r, ok := j.records[name]
		if !ok {
			continue
		}
		file, ok := r.File()
		if !ok {
			return nil, true, fmt.Errorf("%s: the bundle holds it as more than one file", name)
		}
		b, err := fs.ReadFile(j.fsys, file)
		return b, true, err
	}
	return nil, false, nil
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

func (s *Store) Data(name string) ([]byte, error) {
	b, ok, err := s.File(name)
	if !ok {
		return nil, fmt.Errorf("quire: %s is in no journal: run the site's bake, `go run . bake`", name)
	}
	return b, err
}

func (s *Store) Documents(pattern string, _ []string) ([]journal.Document, error) {
	seen := map[string]bool{}
	for _, j := range s.journals {
		for name, r := range j.records {
			if len(r.Formats) == 1 && r.Formats[0].Format() == journal.JSON {
				seen[name] = true
			}
		}
	}
	var out []journal.Document
	for _, name := range slices.Sorted(maps.Keys(seen)) {
		ok, err := path.Match(pattern, name)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", pattern, err)
		}
		if !ok {
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

func (s *Store) Mounted(name string, want model.Slot) (model.Image, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	im, ok := s.byName[name]
	return im, ok && s.maxOf[name] >= want
}

func (s *Store) Mount(name string, im model.Image, max model.Slot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byName[name] = im
	s.maxOf[name] = max
}

var variantName = regexp.MustCompile(`^[0-9a-f]{32}\.[a-z]{3,4}$`)

var copyBufs = sync.Pool{New: func() any { b := make([]byte, 256<<10); return &b }}

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

func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	if !variantName.MatchString(file) {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	v, ok := s.byKey[file]
	s.mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	var body []byte
	var f fs.File
	hit := false
	if s.cache != nil {
		body, hit = s.cache.Get(file)
	}
	if !hit {
		var err error
		if f, err = v.fsys.Open(file); err != nil {
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
		s.cache.Put(file, body)
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
