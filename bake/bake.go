// Package bake converts what a site asks for, once, into a journal and content-addressed files.
package bake

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/lahickey/quire/bundle"
	"github.com/lahickey/quire/internal/cache"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/model"
)

// Input is one thing to convert. Name is its file under the content root when it came from one, and ID is unique to the input.
type Input struct {
	Name string
	ID   string
	Body []byte
	Args []string
}

// Output is one file. Type is its media type when a browser fetches it, and empty when only the site reads it.
type Output struct {
	Body []byte
	Ext  string
	Type string
}

// Converter turns an input into one file. Its version is in the key of everything it makes.
type Converter interface {
	Name() string
	Version() int
	Convert(Input) (Output, error)
}

// MasterInfo is a master as it was read: its size once fitted and turned upright, and the size of the file it came from.
type MasterInfo struct {
	W, H       int
	SrcW, SrcH int
	Alpha      bool
}

// Master is a picture opened once, to be asked for any number of rungs.
type Master interface {
	Info() MasterInfo
	Rung(width int) ([]byte, error)
}

// Codec opens masters and writes their rungs as files of one extension and media type.
// Its ID is in the name of every file it writes, so it covers the encoder and its settings.
type Codec interface {
	Version() int
	Ext() string
	Type() string
	ID() string
	Open(name string, src []byte) (Master, error)
}

// Tools is a set of what a bake can convert: a converter per document extension, converters by name, and the codec for pictures.
type Tools struct {
	Documents  map[string]Converter
	Converters []Converter
	Pictures   Codec
}

// Baker is a source that converts: a site asks it for content, and Finish writes what was asked for as a bundle.
type Baker struct {
	spec   model.BakeSpec
	tools  Tools
	source fs.FS
	cache  cache.Dir
	done   map[string]journal.Record
	wanted map[string]int64
	made   map[string]journal.Record
}

func report(name string, encoded, kept int) {
	log.Printf("%-30s %2d encoded %2d kept", name, encoded, kept)
}

