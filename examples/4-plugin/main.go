// A shop: a collection with its own frontmatter type and templates, and a
// basket plugin that adds routes, a page, sessions and live updates.
// The basket lives in its own package because a plugin is reusable: any
// site could use it.
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

// shop is the site. Its two methods are the two stages Quire runs.
type shop struct {
	pieces *model.Collection[Piece]
	basket *basket.Plugin
}

// Declare says what the site holds: its content and pages. It runs in a bake
// and when serving, so it only declares.
func (s *shop) Declare(site *quire.Site) error {
	var err error
	s.pieces, err = site.AddCollection(model.Spec[Piece]{
		ID:    "pieces",
		Title: "Pieces",
		Index: wall,
		// The basket is made in Live. A template only runs when a page
		// renders, by which time it is set.
		Item: func(shell model.Shell, c *model.Collection[Piece], d model.Document[Piece]) templ.Component {
			return piecePage(shell, s.basket, c, d)
		},
	})
	if err != nil {
		return err
	}
	site.AddPage("/", home(site.Shell, s.pieces))
	return nil
}

// Live adds what only a served site has: plugins and routes. A bake never
// calls it.
func (s *shop) Live(l *quire.Live) error {
	var items []basket.Item
	for _, d := range s.pieces.Docs() {
		items = append(items, basket.Item{ID: d.Slug, Title: d.Meta.Title})
	}
	s.basket = basket.New(items...)
	// Use attaches the plugin: its routes, page, stylesheets and nav link.
	// A plugin's pages live under /_basket unless the site moves them, as here.
	return l.Use(plugin.At("/basket", s.basket))
}

func main() {
	quire.Run(model.Identity{
		Name:        "Prints",
		BaseURL:     "https://example.com",
		Description: "Screenprints, and a basket that keeps up across tabs.",
	}, &shop{},
		quire.WithPalette(model.PaletteDusk),
		quire.WithStylesheets(css, "site.css"),
	)
}
