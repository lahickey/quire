// Package mermaid draws a mermaid diagram as SVG, measured in the site's font set.
package mermaid

import (
	"fmt"
	"strings"

	"github.com/lahickey/quire/bake"
	"github.com/lahickey/quire/internal/fonts/measure"
	"tuios.dev/sketch"
)

const (
	Version = 1

	styleBold = 1
	styleMono = 4
	styleSub  = 8
	styleSup  = 16
)

type measurer struct{ faces *measure.Faces }

type Converter struct{ faces *measure.Faces }

func (m measurer) Width(s string, size float64, style int) float64 {
	weight := 400
	if style&styleBold != 0 {
		weight = 700
	}
	if style&(styleSub|styleSup) != 0 {
		size /= 1.2
	}
	return measure.Width(m.faces.Face(style&styleMono != 0, weight), s, size)
}

func New(faces *measure.Faces) Converter { return Converter{faces} }

func (Converter) Name() string { return "mermaid" }

func (Converter) Version() int { return Version }

// Convert takes the font set's id as its one argument, so a site on other fonts is refused and not drawn wrong.
func (c Converter) Convert(in bake.Input) (bake.Output, error) {
	src := string(in.Body)
	first, _, _ := strings.Cut(src, "\n")
	set := c.faces.Set
	if len(in.Args) != 1 || in.Args[0] != set.ID {
		return bake.Output{}, fmt.Errorf("quire: mermaid %q: asked for in fonts %q, and the bake has %s", first, in.Args, set.ID)
	}
	out, err := sketch.Render(src, sketch.RenderOptions{
		Measurer:       measurer(c),
		FontFamily:     set.Text.Name,
		FontFamilyMono: set.Mono.Name,

		IDPrefix: new("d" + in.ID[len(in.ID)-8:]),
	})
	if err != nil {
		return bake.Output{}, fmt.Errorf("quire: mermaid %q: %w", first, err)
	}
	return bake.Output{Body: []byte(out.Data), Ext: "svg"}, nil
}
