package model

import (
	"fmt"
	"strconv"
)

const (
	pageCols = 2
	wideCols = 3
	fullCols = 4
)

type Width int

const (
	WidthNarrow Width = iota + 1
	WidthWide
	WidthFull
)

const narrowest = 320

var Widths = []Width{WidthNarrow, WidthWide, WidthFull}

type Scale struct {
	Root int
	Col  int
}

func (s Scale) inner(cols int) int {
	if cols < 1 {
		return 0
	}
	return cols*s.Col + (cols-1)*s.Gutter()
}

func (s Scale) pageWidth(cols int) int {
	return s.inner(cols) + 2*s.Gutter()
}

func (s Scale) narrowClause() string {
	return "calc(100vw - " + s.Rem(2*s.Gutter()) + ")"
}

func (s Scale) cols(size Size) int {
	switch size {
	case SizePageWide:
		return wideCols
	case SizePageFull, SizeBleed:
		return fullCols
	}
	return pageCols
}

func (s Scale) top(size Size) int {
	if size == SizeThumb {
		return 3 * s.Thumb()
	}
	if size == SizeBleed {
		return MasterCap
	}
	return min(2*s.inner(s.cols(size)), MasterCap)
}

func (s Scale) sizes(size Size) string {
	switch size {
	case SizeThumb:
		return s.Rem(s.Thumb())
	case SizeBleed:
		return "100vw"
	case SizeColumn:
		return fmt.Sprintf("(min-width: %s) %s, %s",
			s.Rem(s.Measure()), s.Rem(s.Col), s.narrowClause())
	}
	n := s.cols(size)
	return fmt.Sprintf("(min-width: %s) %s, %s",
		s.Rem(s.pageWidth(n)), s.Rem(s.inner(n)), s.narrowClause())
}

func (s Scale) display(size Size) int {
	switch size {
	case SizeThumb:
		return s.Thumb()
	case SizeColumn:
		return s.Col
	case SizeBleed:
		return MasterCap / 2
	}
	return s.inner(s.cols(size))
}

func (s Scale) bottom(size Size) int {
	if size == SizeThumb {
		return s.Thumb()
	}
	return narrowest - 2*s.Gutter()
}

func DefaultScale() Scale {
	return Scale{Root: 16, Col: 336}
}

func (s Scale) Rem(px int) string {
	return strconv.FormatFloat(float64(px)/float64(s.Root), 'f', -1, 64) + "rem"
}

func (s Scale) Gutter() int { return s.Root * 3 / 2 }

func (s Scale) Thumb() int { return s.Root * 4 }

func (s Scale) Measure() int { return s.pageWidth(pageCols) }

func (s Scale) Wide() int { return s.pageWidth(wideCols) }

func (s Scale) Full() int { return s.pageWidth(fullCols) }
