// Package bundle is what a bake leaves for a site: a journal of records, and the content-addressed files they name.
package bundle

import (
	"fmt"
	"io/fs"
	"iter"
	"maps"
	"path"
	"slices"
	"strings"
)

// Bundle reads a journal and its files. It answers what a site asks of a source, and converts nothing.
// The zero value is an empty bundle.
type Bundle struct {
	records map[string]Record
	fsys    fs.FS
}

func (b *Bundle) baked(what, name string) (Record, error) {
	r, ok := b.records[name]
	if !ok {
		return Record{}, fmt.Errorf("quire: %s %s is not baked: run the site's bake, `go run . bake`", what, name)
	}
	return r, nil
}

func Open(fsys fs.FS) (*Bundle, error) {
	recs, err := LoadJournal(fsys)
	if err != nil {
		return nil, err
	}
	return &Bundle{records: recs, fsys: fsys}, nil
}

func (b *Bundle) Records() iter.Seq[Record] { return maps.Values(b.records) }

func (b *Bundle) Open(name string) (fs.File, error) {
	if b.fsys == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return b.fsys.Open(name)
}

func (b *Bundle) Names(pattern string) ([]string, error) {
	var out []string
	for _, name := range slices.Sorted(maps.Keys(b.records)) {
		ok, err := path.Match(pattern, name)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", pattern, err)
		}
		if ok && b.records[name].Kind == KindDocument {
			out = append(out, name)
		}
	}
	return out, nil
}

func (b *Bundle) Document(name string, _ []string) (Document, error) {
	rec, err := b.baked("document", name)
	if err != nil {
		return Document{}, err
	}
	body, err := b.Body(rec)
	if err != nil {
		return Document{}, err
	}
	d, err := DecodeDocument(body)
	if err != nil {
		return d, fmt.Errorf("%s: %w", name, err)
	}
	return d, nil
}

func (b *Bundle) Picture(name string) (Record, error) { return b.baked("picture", name) }

func (b *Bundle) File(name string) (Record, error) { return b.baked("file", name) }

func (b *Bundle) Converted(converter string, body []byte, args ...string) (Record, error) {
	rec, ok := b.records[ConvertedName(converter, body, args)]
	if !ok {
		first, _, _ := strings.Cut(string(body), "\n")
		return rec, fmt.Errorf("quire: %s %.60q is not baked as this site asks for it: run the site's bake, `go run . bake`", converter, first)
	}
	return rec, nil
}

func (b *Bundle) Body(rec Record) ([]byte, error) {
	file, ok := rec.File()
	if !ok {
		return nil, fmt.Errorf("quire: %s is not one file", rec.Name)
	}
	return fs.ReadFile(b, file)
}
