package fonts

import (
	"embed"
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

const FontsName = "fonts.json"

const (
	sansStack = `system-ui, -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif`
	monoStack = `ui-monospace, Menlo, Consolas, monospace`
)

//go:embed web
var web embed.FS

type Face struct {
	File   string `json:"file"`
	Subset string `json:"subset"`
	Weight string `json:"weight"`
	Style  string `json:"style"`
	Range  string `json:"range"`
}

type Family struct {
	Name  string `json:"name"`
	Faces []Face `json:"faces"`

	Measure map[int]string `json:"measure"`
}

type FontSet struct {
	ID   string `json:"id"`
	Text Family `json:"text"`
	Mono Family `json:"mono"`
}

func family(name, stack string) string {
	if name == "" {
		return stack
	}
	return strconv.Quote(name) + ", " + stack
}

func ReadFontSet(fsys fs.FS) (FontSet, error) {
	var s FontSet
	b, err := fs.ReadFile(fsys, FontsName)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("%s: %w", FontsName, err)
	}
	if s.ID == "" || s.Text.Name == "" || s.Mono.Name == "" {
		return s, fmt.Errorf("%s: needs an id and both families", FontsName)
	}
	return s, nil
}

func Default() fs.FS {
	sub, err := fs.Sub(web, "web")
	if err != nil {
		panic("quire: default fonts: " + err.Error())
	}
	return sub
}

func Faces(set FontSet, fsys fs.FS, serve func(file string, body []byte) string) (css []byte, preload []string) {
	var b strings.Builder
	for _, f := range []Family{set.Text, set.Mono} {
		for _, face := range f.Faces {
			body, err := fs.ReadFile(fsys, face.File)
			if err != nil {
				panic("quire: font " + face.File + ": " + err.Error())
			}
			url := serve(face.File, body)
			fmt.Fprintf(&b, "@font-face {\n  font-family: %q;\n  font-style: %s;\n  font-weight: %s;\n  font-display: swap;\n  src: url(%q) format(\"woff2\");\n  unicode-range: %s;\n}\n\n",
				f.Name, face.Style, face.Weight, url, face.Range)
			if f.Name == set.Text.Name && face.Subset == "latin" {
				preload = append(preload, url)
			}
		}
	}
	fmt.Fprintf(&b, ":root {\n  --qr-font: %s;\n  --qr-font-mono: %s;\n}\n",
		family(set.Text.Name, sansStack), family(set.Mono.Name, monoStack))
	return []byte(b.String()), preload
}
