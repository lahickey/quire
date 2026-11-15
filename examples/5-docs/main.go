// A documentation site: the docs pack makes the landing page, the sidebar and search from content/.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/docs"
)

func main() {
	quire.Run(model.Site{
		Name:        "Example",
		BaseURL:     "https://example.com",
		Description: "What the project is, in a sentence.",
	}, &docs.Pack{},
		// Optional: a footer link back to Quire.
		quire.WithFooterLinks(model.BuiltWithQuire))
}
