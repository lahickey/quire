// A site with its own look: a home page, projects page and not-found page of
// its own, tags, a markdown widget, and its own palette, theme, stylesheets,
// icon and footer.
// Everything else is the default look, including each article and the writing index.
package main

import (
	"embed"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/app"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base"
	"github.com/lahickey/quire/pack/base/ui"
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
func swatch(_, body string) (string, error) {
	return `<p class="swatch">` + strings.TrimSpace(body) + `</p>`, nil
}

// A palette of the site's own: a name, and the Radix hue behind each role.
var harbour = model.Palette{
	Name: "Harbour", Neutral: "olive", Accent: "teal", Spot: "tomato",
	Tone: [6]string{"sky", "pink", "indigo", "orange", "violet", "brown"},
	OK:   "grass", Warn: "amber", Error: "red",
}

// declare says what the site holds and how it looks. It runs at bake and at
// boot, so it only declares.
func declare(site *quire.Site) error {
	// What changes how a document is drawn comes before any is read.
	site.SetWidget("swatch", swatch)
	// The default frame, with a footer of the site's own.
	site.SetFrame(ui.FrameWith(ui.Header, siteFooter))
	// site.css applies at every width, site-narrow.css only on narrow screens.
	site.AddStylesheets(static, "site.css")
	site.AddStylesheetsAt(model.WidthNarrow, static, "site-narrow.css")

	// content/writing/*.md, with the default look's listing at /writing and an
	// article for each document.
	notes, err := site.Collection(model.Basic{ID: "writing", Title: "Writing", Tags: kinds})
	if err != nil {
		return err
	}
	base.Pages(site, notes)
	// content/projects/*.md, with an index of its own at /projects, a grid of
	// cards, and the default article for each document.
	projects, err := site.Collection(model.Basic{ID: "projects", Title: "Projects", Tags: kinds})
	if err != nil {
		return err
	}
	site.AddPage(projects.Page(), projectsPage(projects))
	site.AddDocumentPages(projects, func(d model.Document[model.Meta]) (model.PageSpec, templ.Component) {
		return model.DocumentPage(projects, d), ui.Article(projects, d)
	})
	// A page is its head data and its body. This one takes the place of the
	// default home page.
	site.AddPage(model.PageSpec{Title: "Writing", Path: "/"}, homePage(notes, projects))
	site.SetNotFound(notFoundPage())
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
		quire.WithPalette(model.PaletteInk),
		// A control in the navigation switches between Ink, the site's own
		// palette and two more of the built-in ones.
		quire.WithPalettePicker(harbour, model.PaletteMoss, model.PaletteDusk),
		quire.WithIcons(static, "favicon.svg"),
	)
}
