package model

import (
	"fmt"
	"strconv"

	"github.com/lahickey/quire/internal/journal"
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
	Root   int
	Col    int
	Widest Width
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

func (s Scale) cols(slot Slot) int {
	switch slot {
	case SlotPageWide:
		return wideCols
	case SlotPageFull, SlotBleed:
		return fullCols
	}
	return pageCols
}

func (s Scale) top(slot Slot) int {
	if slot == SlotThumb {
		return 3 * s.Thumb()
	}
	if slot == SlotBleed {
		return journal.MasterCap
	}
	return min(2*s.inner(s.cols(slot)), journal.MasterCap)
}

func (s Scale) sizes(slot Slot) string {
	switch slot {
	case SlotThumb:
		return s.Rem(s.Thumb())
	case SlotBleed:
		return "100vw"
	case SlotColumn:
		return fmt.Sprintf("(min-width: %s) %s, %s",
			s.Rem(s.Measure()), s.Rem(s.Col), s.narrowClause())
	}
	n := s.cols(slot)
	return fmt.Sprintf("(min-width: %s) %s, %s",
		s.Rem(s.pageWidth(n)), s.Rem(s.inner(n)), s.narrowClause())
}

func (s Scale) display(slot Slot) int {
	switch slot {
	case SlotThumb:
		return s.Thumb()
	case SlotColumn:
		return s.Col
	case SlotBleed:
		return journal.MasterCap / 2
	}
	return s.inner(s.cols(slot))
}

func (s Scale) bottom(slot Slot) int {
	if slot == SlotThumb {
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

func (s Scale) Slots() map[Slot]SlotSpec {
	out := map[Slot]SlotSpec{}
	for slot := SlotThumb; slot <= SlotBleed; slot++ {
		out[slot] = SlotSpec{Width: journal.Snap(journal.Ladder(), 2*s.display(slot)), Sizes: s.sizes(slot)}
	}
	return out
}
