package model

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/journal"
)

const ImagePath = "/_quire/img/"

type Slot int

const (
	SlotThumb Slot = iota + 1
	SlotColumn
	SlotPage
	SlotPageWide
	SlotPageFull
	SlotBleed
)

const SlotDefault = SlotPage

func (s Slot) String() string {
	switch s {
	case SlotThumb:
		return "thumb"
	case SlotColumn:
		return "column"
	case SlotPage:
		return "page"
	case SlotPageWide:
		return "page-wide"
	case SlotPageFull:
		return "page-full"
	case SlotBleed:
		return "bleed"
	}
	return "slot(" + strconv.Itoa(int(s)) + ")"
}

func upTo(max Slot) []Slot {
	if max < SlotThumb {
		max = SlotDefault
	}
	out := make([]Slot, 0, int(max))
	for s := SlotThumb; s <= max; s++ {
		out = append(out, s)
	}
	return out
}

type SlotSpec struct {
	Width int
	Sizes string
}

type imgRef struct {
	slot     Slot
	w        int
	bot, top int
	sizes    string
}

func window(rungs []int, bot, top int) []int {
	lo, hi := 0, len(rungs)-1
	for i, w := range rungs {
		if w <= bot {
			lo = i
		}
		if w >= top {
			hi = i
			break
		}
	}
	return rungs[min(lo, hi) : hi+1]
}

type imgKey struct {
	ext string
	w   int
}

type Image struct {
	Name    string
	Hash    string
	Width   int
	Height  int
	Alpha   bool
	formats []journal.Format
	files   map[imgKey]string
	refs    []imgRef
	rungs   []int
}

func imgStem(name string) string {
	s := path.Base(filepath.ToSlash(name))
	return strings.TrimSuffix(s, path.Ext(s))
}

func ImageURL(file string) string { return ImagePath + file }

func rungsOf(rec journal.Record) []int {
	seen := map[int]bool{}
	var rungs []int
	for _, vs := range rec.Formats {
		for _, v := range vs.Out {
			if seen[v.W] {
				continue
			}
			seen[v.W] = true
			rungs = append(rungs, v.W)
		}
	}
	slices.Sort(rungs)
	return rungs
}

func NewImage(rec journal.Record, sc Scale, ceiling Slot, slots map[Slot]SlotSpec) Image {
	stem := imgStem(rec.Name)
	m := rec.Prep
	rungs := rungsOf(rec)
	formats := make([]journal.Format, len(rec.Formats))
	files := map[imgKey]string{}
	for i, vs := range rec.Formats {
		formats[i] = vs.Format()
		for _, v := range vs.Out {
			files[imgKey{vs.Ext, v.W}] = v.Name
		}
	}
	used := upTo(ceiling)
	refs := make([]imgRef, 0, len(used))
	for _, s := range used {
		spec, ok := slots[s]
		if !ok {
			continue
		}
		refs = append(refs, imgRef{
			slot:  s,
			w:     journal.Snap(rungs, spec.Width),
			bot:   min(sc.bottom(s), spec.Width),
			top:   max(sc.top(s), spec.Width),
			sizes: spec.Sizes,
		})
	}
	return Image{
		Name:    stem,
		Hash:    m.Hash,
		Width:   m.W,
		Height:  m.H,
		Alpha:   m.Alpha,
		formats: formats,
		files:   files,
		refs:    refs,
		rungs:   rungs,
	}
}

func (im Image) file(ext string, w int) string { return im.files[imgKey{ext, w}] }

func (im Image) last() journal.Format { return im.formats[len(im.formats)-1] }

func (im Image) ext() string { return im.last().Ext }

func (im Image) shareURL() string {
	if len(im.rungs) == 0 {
		return ""
	}
	w := im.rungs[0]
	for _, r := range im.rungs {
		if r <= journal.ShareWidth {
			w = r
		}
	}
	if f := im.file(im.ext(), w); f != "" {
		return ImageURL(f)
	}
	return ""
}

func (im Image) ref(slot Slot) (imgRef, bool) {
	for _, r := range im.refs {
		if r.slot == slot {
			return r, true
		}
	}
	return imgRef{}, false
}

func (im Image) URL(slot Slot) string {
	r, ok := im.ref(slot)
	if !ok {
		return ""
	}
	name := im.file(im.last().Ext, r.w)
	if name == "" {
		return ""
	}
	return ImageURL(name)
}

func (im Image) Src() string {
	if len(im.refs) == 0 {
		return ""
	}
	return im.URL(im.refs[len(im.refs)-1].slot)
}

func (im Image) srcSet(slot Slot, f journal.Format) string {
	r, ok := im.ref(slot)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, w := range window(im.rungs, r.bot, r.top) {
		name := im.file(f.Ext, w)
		if name == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s %dw", ImageURL(name), w)
	}
	return b.String()
}

func (im Image) SrcSet(slot Slot) string { return im.srcSet(slot, im.last()) }

func (im Image) Sizes(slot Slot) string {
	if r, ok := im.ref(slot); ok {
		return r.sizes
	}
	return "100vw"
}

func (im Image) Sources(slot Slot) []templ.Attributes {
	out := make([]templ.Attributes, 0, len(im.formats))
	for _, f := range im.formats[:max(0, len(im.formats)-1)] {
		out = append(out, templ.Attributes{
			"type":   f.Type,
			"srcset": im.srcSet(slot, f),
			"sizes":  im.Sizes(slot),
		})
	}
	return out
}

func (im Image) attrs(slot Slot, priority bool) templ.Attributes {
	a := templ.Attributes{
		"src":      im.URL(slot),
		"srcset":   im.SrcSet(slot),
		"sizes":    im.Sizes(slot),
		"width":    strconv.Itoa(im.Width),
		"height":   strconv.Itoa(im.Height),
		"decoding": "async",
	}
	if priority {
		a["fetchpriority"] = "high"
	} else {
		a["loading"] = "lazy"
	}
	return a
}

func (im Image) Attrs(slot Slot) templ.Attributes {
	return im.attrs(slot, false)
}

func (im Image) PriorityAttrs(slot Slot) templ.Attributes {
	return im.attrs(slot, true)
}