func within(dir, p string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func apart(content, dir string) error {
	if content == "" {
		return nil
	}
	c, err := filepath.Abs(content)
	if err != nil {
		return err
	}
	d, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if within(c, d) || within(d, c) {
		return fmt.Errorf("quire: bake content %s and %s overlap", c, d)
	}
	return nil
}

func (b *Baker) read(name string) ([]byte, error) {
	if b.source == nil {
		return nil, fmt.Errorf("quire: %s: no content to read it from", name)
	}
	return fs.ReadFile(b.source, name)
}

func (b *Baker) single(rec journal.Record, make func() (Output, error)) ([]byte, error) {
	if made, ok := b.made[rec.Name]; ok {
		file, _ := made.File()
		return b.cache.Read(file)
	}
	if prev := b.done[rec.Name]; prev.Key == rec.Key {
		if file, ok := prev.File(); ok {
			if body, err := b.cache.Read(file); err == nil {
				b.made[rec.Name] = prev
				report(rec.Name, 0, 1)
				return body, nil
			}
		}
	}
	out, err := make()
	if err != nil {
		return nil, err
	}
	file := journal.Hash(out.Body) + "." + out.Ext
	if err := b.cache.Write(file, out.Body); err != nil {
		return nil, err
	}
	rec.Type, rec.Files = out.Type, []journal.File{{Name: file, Size: int64(len(out.Body))}}
	b.made[rec.Name] = rec
	report(rec.Name, 1, 0)
	return out.Body, nil
}

func variants(p journal.Picture, c Codec) []journal.File {
	var out []journal.File
	for _, w := range p.Widths {
		out = append(out, journal.File{Name: journal.Hash(fmt.Appendf(nil, "%s|%d|%s", p.Hash, w, c.ID())) + "." + c.Ext()})
	}
	return out
}

func (b *Baker) work(name string) (journal.Record, int, error) {
	src, err := b.read(name)
	if err != nil {
		return journal.Record{}, 0, err
	}
	c := b.tools.Pictures
	hash := journal.Hash(src)
	key := journal.PictureKey(c.Version(), hash)
	var master Master
	open := func() (err error) {
		master, err = c.Open(name, src)
		return err
	}
	p := b.done[name].Picture()
	if b.done[name].Key != key {
		if err := open(); err != nil {
			return journal.Record{}, 0, err
		}
		m := master.Info()
		p = journal.Picture{Hash: hash, W: m.W, H: m.H, SrcW: m.SrcW, SrcH: m.SrcH, Alpha: m.Alpha, Type: c.Type(), Widths: model.Ladder(m.W)}
	}

	rec := journal.Record{Name: name, Kind: journal.KindPicture, Key: key, Files: variants(p, c)}.WithPicture(p)
	made := 0
	for i := range rec.Files {
		v := &rec.Files[i]
		if size, ok := b.cache.Size(v.Name); ok && size > 0 {
			v.Size = size
			continue
		}
		if master == nil {
			if err := open(); err != nil {
				return journal.Record{}, 0, err
			}
		}
		body, err := master.Rung(p.Widths[i])
		if err != nil {
			return journal.Record{}, 0, fmt.Errorf("quire: %s: %w", name, err)
		}
		if err := b.cache.Write(v.Name, body); err != nil {
			return journal.Record{}, 0, err
		}
		v.Size = int64(len(body))
		made++
	}
	return rec, made, nil
}

func (b *Baker) encode() (map[string]journal.Record, error) {
	records := map[string]journal.Record{}
	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		first error
	)
	ch := make(chan string)
	for range min(runtime.GOMAXPROCS(0), len(b.wanted)) {
		wg.Go(func() {
			for name := range ch {
				mu.Lock()
				stop := first != nil
				mu.Unlock()
				if stop {
					continue
				}
				rec, made, err := b.work(name)
				mu.Lock()
				switch {
				case err != nil && first == nil:
					first = err
				case err == nil:
					records[rec.Name] = rec
					report(rec.Name, made, len(rec.Files)-made)
				}
				mu.Unlock()
			}
		})
	}
	names := slices.SortedFunc(maps.Keys(b.wanted), func(x, y string) int { return cmp.Compare(b.wanted[y], b.wanted[x]) })
	for _, name := range names {
		ch <- name
	}
	close(ch)
	wg.Wait()
	return records, first
}

func merge(sets []Tools) (Tools, error) {
	out := Tools{Documents: map[string]Converter{}}
	for _, t := range sets {
		for ext, c := range t.Documents {
			if _, taken := out.Documents[ext]; taken {
				return out, fmt.Errorf("quire: two converters make a document of %s", ext)
			}
			out.Documents[ext] = c
		}
		for _, c := range t.Converters {
			if slices.ContainsFunc(out.Converters, func(o Converter) bool { return o.Name() == c.Name() }) {
				return out, fmt.Errorf("quire: two converters are named %q", c.Name())
			}
			out.Converters = append(out.Converters, c)
		}
		if t.Pictures == nil {
			continue
		}
		if out.Pictures != nil {
			return out, errors.New("quire: two codecs for pictures")
		}
		out.Pictures = t.Pictures
	}
	return out, nil
}

// Start opens a bake that can convert what the sets hold between them.
func Start(s model.BakeSpec, sets ...Tools) (*Baker, error) {
	tools, err := merge(sets)
	if err != nil {
		return nil, err
	}
	if err := apart(s.Content, s.Dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.Cache(), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(s.Dir, ".gitignore"), []byte("*\n"), 0o644); err != nil {
		return nil, err
	}
	b := &Baker{
		spec:   s,
		tools:  tools,
		cache:  cache.Dir(s.Cache()),
		done:   cache.Dir(s.Cache()).Journal(),
		wanted: map[string]int64{},
		made:   map[string]journal.Record{},
	}
	if s.Content != "" {
		b.source = os.DirFS(s.Content)
	}
	return b, nil
}

