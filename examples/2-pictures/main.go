// The minimal site's code, with pictures and a diagram in its content.
// Nothing here asks for them: the bake finds them in the markdown.
// See content/posts for what each one does.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
)

func main() {
	quire.Run(model.Site{
		Name:        "Pictures",
		BaseURL:     "https://example.com",
		Description: "A Quire site with pictures and no CSS of its own.",
	}, func(e *quire.Engine) error {
		_, err := e.AddCollection(model.Basic{ID: "posts"})
		return err
	})
}
