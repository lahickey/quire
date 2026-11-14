// A shop: a collection with its own frontmatter type and templates, and a
// basket plugin that adds routes, a page, sessions and live updates.
// The basket lives in its own package because a plugin is reusable: any
// site could mount it.
package main

import (
	"embed"

	"github.com/a-h/templ"
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/examples/4-plugin/basket"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
)

//go:embed site.css
var css embed.FS

// Piece is the frontmatter of a print: the stock fields, plus a medium.
// Embedding model.Meta keeps title, date, image and the rest working.
type Piece struct {
	model.Meta `yaml:",inline"`
	Medium     string `yaml:"medium"`
}

func mount(e *quire.Engine) error {
	// The templates need the basket, which needs the pieces, which need the
	// templates. Templates only run when a page renders, by which time b is set.
	var b *basket.Plugin
	pieces, err := e.AddCollection(model.Spec[Piece]{
		ID:    "pieces",
		Title: "Pieces",
		Index: wall,
		Item: func(shell model.Shell, c *model.Collection[Piece], d model.Document[Piece]) templ.Component {
			return piecePage(shell, b, c, d)
		},
	})
	if err != nil {
		return err
	}
	var items []basket.Item
	for _, d := range pieces.Docs() {
		items = append(items, basket.Item{ID: d.Slug, Title: d.Meta.Title})
	}
	b = basket.New(items...)
	e.AddPage("/", home(e.Shell, pieces))
	// Use attaches the plugin: its routes, page, stylesheets and nav link.
	// A plugin's pages live under /_basket unless the site moves them, as here.
	return e.Use(plugin.At("/basket", b))
}

func main() {
	quire.Run(model.Site{
		Name:        "Prints",
		BaseURL:     "https://example.com",
		Description: "Screenprints, and a basket that keeps up across tabs.",
	}, mount,
		quire.WithPalette(model.PaletteDusk),
		quire.WithStylesheets(css, "site.css"),
	)
}
