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

type Image struct {
	Name   string
	Hash   string
	Width  int
	Height int
	Alpha  bool
	files  map[int]string
	refs   []imgRef
	rungs  []int
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
	return ImageURL(im.files[w])
}

func (im Image) ref(slot Slot) (imgRef, bool) {
	for _, r := range im.refs {
		if r.slot == slot {
			return r, true
		}
	}
	return imgRef{}, false
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

func ImageURL(file string) string { return ImagePath + file }

func NewImage(rec journal.Record, sc Scale) Image {
	base := path.Base(filepath.ToSlash(rec.Name))
	stem := strings.TrimSuffix(base, path.Ext(base))
	m := rec.Prep
	var rungs []int
	files := map[int]string{}
	for _, v := range rec.Formats[len(rec.Formats)-1].Out {
		files[v.W] = v.Name
		rungs = append(rungs, v.W)
	}
	slices.Sort(rungs)
	slots := sc.Slots()
	refs := make([]imgRef, 0, len(slots))
	for s := SlotThumb; s <= SlotBleed; s++ {
		spec := slots[s]
		refs = append(refs, imgRef{
			slot:  s,
			w:     journal.Snap(rungs, spec.Width),
			bot:   min(sc.bottom(s), spec.Width),
			top:   max(sc.top(s), spec.Width),
			sizes: spec.Sizes,
		})
	}
	return Image{
		Name:   stem,
		Hash:   m.Hash,
		Width:  m.W,
		Height: m.H,
		Alpha:  m.Alpha,
		files:  files,
		refs:   refs,
		rungs:  rungs,
	}
}

func (im Image) URL(slot Slot) string {
	r, ok := im.ref(slot)
	if !ok {
		return ""
	}
	name, ok := im.files[r.w]
	if !ok {
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

func (im Image) SrcSet(slot Slot) string {
	r, ok := im.ref(slot)
	if !ok || len(im.rungs) == 0 {
		return ""
	}
	var parts []string
	for _, w := range window(im.rungs, r.bot, r.top) {
		parts = append(parts, fmt.Sprintf("%s %dw", ImageURL(im.files[w]), w))
	}
	return strings.Join(parts, ", ")
}

func (im Image) Sizes(slot Slot) string {
	if r, ok := im.ref(slot); ok {
		return r.sizes
	}
	return "100vw"
}

func (im Image) Attrs(slot Slot) templ.Attributes {
	return im.attrs(slot, false)
}

func (im Image) PriorityAttrs(slot Slot) templ.Attributes {
	return im.attrs(slot, true)
}
