// Package measure is a font set's faces for measuring and drawing text, which only a bake does.
package measure

import (
	"fmt"
	"io/fs"

	"github.com/lahickey/quire/model"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type Faces struct {
	Set        model.FontSet
	text, mono map[int]*sfnt.Font
}

func parse(ttf fs.FS, files map[int]string) (map[int]*sfnt.Font, error) {
	out := map[int]*sfnt.Font{}
	for weight, file := range files {
		b, err := fs.ReadFile(ttf, file)
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

func Read(set model.FontSet, ttf fs.FS) (*Faces, error) {
	f := &Faces{Set: set}
	var err error
	if f.text, err = parse(ttf, set.Text.Measure); err != nil {
		return nil, fmt.Errorf("%s: %w", set.Text.Name, err)
	}
	if f.mono, err = parse(ttf, set.Mono.Measure); err != nil {
		return nil, fmt.Errorf("%s: %w", set.Mono.Name, err)
	}
	return f, nil
}

func (f *Faces) Face(mono bool, weight int) *sfnt.Font {
	faces := f.text
	if mono {
		faces = f.mono
	}
	if face := faces[weight]; face != nil {
		return face
	}
	return faces[400]
}

func Width(f *sfnt.Font, s string, size float64) float64 {
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
