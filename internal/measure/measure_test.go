package measure

import (
	"os"
	"testing"

	"github.com/lahickey/quire/model"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

func TestWidthsAreTheFaces(t *testing.T) {
	ttf, err := os.ReadFile("../../pack/base/convert/card/fonts/inter-400.ttf")
	if err != nil {
		t.Fatal(err)
	}
	w, err := Widths(ttf, "U+0020-007E, U+2014", "U+00E9")
	if err != nil {
		t.Fatal(err)
	}
	f, _ := sfnt.Parse(ttf)
	var buf sfnt.Buffer
	const s, size = "Wide — café, il.", 100
	var want fixed.Int26_6
	for _, r := range s {
		i, _ := f.GlyphIndex(&buf, r)
		a, _ := f.GlyphAdvance(&buf, i, fixed.I(size), font.HintingNone)
		want += a
	}
	got := model.FontFamily{Widths: map[int]model.FontWidths{400: w}}.Width(700, s, size)
	if d := got - float64(want)/64; d > 0.1 || d < -0.1 {
		t.Errorf("%q is %.2f wide by its widths and %.2f by the face", s, got, float64(want)/64)
	}
}
