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

type FontFamily struct {
	Name  string     `json:"name"`
	Faces []FontFace `json:"faces"`

	Measure map[int]string `json:"measure"`
}

// FontSet is the fonts a site draws in: a family for text and one for code, each with the faces a browser is sent
// and the files a bake measures with. A bake and a site are handed the same set.
type FontSet struct {
	ID   string     `json:"id"`
	Text FontFamily `json:"text"`
	Mono FontFamily `json:"mono"`
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
	if s.ID == "" || s.Text.Name == "" || s.Mono.Name == "" {
		return s, fmt.Errorf("%s: needs an id and both families", FontsFile)
	}
	return s, nil
}
