package model

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/lahickey/quire/internal/radix"
)

var (
	PaletteEmber = Palette{Name: "Ember", Neutral: "sand", Accent: "orange", Spot: "cyan", Tone: [6]string{"brown", "indigo", "teal", "purple", "blue", "pink"}, Error: "tomato", OK: "grass", Warn: "amber"}
	PaletteBrass = Palette{Name: "Brass", Neutral: "sand", Accent: "amber", Spot: "indigo", Tone: [6]string{"lime", "plum", "teal", "pink", "sky", "brown"}, Error: "ruby", OK: "grass", Warn: "orange"}
	PaletteMoss  = Palette{Name: "Moss", Neutral: "sage", Accent: "grass", Spot: "violet", Tone: [6]string{"cyan", "orange", "blue", "brown", "pink", "lime"}, Error: "red", OK: "teal", Warn: "amber"}
	PaletteTide  = Palette{Name: "Tide", Neutral: "slate", Accent: "cyan", Spot: "orange", Tone: [6]string{"indigo", "brown", "purple", "lime", "pink", "teal"}, Error: "red", OK: "grass", Warn: "amber"}
	PaletteInk   = Palette{Name: "Ink", Neutral: "slate", Accent: "indigo", Spot: "crimson", Tone: [6]string{"violet", "teal", "orange", "cyan", "gold", "blue"}, Error: "tomato", OK: "grass", Warn: "amber"}
	PaletteDusk  = Palette{Name: "Dusk", Neutral: "mauve", Accent: "purple", Spot: "orange", Tone: [6]string{"pink", "sky", "brown", "blue", "lime", "indigo"}, Error: "red", OK: "jade", Warn: "amber"}
	PaletteBloom = Palette{Name: "Bloom", Neutral: "mauve", Accent: "pink", Spot: "iris", Tone: [6]string{"orange", "cyan", "brown", "blue", "lime", "plum"}, Error: "tomato", OK: "jade", Warn: "amber"}
	PaletteWine  = Palette{Name: "Wine", Neutral: "mauve", Accent: "ruby", Spot: "indigo", Tone: [6]string{"orange", "sky", "gold", "violet", "teal", "plum"}, Error: "tomato", OK: "grass", Warn: "amber"}
	PaletteAsh   = Palette{Name: "Ash", Neutral: "gray", Accent: "blue", Spot: "plum", Tone: [6]string{"violet", "brown", "pink", "teal", "orange", "cyan"}, Error: "red", OK: "grass", Warn: "amber"}

	Palettes = []Palette{PaletteEmber, PaletteBrass, PaletteMoss, PaletteTide, PaletteInk, PaletteDusk, PaletteBloom, PaletteWine, PaletteAsh}
)

type Palette struct {
	Name    string
	Neutral string
	Accent  string
	Spot    string
	Tone    [6]string

	// OK, Warn and Error are for what something means, not for telling things apart.
	OK, Warn, Error string
}

// Role is a part a palette's colours play. A range has all twelve steps of
// its hue; any other has the Steps, by name.
type Role struct {
	Name  string
	Hue   string
	Range bool
}

// Step is one of the steps every role has, by the job it does, and which of
// a hue's twelve it is.
type Step struct {
	Name string
	N    int
}

var Steps = []Step{{"tint", 3}, {"line", 7}, {"solid", 9}, {"text", 11}}

func (p Palette) ID() string { return strings.ToLower(p.Name) }

func (p Palette) Roles() []Role {
	rs := []Role{{"neutral", p.Neutral, true}, {"accent", p.Accent, true}, {Name: "spot", Hue: p.Spot}}
	for i, h := range p.Tone {
		rs = append(rs, Role{Name: fmt.Sprintf("tone%d", i+1), Hue: h})
	}
	return append(rs, Role{Name: "ok", Hue: p.OK}, Role{Name: "warn", Hue: p.Warn}, Role{Name: "error", Hue: p.Error})
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

// PickerPalettes is what a picker offers: the palettes asked for, led by the site's own unless it is among them.
func PickerPalettes(site Palette, asked []Palette) []Palette {
	if !slices.Contains(asked, site) {
		asked = slices.Concat([]Palette{site}, asked)
	}
	ids := map[string]bool{}
	for _, p := range asked {
		p.Check()
		if p.Name == "" || ids[p.ID()] {
			panic("quire: every palette in the picker needs a name of its own")
		}
		ids[p.ID()] = true
	}
	return asked
}