func (b *Baker) Names(pattern string) ([]string, error) {
	if b.source == nil {
		return nil, nil
	}
	names, err := fs.Glob(b.source, pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %q: %w", pattern, err)
	}
	slices.Sort(names)
	return names, nil
}

func (b *Baker) Document(name string) (model.BakedDocument, error) {
	c := b.tools.Documents[path.Ext(name)]
	if c == nil {
		return model.BakedDocument{}, fmt.Errorf("quire: %s: no converter makes a document of it", name)
	}
	raw, err := b.read(name)
	if err != nil {
		return model.BakedDocument{}, err
	}
	key := journal.Hash(fmt.Appendf(nil, "%s\x00%d\x00%s", c.Name(), c.Version(), raw))
	body, err := b.single(journal.Record{Name: name, Kind: journal.KindDocument, Key: key}, func() (Output, error) {
		out, err := c.Convert(Input{Name: name, ID: journal.Hash(raw), Body: raw})
		if err != nil {
			return out, fmt.Errorf("%s: %w", name, err)
		}
		return out, nil
	})
	if err != nil {
		return model.BakedDocument{}, err
	}
	return model.DecodeBaked(body)
}

// Picture notes that a master is wanted. Pictures are encoded together by Finish, so nothing is known of it yet
// and what comes back is empty: the picture is in the bundle Finish returns.
func (b *Baker) Picture(name string) (model.Picture, error) {
	if b.tools.Pictures == nil {
		return model.Picture{}, fmt.Errorf("quire: %s: the bake has no codec for pictures", name)
	}
	if b.source == nil {
		return model.Picture{}, fmt.Errorf("quire: %s: no content to read it from", name)
	}
	st, err := fs.Stat(b.source, name)
	if err != nil {
		return model.Picture{}, err
	}
	b.wanted[name] = st.Size()
	return model.Picture{}, nil
}

func (b *Baker) madeFile(name string, err error) (string, error) {
	file, _ := b.made[name].File()
	return file, err
}

func (b *Baker) File(name string) (string, error) {
	src, err := b.read(name)
	if err != nil {
		return "", err
	}
	rec := journal.Record{Name: name, Kind: journal.KindFile, Key: journal.Hash(src)}
	_, err = b.single(rec, func() (Output, error) {
		return Output{Body: src, Ext: strings.TrimPrefix(path.Ext(name), "."), Type: journal.VideoType(name)}, nil
	})
	return b.madeFile(name, err)
}

func (b *Baker) Converted(converter string, body []byte, args ...string) (string, error) {
	i := slices.IndexFunc(b.tools.Converters, func(c Converter) bool { return c.Name() == converter })
	if i < 0 {
		return "", fmt.Errorf("quire: the bake has no converter %q", converter)
	}
	c := b.tools.Converters[i]
	name := journal.ConvertedName(converter, body, args)
	key := journal.Hash(fmt.Appendf(nil, "%s\x00%d", name, c.Version()))
	_, err := b.single(journal.Record{Name: name, Kind: journal.Kind(converter), Key: key}, func() (Output, error) {
		return c.Convert(Input{ID: path.Base(name), Body: body, Args: args})
	})
	return b.madeFile(name, err)
}

func (b *Baker) Body(file string) ([]byte, error) { return b.cache.Read(file) }

// Finish encodes the pictures asked for, journals everything, and returns the bundle of what this bake was asked for.
func (b *Baker) Finish() (*bundle.Bundle, error) {
	records, first := b.encode()
	maps.Copy(records, b.made)

	kept := maps.Clone(b.done)
	maps.Copy(kept, records)
	if err := b.cache.Write(journal.Name, journal.Write(slices.Collect(maps.Values(kept)))); err != nil {
		return nil, errors.Join(first, err)
	}
	if first != nil {
		return nil, first
	}
	if err := b.cache.CopyOut(b.spec.Bundle(), slices.Collect(maps.Values(records))); err != nil {
		return nil, err
	}
	return bundle.Open(os.DirFS(b.spec.Bundle()))
}
