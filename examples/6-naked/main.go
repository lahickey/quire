// A site with no pack beneath it: Quire's machinery and nothing of its look.
// It has no stylesheet, no script, no frame and no default pages. What it
// shows is what this file and pages.templ say, and the look is yours to bring.
package main

import (
	"github.com/a-h/templ"
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/site"
)

func declare(s *site.Site) error {
	// A collection is its documents. Nothing draws them until a page does.
	posts, err := s.Collection(model.Basic{ID: "posts"})
	if err != nil {
		return err
	}
	// A page is its head data and a body. With no frame set, the body is the
	// whole of what is inside the document's body.
	s.AddPage(model.PageSpec{Title: "Naked", Path: "/"}, index(posts))
	s.AddDocumentPages(posts, func(d model.Document[model.Meta]) (model.PageSpec, templ.Component) {
		return model.DocumentPage(posts, d), post(d)
	})
	return nil
}

func main() {
	// RunBare is Run with no pack: it bakes and serves, and adds nothing.
	quire.RunBare(model.Identity{Name: "Naked", BaseURL: "https://example.com"}, site.FeatureFunc(declare))
}
