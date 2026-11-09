package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/lahickey/quire/model"
)

const layerOrder = "@layer quire, plugin, site;\n"

//go:embed *.css *.js
var files embed.FS

var ownSheets = []string{"base.css", "toggle.css", "nav.css", "layout.css", "prose.css", "highlight.css"}

type Sheet struct {
	FS    fs.FS
	Name  string
	Layer string
	At    model.Width
}

type Parts struct {
	Scale   model.Scale
	Palette model.Palette
	Fonts   []byte
	Sheets  []Sheet
}

func own(name string) []byte {
	body, err := files.ReadFile(name)
	if err != nil {
		panic("quire: asset " + name + ": " + err.Error())
	}
	return body
}

func read(f Sheet) []byte {
	body, err := fs.ReadFile(f.FS, f.Name)
	if err != nil {
		panic("quire: " + f.Name + ": " + err.Error())
	}
	return body
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

func Stylesheet(p Parts) []byte {
	var b strings.Builder
	appendCSS(&b, "quire", "scale", "", scaleCSS(p.Scale))
	appendCSS(&b, "quire", "fonts", "", p.Fonts)
	appendCSS(&b, "quire", "palette", "", paletteCSS(p.Palette))
	for _, name := range ownSheets {
		appendCSS(&b, "quire", name, "", own(name))
	}
	appendCSS(&b, "quire", "breakpoint", "", breakpointCSS(p.Scale))
	for _, w := range append([]model.Width{0}, model.Widths...) {
		q := ""
		if w != 0 {
			q = media(p.Scale, w)
		}
		for _, layer := range []string{"plugin", "site"} {
			for _, f := range p.Sheets {
				if f.At == w && f.Layer == layer {
					appendCSS(&b, layer, f.Name, q, read(f))
				}
			}
		}
	}
	return []byte(b.String())
}

func Script() []byte {
	var b strings.Builder
	appendJS(&b, "theme.js", own("theme.js"))
	appendJS(&b, "nav.js", own("nav.js"))
	return []byte(b.String())
}

func Datastar() []byte { return own("datastar.js") }
