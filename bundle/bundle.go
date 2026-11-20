// Package bundle reads what a bake leaves for a site: a journal, and the content-addressed files it names.
package bundle

import (
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/model"
)

type served struct {
	mediaType string
	size      int64
}

// Bundle answers what a site asks of a source by reading, and converts nothing. The zero value is an empty bundle.
type Bundle struct {
	records map[string]journal.Record
	fsys    fs.FS
	served  map[string]served
}

func (b *Bundle) baked(what, name string) (journal.Record, error) {
	r, ok := b.records[name]
	if !ok {
		return r, fmt.Errorf("quire: %s %s is not baked: run the site's bake, `go run . bake`", what, name)
	}
	return r, nil
}

func (b *Bundle) file(what, name string) (string, error) {
	r, err := b.baked(what, name)
	if err != nil {
		return "", err
	}
	file, ok := r.File()
	if !ok {
		return "", fmt.Errorf("quire: %s is not one file", name)
	}
	return file, nil
}

func Open(fsys fs.FS) (*Bundle, error) {
	recs, err := journal.Load(fsys)
	if err != nil {
		return nil, err
	}
	b := &Bundle{records: recs, fsys: fsys, served: map[string]served{}}
	for _, r := range recs {
		mediaType := r.Type
		if r.Kind == journal.KindPicture {
			mediaType = r.Picture().Type
		}
		if mediaType == "" {
			continue
		}
		for _, f := range r.Files {
			b.served[f.Name] = served{mediaType, f.Size}
		}
	}
	return b, nil
}

// Open opens one of the bundle's files, so a bundle is an fs.FS of them.
func (b *Bundle) Open(file string) (fs.File, error) {
	if b.fsys == nil {
		return nil, &fs.PathError{Op: "open", Path: file, Err: fs.ErrNotExist}
	}
	return b.fsys.Open(file)
}

// Served is the media type and size of a file a browser may fetch. A document or an inlined diagram is not one.
func (b *Bundle) Served(file string) (mediaType string, size int64, ok bool) {
	s, ok := b.served[file]
	return s.mediaType, s.size, ok
}

func (b *Bundle) Names(pattern string) ([]string, error) {
	var out []string
	for _, name := range slices.Sorted(maps.Keys(b.records)) {
		ok, err := path.Match(pattern, name)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", pattern, err)
		}
		if ok && b.records[name].Kind == journal.KindDocument {
			out = append(out, name)
		}
	}
	return out, nil
}

func (b *Bundle) Document(name string) (model.BakedDocument, error) {
	file, err := b.file("document", name)
	if err != nil {
		return model.BakedDocument{}, err
	}
	body, err := b.Body(file)
	if err != nil {
		return model.BakedDocument{}, err
	}
	d, err := model.DecodeBaked(body)
	if err != nil {
		return d, fmt.Errorf("%s: %w", name, err)
	}
	return d, nil
}

func (b *Bundle) Picture(name string) (model.Picture, error) {
	r, err := b.baked("picture", name)
	if err != nil {
		return model.Picture{}, err
	}
	p := r.Picture()
	files := map[int]string{}
	for i, w := range p.Widths {
		files[w] = r.Files[i].Name
	}
	return model.Picture{Hash: p.Hash, Width: p.W, Height: p.H, Alpha: p.Alpha, Files: files}, nil
}

func (b *Bundle) File(name string) (string, error) { return b.file("file", name) }

func (b *Bundle) Converted(converter string, body []byte, args ...string) (string, error) {
	r, ok := b.records[journal.ConvertedName(converter, body, args)]
	file, one := r.File()
	if !ok || !one {
		first, _, _ := strings.Cut(string(body), "\n")
		return "", fmt.Errorf("%.60q is not baked as this site asks for it: run the site's bake, `go run . bake`", first)
	}
	return file, nil
}

func (b *Bundle) Body(file string) ([]byte, error) { return fs.ReadFile(b, file) }
