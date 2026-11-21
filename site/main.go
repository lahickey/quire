// Quire's own documentation site, made by the docs pack from content/.
package main

import (
	"embed"
	"strings"

	"github.com/lahickey/quire"
	"github.com/lahickey/quire/app"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/docs"
)

//go:embed site.css
var css embed.FS

func swatch(_, body string) (string, error) {
	return `<p class="swatch">` + strings.TrimSpace(body) + `</p>`, nil
}

func declare(site *quire.Site) error {
	site.SetWidget("swatch", swatch)
	site.AddStylesheets(css, "site.css")
	site.AddFooterLinks(model.BuiltWithQuire, model.Link{Label: "A Hidden Bridge project", URL: "https://hiddenbridge.io"})
	return site.Include(&docs.Pack{})
}

func main() {
	app.Run(model.Identity{
		Name:        "Quire",
		BaseURL:     "https://quire.hiddenbridge.io",
		Description: "Composable parts for static and live sites in Go.",
	}, quire.BlueprintFunc(declare),
		quire.WithPalettePicker(),
		quire.WithPalette(model.PaletteInk),
	)
}
