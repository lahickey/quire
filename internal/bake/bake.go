//go:build !quire_nobake

package bake

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/lahickey/quire/internal/bake/cache"
	"github.com/lahickey/quire/internal/bake/card"
	"github.com/lahickey/quire/internal/bake/diagram"
	"github.com/lahickey/quire/internal/bake/markdown"
	"github.com/lahickey/quire/internal/bake/picture"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/model"
)

type Run struct {
	spec   model.BakeSpec
	source fs.FS
	cache  cache.Dir
	done   map[string]journal.Record
	draw   *diagram.Renderer
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

func (b *Run) single(rec journal.Record, ext string, make func() ([]byte, error)) ([]byte, error) {
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
	body, err := make()
	if err != nil {
		return nil, err
	}
	file := journal.Hash(body) + "." + ext
	if err := b.cache.Write(file, body); err != nil {
		return nil, err
	}
	rec.Files = []journal.File{{Name: file, Size: int64(len(body))}}
	b.made[rec.Name] = rec
	report(rec.Name, 1, 0)
	return body, nil
}

func (b *Run) document(name string, fences []string) (journal.Document, error) {
	raw, err := fs.ReadFile(b.source, name)
	if err != nil {
		return journal.Document{}, err
	}
	key := journal.Hash(fmt.Appendf(nil, "%d\x00%s\x00%s", version("markdown"), strings.Join(fences, "\x00"), raw))
	body, err := b.single(journal.Record{Name: name, Kind: journal.KindDocument, Key: key}, "json", func() ([]byte, error) {
		d, err := markdown.Convert(name, raw, fences)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return journal.EncodeDocument(d)
	})
	if err != nil {
		return journal.Document{}, err
	}
	return journal.DecodeDocument(body)
}

func (b *Run) work(name string) (journal.Record, int, error) {
	src, err := fs.ReadFile(b.source, name)
	if err != nil {
		return journal.Record{}, 0, err
	}
	hash := journal.Hash(src)
	key := journal.PictureKey(version("picture"), hash)
	var img *image.NRGBA
	decode := func() (err error) {
		img, err = picture.Read(name, src, journal.MasterCap)
		return err
	}
	m := b.done[name].Picture.Master
	if b.done[name].Key != key {
		if err := decode(); err != nil {
			return journal.Record{}, 0, err
		}
		m = picture.Describe(img, src, hash)
	}

	f := picture.FormatWebP
	rec := journal.Record{
		Name: name, Kind: journal.KindPicture, Key: key,
		Files: picture.Variants(m, f), Picture: journal.Picture{Master: m, Format: f},
	}
	made := 0
	for i := range rec.Files {
		v := &rec.Files[i]
		if size, ok := b.cache.Size(v.Name); ok && size > 0 {
			v.Size = size
			continue
		}
		if img == nil {
			if err := decode(); err != nil {
				return journal.Record{}, 0, err
			}
		}
		body, err := picture.Encode(img, v.W, f)
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

func (b *Run) encode() (map[string]journal.Record, error) {
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

func Start(s model.BakeSpec) (*Run, error) {
	if err := apart(s.Content, s.Dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.Cache(), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(s.Dir, ".gitignore"), []byte("*\n"), 0o644); err != nil {
		return nil, err
	}
	b := &Run{
		spec:   s,
		cache:  cache.Dir(s.Cache()),
		done:   cache.Dir(s.Cache()).Journal(),
		draw:   diagram.Default(),
		wanted: map[string]int64{},
		made:   map[string]journal.Record{},
	}
	if s.Content != "" {
		b.source = os.DirFS(s.Content)
	}
	return b, nil
}

func (b *Run) Master(name string) (journal.Record, error) {
	if b.source == nil {
		return journal.Record{}, fmt.Errorf("quire: %s: no content to read masters from", name)
	}
	st, err := fs.Stat(b.source, name)
	if err != nil {
		return journal.Record{}, err
	}
	b.wanted[name] = st.Size()
	return journal.Record{Name: name}, nil
}

func (b *Run) Diagram(src string) (string, error) {
	name := journal.DiagramName(src, b.draw.FontID())
	key := journal.Hash(fmt.Appendf(nil, "%s\x00%d", name, version("diagram")))
	svg, err := b.single(journal.Record{Name: name, Kind: journal.KindDiagram, Key: key}, "svg", func() ([]byte, error) {
		svg, err := b.draw.Render(src)
		if err != nil {
			first, _, _ := strings.Cut(strings.TrimSpace(src), "\n")
			return nil, fmt.Errorf("quire: diagram %q: %w", first, err)
		}
		return []byte(svg), nil
	})
	return string(svg), err
}

func (b *Run) Documents(pattern string, fences []string) ([]journal.Document, error) {
	if b.source == nil {
		return nil, nil
	}
	names, err := fs.Glob(b.source, pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %q: %w", pattern, err)
	}
	slices.Sort(names)
	out := make([]journal.Document, 0, len(names))
	for _, name := range names {
		d, err := b.document(name, fences)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (b *Run) Finish() error {
	records, first := b.encode()
	maps.Copy(records, b.made)

	kept := maps.Clone(b.done)
	maps.Copy(kept, records)
	if err := b.cache.Write(journal.JournalName, journal.WriteJournal(slices.Collect(maps.Values(kept)))); err != nil {
		return errors.Join(first, err)
	}
	if first != nil {
		return first
	}
	return b.cache.CopyOut(b.spec.Bundle(), slices.Collect(maps.Values(records)))
}

func (b *Run) Card(name string, c model.ShareCard) error {
	text, _ := json.Marshal(c)
	key := journal.Hash(fmt.Appendf(nil, "%d\x00%s\x00%s", version("card"), b.draw.FontID(), text))
	f := picture.FormatPNG
	_, err := b.single(journal.Record{Name: name, Kind: journal.KindPicture, Key: key, Picture: journal.Picture{Format: f}}, f.Ext, func() ([]byte, error) {
		img, err := card.Draw(c, b.draw)
		if err != nil {
			return nil, fmt.Errorf("quire: %s: %w", name, err)
		}
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		return buf.Bytes(), err
	})
	return err
}
