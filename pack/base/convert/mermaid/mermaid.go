// Package mermaid draws a mermaid diagram as SVG, measured in the site's font set.
package mermaid

import (
	"encoding/json/v2"
	"fmt"
	"strings"

	"github.com/lahickey/quire/bake"
	"github.com/lahickey/quire/model"
	"tuios.dev/sketch"
)

const (
	Version = 2

	styleBold = 1
	styleMono = 4
	styleSub  = 8
	styleSup  = 16
)

type measurer model.FontSet

type Converter struct{}

func (m measurer) Width(s string, size float64, style int) float64 {
	weight := 400
	if style&styleBold != 0 {
		weight = 700
	}
	if style&(styleSub|styleSup) != 0 {
		size /= 1.2
	}
	if style&styleMono != 0 {
		return m.Mono.Width(weight, s, size)
	}
	return m.Text.Width(weight, s, size)
}

func (Converter) Name() string { return "mermaid" }

func (Converter) Version() int { return Version }

// Convert takes the site's font set as JSON, as its one argument, and lays the text out by the widths in it.
func (Converter) Convert(in bake.Input) (bake.Output, error) {
	src := string(in.Body)
	first, _, _ := strings.Cut(src, "\n")
	var set model.FontSet
	if len(in.Args) != 1 || json.Unmarshal([]byte(in.Args[0]), &set) != nil {
		return bake.Output{}, fmt.Errorf("%q: asked for with no font set", first)
	}
	if set.Text.Widths[400].Em == 0 || set.Mono.Widths[400].Em == 0 {
		return bake.Output{}, fmt.Errorf("%q: the font set has no widths at weight 400: quire fonts writes them", first)
	}
	out, err := sketch.Render(src, sketch.RenderOptions{
		Measurer:       measurer(set),
		FontFamily:     set.Text.Name,
		FontFamilyMono: set.Mono.Name,

		IDPrefix: new("d" + in.ID[len(in.ID)-8:]),
	})
	if err != nil {
		return bake.Output{}, fmt.Errorf("%q: %w", first, err)
	}
	return bake.Output{Body: []byte(out.Data), Ext: "svg"}, nil
}
