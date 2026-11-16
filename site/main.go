// Quire's own documentation site, made by the docs pack from content/.
package main

import (
	"embed"
	"strings"

	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/docs"
)

//go:embed site.css
var css embed.FS

var widgets = map[string]model.Widget{
	"swatch": func(_, body string) (string, error) {
		return `<p class="swatch">` + strings.TrimSpace(body) + `</p>`, nil
	},
}

func main() {
	quire.Run(model.Site{
		Name:        "Quire",
		BaseURL:     "https://quire.hiddenbridge.io",
		Description: "A Go engine that turns a folder of markdown into a finished website, served from a single binary.",
	}, &docs.Pack{Widgets: widgets},
		quire.WithStylesheets(css, "site.css"),
		quire.WithPalettePicker(),
		quire.WithFooterLinks(model.BuiltWithQuire, model.Link{Label: "A Hidden Bridge project", URL: "https://hiddenbridge.io"}),
		quire.WithPalette(model.PaletteInk),
	)
}
