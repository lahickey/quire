// A site with no pack beneath it: Quire's machinery and nothing of its look.
// It has no stylesheet, no script, no frame and no default pages. What it
// shows is what pages.templ says, and the look is yours to bring.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/feature/content"
	"github.com/lahickey/quire/model"
)

func main() {
	// RunBare is Run with no pack: it bakes and serves, and adds nothing.
	//
	// content.Pages makes the site from the content folder, as base.Content
	// does, and has no look in it: it is handed how each kind of page is
	// drawn. With no frame set, each body is the whole of what is inside the
	// document's body.
	quire.RunBare(model.Identity{Name: "Naked", BaseURL: "https://example.com"},
		content.Pages{Page: page, Listing: listing, Document: post})
}
