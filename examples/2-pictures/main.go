// The minimal site's code, with pictures and a diagram in its content.
// Nothing here asks for them: the bake finds them in the markdown.
// See content/posts for what each one does.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/app"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base"
)

func main() {
	app.Run(model.Identity{
		Name:        "Pictures",
		BaseURL:     "https://example.com",
		Description: "A Quire site with pictures and no CSS of its own.",
	}, quire.BlueprintFunc(func(site *quire.Site) error {
		posts, err := site.Collection(model.Basic{ID: "posts"})
		if err != nil {
			return err
		}
		base.Pages(site, posts)
		return nil
	}))
}
