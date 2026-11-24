// Quire's own documentation site, made by the docs pack from content/, with each example running beneath /examples/.
package main

import (
	"embed"
	"strings"

	"github.com/lahickey/quire"
	"github.com/lahickey/quire/examples/3-portfolio/portfolio"
	"github.com/lahickey/quire/examples/4-shop/shop"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base"
	"github.com/lahickey/quire/pack/docs"
	"github.com/lahickey/quire/site"
)

const address = "https://quire.hiddenbridge.io"

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

// example is one of the examples as it is in the repository, served under /examples/ from its own content folder.
func example(dir string, id model.Identity, f site.Feature, opts ...site.Option) quire.Site {
	_, name, _ := strings.Cut(dir, "-")
	id.BaseURL = address
	return quire.Site{Identity: id, Feature: f, Options: opts, Root: "/examples/" + name, Content: "../examples/" + dir + "/content"}
}

func main() {
	quire.RunSites(
		quire.Site{
			Identity: model.Identity{Name: "Quire", BaseURL: address, Description: "Composable parts for static and live sites in Go."},
			Feature:  site.FeatureFunc(declare),
			Options:  []site.Option{site.WithPalettePicker(), site.WithPalette(model.PaletteInk)},
		},
		example("1-site", model.Identity{Name: "Example"}, base.Content()),
		example("2-blog", model.Identity{Name: "Pictures", Description: "A Quire site with pictures and no CSS of its own."}, base.Content()),
		example("3-portfolio", portfolio.Identity, site.FeatureFunc(portfolio.Declare), portfolio.Options()...),
		example("4-shop", shop.Identity, shop.Feature(), shop.Options()...),
		example("5-docs", model.Identity{Name: "Example", Description: "What the project is, in a sentence."}, site.FeatureFunc(func(s *site.Site) error {
			s.AddFooterLinks(model.BuiltWithQuire)
			return s.Include(&docs.Pack{})
		})),
	)
}
