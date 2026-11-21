// A documentation site: the docs pack makes the landing page, the sidebar and search from content/.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/app"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/docs"
)

func main() {
	app.Run(model.Identity{
		Name:        "Example",
		BaseURL:     "https://example.com",
		Description: "What the project is, in a sentence.",
	}, quire.BlueprintFunc(func(site *quire.Site) error {
		// Optional: a footer link back to Quire.
		site.AddFooterLinks(model.BuiltWithQuire)
		// The docs pack is the whole site; a pack is included like this.
		return site.Include(&docs.Pack{})
	}))
}
