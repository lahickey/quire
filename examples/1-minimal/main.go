// The smallest Quire site: one collection of markdown and the default look.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
)

func main() {
	// Run is the whole program: it bakes content/ into .quire, then serves the
	// site. Flags: --dev for live reload, --drafts, --addr (default :8090);
	// the argument bake bakes and exits.
	quire.Run(model.Identity{Name: "Example", BaseURL: "https://example.com"},
		// The blueprint declares what the site holds. content/posts/*.md becomes
		// /posts and /posts/{slug}; the home page, navigation, sitemap and
		// robots.txt follow from it.
		quire.BlueprintFunc(func(site *quire.Site) error {
			_, err := site.AddCollection(model.Basic{ID: "posts"})
			return err
		}))
}
