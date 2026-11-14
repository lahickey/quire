package model

import (
	"fmt"
	"maps"
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

type Image struct {
	Name   string
	Hash   string
	Width  int
	Height int
	Alpha  bool
	Files  map[int]string
	Scale  Scale
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

func (im Image) rungs() []int { return slices.Sorted(maps.Keys(im.Files)) }

func (im Image) shareURL() string {
	rungs := im.rungs()
	if len(rungs) == 0 {
		return ""
	}
	w := rungs[0]
	for _, r := range rungs {
		if r <= journal.ShareWidth {
			w = r
		}
	}
	return ImageURL(im.Files[w])
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

func (im Image) URL(slot Slot) string {
	name, ok := im.Files[journal.Snap(im.rungs(), 2*im.Scale.display(slot))]
	if !ok {
		return ""
	}
	return ImageURL(name)
}

func (im Image) SrcSet(slot Slot) string {
	rungs := im.rungs()
	if len(rungs) == 0 {
		return ""
	}
	w := 2 * im.Scale.display(slot)
	var parts []string
	for _, r := range window(rungs, min(im.Scale.bottom(slot), w), max(im.Scale.top(slot), w)) {
		parts = append(parts, fmt.Sprintf("%s %dw", ImageURL(im.Files[r]), r))
	}
	return strings.Join(parts, ", ")
}

func (im Image) Sizes(slot Slot) string { return im.Scale.sizes(slot) }

func (im Image) Attrs(slot Slot, priority bool) templ.Attributes {
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
