package model

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/lahickey/quire/internal/radix"
)

type Palette struct {
	Neutral string
	Accent  string
	Spot    string
	Swatch  [4]string

	Red, Green, Amber string
}

var (
	PaletteEmber = Palette{Neutral: "sand", Accent: "orange", Spot: "yellow", Swatch: [4]string{"orange", "tomato", "amber", "brown"}, Red: "tomato", Green: "grass", Amber: "amber"}
	PaletteMoss  = Palette{Neutral: "sage", Accent: "grass", Spot: "lime", Swatch: [4]string{"grass", "teal", "sky", "amber"}, Red: "red", Green: "green", Amber: "amber"}
	PaletteInk   = Palette{Neutral: "slate", Accent: "indigo", Spot: "crimson", Swatch: [4]string{"blue", "violet", "cyan", "amber"}, Red: "red", Green: "grass", Amber: "amber"}
	PaletteDusk  = Palette{Neutral: "mauve", Accent: "purple", Spot: "amber", Swatch: [4]string{"purple", "ruby", "sky", "pink"}, Red: "red", Green: "jade", Amber: "amber"}
)

type Role struct {
	Name string
	Hue  string
}

func (p Palette) Roles() []Role {
	rs := []Role{{"neutral", p.Neutral}, {"accent", p.Accent}, {"spot", p.Spot}}
	for i, h := range p.Swatch {
		rs = append(rs, Role{fmt.Sprintf("swatch%d", i+1), h})
	}
	return append(rs, Role{"red", p.Red}, Role{"green", p.Green}, Role{"amber", p.Amber})
}

func (p Palette) Check() {
	for _, r := range p.Roles() {
		if _, ok := radix.Scales[r.Hue]; !ok {
			panic(fmt.Sprintf("quire: palette %s is %q, not a Radix hue: %s",
				r.Name, r.Hue, strings.Join(slices.Sorted(maps.Keys(radix.Scales)), " ")))
		}
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
