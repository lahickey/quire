// Package assets assembles a site's stylesheet and script from what its features, packs and the site itself add.
package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/lahickey/quire/model"
)

const layerOrder = "@layer base, feature, pack, site;\n"

//go:embed datastar.js
var files embed.FS

// File is one sheet or script: read from FS by Name, or Body when it was made and not read.
type File struct {
	FS    fs.FS
	Name  string
	Body  []byte
	Layer string
	At    model.Width
}

type Parts struct {
	Scale   model.Scale
	Fonts   []byte
	Sheets  []File
	Scripts []File
}

func read(f File) []byte {
	if f.FS == nil {
		return f.Body
	}
	body, err := fs.ReadFile(f.FS, f.Name)
	if err != nil {
		panic("quire: " + f.Name + ": " + err.Error())
	}
	return body
}

func media(s model.Scale, w model.Width) string {
	switch w {
	case model.WidthNarrow:
		return fmt.Sprintf("@media (width < %dpx)", s.Measure())
	case model.WidthWide:
		return fmt.Sprintf("@media (width >= %dpx)", s.Wide())
	case model.WidthFull:
		return fmt.Sprintf("@media (width >= %dpx)", s.Full())
	}
	return ""
}

func appendCSS(b *strings.Builder, layer, name, query string, body []byte) {
	if b.Len() == 0 {
		b.WriteString(layerOrder)
	} else {
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "/* --- %s --- */\n", name)
	if query != "" {
		fmt.Fprintf(b, "%s {\n", query)
	}
	fmt.Fprintf(b, "@layer %s {\n", layer)
	b.Write(body)
	if !strings.HasSuffix(string(body), "\n") {
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	if query != "" {
		b.WriteString("}\n")
	}
}

func appendJS(b *strings.Builder, name string, body []byte) {
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "/* --- %s --- */\n(function () {\n\"use strict\";\n", name)
	b.Write(body)
	if !strings.HasSuffix(string(body), "\n") {
		b.WriteString("\n")
	}
	b.WriteString("})();\n")
}

// Stylesheet is one sheet in four layers: the look at the bottom, then named features, packs and the site, each at its width.
func Stylesheet(p Parts) []byte {
	var b strings.Builder
	if len(p.Fonts) > 0 {
		appendCSS(&b, "base", "fonts", "", p.Fonts)
	}
	for _, w := range append([]model.Width{0}, model.Widths...) {
		q := ""
		if w != 0 {
			q = media(p.Scale, w)
		}
		for _, layer := range []string{"base", "feature", "pack", "site"} {
			for _, f := range p.Sheets {
				if f.At == w && f.Layer == layer {
					appendCSS(&b, layer, f.Name, q, read(f))
				}
			}
		}
	}
	return []byte(b.String())
}

func Script(p Parts) []byte {
	var b strings.Builder
	for _, f := range p.Scripts {
		appendJS(&b, f.Name, read(f))
	}
	return []byte(b.String())
}

func Datastar() []byte {
	body, err := files.ReadFile("datastar.js")
	if err != nil {
		panic("quire: datastar: " + err.Error())
	}
	return body
}
