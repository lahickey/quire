package model

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/bundle"
)

const BundlePath = "/_quire/img/"

type Slot int

const (
	SlotThumb Slot = iota + 1
	SlotColumn
	SlotPage
	SlotPageWide
	SlotPageFull
	SlotBleed
)

type Picture struct {
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

func (p Picture) rungs() []int { return slices.Sorted(maps.Keys(p.Files)) }

func (p Picture) shareURL() string {
	rungs := p.rungs()
	if len(rungs) == 0 {
		return ""
	}
	w := rungs[0]
	for _, r := range rungs {
		if r <= bundle.ShareWidth {
			w = r
		}
	}
	return BundleURL(p.Files[w])
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

func BundleURL(file string) string { return BundlePath + file }

func (p Picture) URL(slot Slot) string {
	name, ok := p.Files[bundle.Snap(p.rungs(), 2*p.Scale.display(slot))]
	if !ok {
		return ""
	}
	return BundleURL(name)
}

func (p Picture) SrcSet(slot Slot) string {
	rungs := p.rungs()
	if len(rungs) == 0 {
		return ""
	}
	w := 2 * p.Scale.display(slot)
	var parts []string
	for _, r := range window(rungs, min(p.Scale.bottom(slot), w), max(p.Scale.top(slot), w)) {
		parts = append(parts, fmt.Sprintf("%s %dw", BundleURL(p.Files[r]), r))
	}
	return strings.Join(parts, ", ")
}

func (p Picture) Sizes(slot Slot) string { return p.Scale.sizes(slot) }

func (p Picture) Attrs(slot Slot, priority bool) templ.Attributes {
	a := templ.Attributes{
		"src":      p.URL(slot),
		"srcset":   p.SrcSet(slot),
		"sizes":    p.Sizes(slot),
		"width":    strconv.Itoa(p.Width),
		"height":   strconv.Itoa(p.Height),
		"decoding": "async",
	}
	if priority {
		a["fetchpriority"] = "high"
	} else {
		a["loading"] = "lazy"
	}
	return a
}
