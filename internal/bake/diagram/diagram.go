package diagram

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/lahickey/quire/internal/fonts"
	"github.com/lahickey/quire/internal/journal"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"tuios.dev/sketch"
)

const (
	styleBold = 1
	styleMono = 4
	styleSub  = 8
	styleSup  = 16
)

//go:embed fonts
var defaultFonts embed.FS

type measurer struct{ text, mono map[int]*sfnt.Font }

type Renderer struct {
	set  fonts.FontSet
	meas measurer
}

func parseFaces(measure fs.FS, files map[int]string) (map[int]*sfnt.Font, error) {
	out := map[int]*sfnt.Font{}
	for weight, file := range files {
		b, err := fs.ReadFile(measure, file)
		if err != nil {
			return nil, err
		}
		f, err := sfnt.Parse(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		out[weight] = f
	}
	if out[400] == nil {
		return nil, fmt.Errorf("no 400 weight to measure with")
	}
	return out, nil
}

func newRenderer(measure fs.FS) (*Renderer, error) {
	set, err := fonts.ReadFontSet(measure)
	if err != nil {
		return nil, err
	}
	r := &Renderer{set: set}
	if r.meas.text, err = parseFaces(measure, set.Text.Measure); err != nil {
		return nil, fmt.Errorf("%s: %w", set.Text.Name, err)
	}
	if r.meas.mono, err = parseFaces(measure, set.Mono.Measure); err != nil {
		return nil, fmt.Errorf("%s: %w", set.Mono.Name, err)
	}
	return r, nil
}

func (m measurer) Width(s string, size float64, style int) float64 {
	faces := m.text
	if style&styleMono != 0 {
		faces = m.mono
	}
	f := faces[400]
	if b := faces[700]; style&styleBold != 0 && b != nil {
		f = b
	}
	if style&(styleSub|styleSup) != 0 {
		size /= 1.2
	}
	var buf sfnt.Buffer
	ppem := fixed.Int26_6(size * 64)
	var w fixed.Int26_6
	for _, r := range s {
		i, err := f.GlyphIndex(&buf, r)
		if err != nil {
			continue
		}
		if a, err := f.GlyphAdvance(&buf, i, ppem, font.HintingNone); err == nil {
			w += a
		}
	}
	return float64(w) / 64
}

func Default() *Renderer {
	sub, err := fs.Sub(defaultFonts, "fonts")
	if err != nil {
		panic(err)
	}
	r, err := newRenderer(sub)
	if err != nil {
		panic("quire/bake/diagram: default fonts: " + err.Error())
	}
	return r
}

func (r *Renderer) FontID() string { return r.set.ID }

func (r *Renderer) Text(weight int) *sfnt.Font {
	if f := r.meas.text[weight]; f != nil {
		return f
	}
	return r.meas.text[400]
}

func (r *Renderer) Render(src string) (string, error) {
	name := journal.DiagramName(src, r.set.ID)
	out, err := sketch.Render(strings.TrimSpace(src), sketch.RenderOptions{
		Measurer:       r.meas,
		FontFamily:     r.set.Text.Name,
		FontFamilyMono: r.set.Mono.Name,

		IDPrefix: new("d" + name[len(name)-8:]),
	})
	if err != nil {
		return "", err
	}
	return out.Data, nil
}
