package model

import (
	"fmt"
	"math"
	"testing"

	"github.com/lahickey/quire/internal/radix"
)

func apca(t *testing.T, text, bg string) float64 {
	t.Helper()
	y := func(hex string) float64 {
		var r, g, b uint8
		if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil {
			t.Fatalf("%s: %v", hex, err)
		}
		c := func(v uint8) float64 { return math.Pow(float64(v)/255, 2.4) }
		l := 0.2126729*c(r) + 0.7151522*c(g) + 0.0721750*c(b)
		if l < 0.022 {
			l += math.Pow(0.022-l, 1.414)
		}
		return l
	}
	yt, yb := y(text), y(bg)
	if yb > yt {
		return ((math.Pow(yb, 0.56)-math.Pow(yt, 0.57))*1.14 - 0.027) * 100
	}
	return ((math.Pow(yb, 0.65)-math.Pow(yt, 0.62))*1.14 + 0.027) * 100
}

func TestPalettes(t *testing.T) {
	palettes := map[string]Palette{"ember": PaletteEmber, "moss": PaletteMoss, "ink": PaletteInk, "dusk": PaletteDusk}
	accents := map[string]string{}
	for name, p := range palettes {
		p.Check()
		for _, mode := range []string{"dark", "light"} {
			steps := func(hue string) [12]string {
				if mode == "dark" {
					return radix.Scales[hue].Dark
				}
				return radix.Scales[hue].Light
			}
			n := steps(p.Neutral)
			for _, r := range p.Roles() {
				text := steps(r.Hue)[10]
				for _, bg := range []string{n[0], n[1]} {
					// APCA Lc 60 is the floor for body text; 59.5 allows for hex rounding.
					if lc := math.Abs(apca(t, text, bg)); lc < 59.5 {
						t.Errorf("%s %s: %s-11 %s on %s is Lc %.1f, under 60", name, mode, r.Name, text, bg, lc)
					}
				}
			}
		}
		if prev, ok := accents[p.Accent]; ok {
			t.Errorf("%s and %s share an accent", prev, name)
		}
		accents[p.Accent] = name
	}
	defer func() {
		if recover() == nil {
			t.Error("a misspelt hue loaded")
		}
	}()
	Palette{Neutral: "slte"}.Check()
}

func TestTags(t *testing.T) {
	tags := Tags{"lang": {"go", "c"}, "gfx": {" Vulkan "}}
	doc := func(tags ...string) []Document[Meta] { return []Document[Meta]{{Slug: "a", Meta: Meta{Tags: tags}}} }
	col, err := Collect(Basic{ID: "x", Tags: tags}, doc("Go", "vulkan"), Site{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := col.Tags(col.Docs()[0]); got[0].Kind != "lang" || got[1].Kind != "gfx" {
		t.Errorf("kinds %v", got)
	}
	if _, err := Collect(Basic{ID: "x", Tags: tags}, doc("golang"), Site{}, false); err == nil {
		t.Error("a tag outside the vocabulary loaded")
	}
}
