// Package measure reads how wide a font's characters are, for a font set's description.
package measure

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lahickey/quire/model"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Widths is the advance of every character a TTF has in the unicode ranges given, each written as a browser's unicode-range is.
func Widths(ttf []byte, ranges ...string) (model.FontWidths, error) {
	f, err := sfnt.Parse(ttf)
	if err != nil {
		return model.FontWidths{}, err
	}
	var buf sfnt.Buffer
	em := f.UnitsPerEm()
	w := model.FontWidths{Em: int(em), Of: map[rune]int{}}
	count := map[int]int{}
	for part := range strings.SplitSeq(strings.Join(ranges, ","), ",") {
		lo, hi, isRange := strings.Cut(strings.TrimPrefix(strings.TrimSpace(part), "U+"), "-")
		if !isRange {
			hi = lo
		}
		from, err1 := strconv.ParseInt(lo, 16, 32)
		to, err2 := strconv.ParseInt(hi, 16, 32)
		if err1 != nil || err2 != nil {
			return w, fmt.Errorf("unicode range %q", part)
		}
		for r := rune(from); r <= rune(to); r++ {
			i, err := f.GlyphIndex(&buf, r)
			if err != nil || i == 0 {
				continue
			}
			a, err := f.GlyphAdvance(&buf, i, fixed.Int26_6(em)<<6, font.HintingNone)
			if err != nil {
				return w, err
			}
			w.Of[r] = a.Round()
			count[a.Round()]++
		}
	}
	for a, n := range count {
		if n > count[w.Default] || n == count[w.Default] && a < w.Default {
			w.Default = a
		}
	}
	for r, a := range w.Of {
		if a == w.Default {
			delete(w.Of, r)
		}
	}
	if len(count) == 0 {
		return w, fmt.Errorf("no character in %v", ranges)
	}
	return w, nil
}
