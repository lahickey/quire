// The smallest Quire site: one collection of markdown and the default look.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base"
	"github.com/lahickey/quire/site"
)

func main() {
	// Run is the whole program: it bakes content/ into .quire, then serves the
	// site. Flags: --dev for live reload, --drafts, --addr (default :8090);
	// the argument bake bakes and exits.
	quire.Run(model.Identity{Name: "Example", BaseURL: "https://example.com"},
		// The feature declares what the site holds. content/posts/*.md is a
		// collection, and base.Pages gives it the default look's two pages: a
		// listing at /posts and an article at /posts/{slug}. The home page,
		// navigation, sitemap and robots.txt follow from them.
		site.FeatureFunc(func(s *site.Site) error {
			posts, err := s.Collection(model.Basic{ID: "posts"})
			if err != nil {
				return err
			}
			base.Pages(s, posts)
			return nil
		}))
}
