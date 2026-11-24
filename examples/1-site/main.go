// The smallest Quire site: a folder of markdown and the default look.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base"
)

func main() {
	// Run is the whole program: it bakes content/ into .quire, then serves the
	// site. Flags: --dev for live reload, --drafts, --addr (default :8090);
	// the argument bake bakes and exits.
	//
	// base.Content makes the site from the content folder. index.md is the
	// home page, about.md is a page at /about, and the folder posts is a
	// collection: a listing at /posts and an article at /posts/{slug}. The
	// navigation, sitemap and robots.txt follow from them.
	quire.Run(model.Identity{Name: "Example", BaseURL: "https://example.com"}, base.Content())
}
