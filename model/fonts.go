package model

import (
	"encoding/json/v2"
	"fmt"
	"io/fs"
)

// FontsFile is a font set's description, beside the faces it names.
const FontsFile = "fonts.json"

type FontFace struct {
	File   string `json:"file"`
	Subset string `json:"subset"`
	Weight string `json:"weight"`
	Style  string `json:"style"`
	Range  string `json:"range"`
}

// FontWidths is how wide each character of one weight is, in units of which Em make the font's size.
// Of holds only the characters that are not Default wide.
type FontWidths struct {
	Em      int          `json:"em"`
	Default int          `json:"default"`
	Of      map[rune]int `json:"of,omitempty"`
}

type FontFamily struct {
	Name  string     `json:"name"`
	Faces []FontFace `json:"faces"`

	// Widths is by weight, for whatever lays text out where no browser is: quire fonts writes it.
	Widths map[int]FontWidths `json:"widths"`
}

// FontSet is the fonts a site draws in: a family for text and one for code, each with the faces a browser is sent
// and how wide its characters are.
type FontSet struct {
	Text FontFamily `json:"text"`
	Mono FontFamily `json:"mono"`
}

// Width is how wide a string is at a size, in the weight asked for or in 400 if the family has no such weight.
func (f FontFamily) Width(weight int, s string, size float64) float64 {
	w, ok := f.Widths[weight]
	if !ok {
		w = f.Widths[400]
	}
	if w.Em == 0 {
		return 0
	}
	units := 0
	for _, r := range s {
		if a, ok := w.Of[r]; ok {
			units += a
		} else {
			units += w.Default
		}
	}
	return float64(units) * size / float64(w.Em)
}

func ReadFontSet(fsys fs.FS) (FontSet, error) {
	var s FontSet
	b, err := fs.ReadFile(fsys, FontsFile)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("%s: %w", FontsFile, err)
	}
	if s.Text.Name == "" || s.Mono.Name == "" {
		return s, fmt.Errorf("%s: needs both families", FontsFile)
	}
	return s, nil
}
