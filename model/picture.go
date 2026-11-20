package model

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

const BundlePath = "/_quire/img/"

type Size int

const (
	SizeThumb Size = iota + 1
	SizeColumn
	SizePage
	SizePageWide
	SizePageFull
	SizeBleed
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
		if r <= ShareWidth {
			w = r
		}
	}
	return BundleURL(p.Files[w])
}

func (s Size) String() string {
	switch s {
	case SizeThumb:
		return "thumb"
	case SizeColumn:
		return "column"
	case SizePage:
		return "page"
	case SizePageWide:
		return "page-wide"
	case SizePageFull:
		return "page-full"
	case SizeBleed:
		return "bleed"
	}
	return "size(" + strconv.Itoa(int(s)) + ")"
}

func BundleURL(file string) string { return BundlePath + file }

func (p Picture) URL(size Size) string {
	name, ok := p.Files[snap(p.rungs(), 2*p.Scale.display(size))]
	if !ok {
		return ""
	}
	return BundleURL(name)
}

func (p Picture) SrcSet(size Size) string {
	rungs := p.rungs()
	if len(rungs) == 0 {
		return ""
	}
	w := 2 * p.Scale.display(size)
	var parts []string
	for _, r := range window(rungs, min(p.Scale.bottom(size), w), max(p.Scale.top(size), w)) {
		parts = append(parts, fmt.Sprintf("%s %dw", BundleURL(p.Files[r]), r))
	}
	return strings.Join(parts, ", ")
}

func (p Picture) Sizes(size Size) string { return p.Scale.sizes(size) }

func (p Picture) Attrs(size Size, priority bool) templ.Attributes {
	a := templ.Attributes{
		"src":      p.URL(size),
		"srcset":   p.SrcSet(size),
		"sizes":    p.Sizes(size),
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
