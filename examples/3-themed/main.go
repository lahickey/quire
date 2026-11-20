// A site with its own look: a home page, projects page and not-found page of
// its own, tags, a markdown widget, and its own palette, theme, stylesheets,
// icon and footer.
// Everything else is the default look, including each article and the writing index.
package main

import (
	"embed"
	"strings"

	"github.com/lahickey/quire"
	"github.com/lahickey/quire/app"
	"github.com/lahickey/quire/model"
)

// The site's own files: stylesheets and the favicon, compiled into the binary.
//
//go:embed *.css favicon.svg
var static embed.FS

// The tag vocabulary. Every tag in the frontmatter must be one of these or the
// site fails to load, and each renders with a class for its kind (tag-lang, tag-topic).
var kinds = model.Tags{"lang": {"Go"}, "topic": {"Design"}}

// A widget turns a named markdown fence into HTML: a ```swatch fence becomes a
// coloured chip, in any document of the site. It runs at serve time, so
// changing it needs no rebake.
var widgets = map[string]model.Widget{
	"swatch": func(_, body string) (string, error) {
		return `<p class="swatch">` + strings.TrimSpace(body) + `</p>`, nil
	},
}

// A palette of the site's own: a name, and the Radix hue behind each role.
var harbour = model.Palette{
	Name: "Harbour", Neutral: "olive", Accent: "teal", Spot: "tomato",
	Tone: [6]string{"sky", "pink", "indigo", "orange", "violet", "brown"},
	OK:   "grass", Warn: "amber", Error: "red",
}

// declare says what the site holds: its content and pages. It runs at bake and at boot.
func declare(site *quire.Site) error {
	// content/writing/*.md at /writing, with the default index and article.
	notes, err := site.AddCollection(model.Basic{ID: "writing", Title: "Writing", Tags: kinds})
	if err != nil {
		return err
	}
	// content/projects/*.md at /projects, with its own index: a grid of cards.
	projects, err := site.AddCollection(model.Basic{
		ID:    "projects",
		Title: "Projects",
		Tags:  kinds,
		Index: projectsPage,
	})
	if err != nil {
		return err
	}
	// The home page replaces the default and shows both collections.
	site.AddPage("/", homePage(site.Shell, notes, projects))
	site.SetNotFound(notFoundPage(site.Shell))
	return nil
}

func main() {
	app.Run(model.Identity{
		Name:        "Writing",
		Author:      "A. Writer",
		BaseURL:     "https://example.com",
		Description: "Notes on building things, and the things built.",
	}, quire.BlueprintFunc(declare),
		// Dark unless the visitor or their system says otherwise.
		quire.WithTheme(model.ThemeDark),
		quire.WithFooter(siteFooter),
		quire.WithPalette(model.PaletteInk),
		// A control in the navigation switches between Ink, the site's own
		// palette and two more of the built-in ones.
		quire.WithPalettePicker(harbour, model.PaletteMoss, model.PaletteDusk),
		quire.WithIcons(static, "favicon.svg"),
		quire.WithWidgets(widgets),
		// site.css applies at every width, site-narrow.css only on narrow screens.
		quire.WithStylesheets(static, "site.css"),
		quire.WithStylesheetsAt(model.WidthNarrow, static, "site-narrow.css"),
	)
}
