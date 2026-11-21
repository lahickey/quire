// A shop: a collection with its own frontmatter type and templates, and a
// basket plugin that adds routes, a page, sessions and live updates.
// The basket lives in its own package because a plugin is reusable: any
// site could use it.
package main

import (
	"embed"

	"github.com/a-h/templ"
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/app"
	"github.com/lahickey/quire/examples/4-plugin/basket"
	"github.com/lahickey/quire/model"
)

//go:embed site.css
var css embed.FS

// Piece is the frontmatter of a print: the stock fields, plus a medium.
// Embedding model.Meta keeps title, date, picture and the rest working.
type Piece struct {
	model.Meta `yaml:",inline"`
	Medium     string `yaml:"medium"`
}

// declare says what the site holds: its content, its pages and its plugin.
// It runs in a bake and when serving, so it only declares.
func declare(site *quire.Site) error {
	site.AddStylesheets(css, "site.css")
	pieces, err := site.Collection(model.Spec[Piece]{ID: "pieces", Title: "Pieces"})
	if err != nil {
		return err
	}
	var items []basket.Item
	for _, d := range pieces.Docs() {
		items = append(items, basket.Item{ID: d.Slug, Title: d.Meta.Title})
	}
	// Use says the site has the plugin: its endpoints, stylesheets and nav
	// link. It is attached only when the site serves, never in a bake.
	b := basket.New("/basket", items...)
	site.AddPlugins(b)
	// A plugin owns its endpoints and no pages. Its page is one the site adds,
	// drawn for each request.
	site.AddLivePage("/basket", b.Page)

	// A collection is its documents and nothing more; these are its pages.
	site.AddPage(pieces.Page(), wall(pieces))
	site.AddDocumentPages(pieces, func(d model.Document[Piece]) (model.PageSpec, templ.Component) {
		return model.DocumentPage(pieces, d), piecePage(b, pieces, d)
	})
	site.AddPage(model.PageSpec{Title: "Prints", Path: "/"}, home(pieces))
	return nil
}

func main() {
	app.Run(model.Identity{
		Name:        "Prints",
		BaseURL:     "https://example.com",
		Description: "Screenprints, and a basket that keeps up across tabs.",
	}, quire.BlueprintFunc(declare),
		quire.WithPalette(model.PaletteDusk),
	)
}
