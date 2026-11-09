// A site with its own look: a home page, projects page and not-found page of
// its own, tags, a markdown widget, and its own palette, theme, stylesheets,
// icon and footer.
// Everything else is the default look, including each article and the writing index.
package main

import (
	"embed"
	"strings"

	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
)

// The site's own files: stylesheets and the favicon, compiled into the binary.
//
//go:embed *.css favicon.svg
var static embed.FS

// The tag vocabulary. Every tag in the frontmatter must be one of these or the
// site fails to load, and each renders with a class for its kind (tag-lang, tag-topic).
var kinds = model.NewClassifier(model.Exact("lang", "Go"), model.Exact("topic", "Design"))

// A widget turns a named markdown fence into HTML: a ```swatch fence becomes a
// coloured chip. It runs at serve time, so changing it needs no rebake.
var widgets = map[string]model.Widget{
	"swatch": func(_, body string) (string, error) {
		return `<p class="swatch">` + strings.TrimSpace(body) + `</p>`, nil
	},
}

// mount declares the site's content and pages. It runs at bake and at boot.
func mount(e *quire.Engine) error {
	// content/writing/*.md at /writing, with the default index and article.
	notes, err := e.AddCollection(model.Basic{ID: "writing", Title: "Writing", Classifier: kinds, Widgets: widgets})
	if err != nil {
		return err
	}
	// content/projects/*.md at /projects, with its own index: a grid of cards.
	projects, err := e.AddCollection(model.Basic{
		ID:         "projects",
		Title:      "Projects",
		Classifier: kinds,
		Index:      projectsPage,
	})
	if err != nil {
		return err
	}
	// The home page replaces the default and shows both collections.
	e.AddPage("/", homePage(e.Shell, notes, projects))
	e.SetNotFound(notFoundPage(e.Shell))
	return nil
}

func main() {
	quire.Run(model.Site{
		Name:        "Writing",
		Author:      "A. Writer",
		BaseURL:     "https://example.com",
		Description: "Notes on building things, and the things built.",
	}, mount,
		// Dark unless the visitor or their system says otherwise.
		quire.WithTheme(model.ThemeDark),
		quire.WithFooter(siteFooter()),
		// Without this, .page--wide (the card grids) is no wider than a plain page.
		quire.WithWidest(model.WidthWide),
		quire.WithPalette(model.PaletteInk),
		quire.WithIcons(static, "favicon.svg"),
		// site.css applies at every width, site-narrow.css only on narrow screens.
		quire.WithStylesheets(static, "site.css"),
		quire.WithStylesheetsAt(model.WidthNarrow, static, "site-narrow.css"),
	)
}
