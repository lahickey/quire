package model

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/lahickey/quire/internal/radix"
)

var (
	PaletteEmber = Palette{Name: "Ember", Neutral: "sand", Accent: "orange", Spot: "cyan", Swatch: [4]string{"brown", "plum", "indigo", "yellow"}, Red: "tomato", Green: "grass", Amber: "amber"}
	PaletteBrass = Palette{Name: "Brass", Neutral: "sand", Accent: "gold", Spot: "indigo", Swatch: [4]string{"plum", "crimson", "teal", "orange"}, Red: "tomato", Green: "grass", Amber: "amber"}
	PaletteMoss  = Palette{Name: "Moss", Neutral: "sage", Accent: "grass", Spot: "violet", Swatch: [4]string{"lime", "brown", "blue", "orange"}, Red: "red", Green: "teal", Amber: "amber"}
	PaletteTide  = Palette{Name: "Tide", Neutral: "slate", Accent: "cyan", Spot: "orange", Swatch: [4]string{"indigo", "pink", "brown", "yellow"}, Red: "red", Green: "grass", Amber: "amber"}
	PaletteInk   = Palette{Name: "Ink", Neutral: "slate", Accent: "indigo", Spot: "crimson", Swatch: [4]string{"blue", "violet", "cyan", "yellow"}, Red: "tomato", Green: "grass", Amber: "amber"}
	PaletteDusk  = Palette{Name: "Dusk", Neutral: "mauve", Accent: "purple", Spot: "orange", Swatch: [4]string{"indigo", "pink", "sky", "yellow"}, Red: "red", Green: "jade", Amber: "amber"}
	PaletteBloom = Palette{Name: "Bloom", Neutral: "mauve", Accent: "pink", Spot: "iris", Swatch: [4]string{"violet", "cyan", "brown", "lime"}, Red: "tomato", Green: "jade", Amber: "amber"}
	PaletteWine  = Palette{Name: "Wine", Neutral: "mauve", Accent: "ruby", Spot: "indigo", Swatch: [4]string{"plum", "violet", "teal", "gold"}, Red: "tomato", Green: "grass", Amber: "amber"}
	PaletteAsh   = Palette{Name: "Ash", Neutral: "gray", Accent: "blue", Spot: "plum", Swatch: [4]string{"cyan", "pink", "violet", "orange"}, Red: "red", Green: "grass", Amber: "amber"}

	Palettes = []Palette{PaletteEmber, PaletteBrass, PaletteMoss, PaletteTide, PaletteInk, PaletteDusk, PaletteBloom, PaletteWine, PaletteAsh}
)

type Palette struct {
	Name    string
	Neutral string
	Accent  string
	Spot    string
	Swatch  [4]string

	Red, Green, Amber string
}

type Role struct {
	Name string
	Hue  string
}

func (p Palette) ID() string { return strings.ToLower(p.Name) }

func (p Palette) Roles() []Role {
	rs := []Role{{"neutral", p.Neutral}, {"accent", p.Accent}, {"spot", p.Spot}}
	for i, h := range p.Swatch {
		rs = append(rs, Role{fmt.Sprintf("swatch%d", i+1), h})
	}
	return append(rs, Role{"red", p.Red}, Role{"green", p.Green}, Role{"amber", p.Amber})
}

func (p Palette) Check() {
	roles := map[string]string{}
	for _, r := range p.Roles() {
		if _, ok := radix.Scales[r.Hue]; !ok {
			panic(fmt.Sprintf("quire: palette %s is %q, not a Radix hue: %s",
				r.Name, r.Hue, strings.Join(slices.Sorted(maps.Keys(radix.Scales)), " ")))
		}
		if other, taken := roles[r.Hue]; taken {
			panic(fmt.Sprintf("quire: palette %s and %s are both %q; each role needs a hue of its own", other, r.Name, r.Hue))
		}
		roles[r.Hue] = r.Name
	}
}

func (p Palette) Ink() Ink {
	n := radix.Scales[p.Neutral].Dark
	return Ink{
		Bg:        n[0],
		Text:      n[11],
		Secondary: n[10],
		Muted:     n[9],
		Accent:    radix.Scales[p.Accent].Dark[10],
		Spot:      radix.Scales[p.Spot].Dark[10],
	}
}
