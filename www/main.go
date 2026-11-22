// Quire's own documentation site, made by the docs pack from content/.
package main

import (
	"embed"
	"strings"

	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/docs"
	"github.com/lahickey/quire/site"
)

//go:embed site.css
var css embed.FS

func swatch(_, body string) (string, error) {
	return `<p class="swatch">` + strings.TrimSpace(body) + `</p>`, nil
}

func declare(s *site.Site) error {
	s.SetWidget("swatch", swatch)
	s.AddStylesheets(css, "site.css")
	s.AddFooterLinks(model.BuiltWithQuire, model.Link{Label: "A Hidden Bridge project", URL: "https://hiddenbridge.io"})
	return s.Include(&docs.Pack{})
}

func main() {
	quire.Run(model.Identity{
		Name:        "Quire",
		BaseURL:     "https://quire.hiddenbridge.io",
		Description: "Composable parts for static and live sites in Go.",
	}, site.FeatureFunc(declare),
		site.WithPalettePicker(),
		site.WithPalette(model.PaletteInk),
	)
}
