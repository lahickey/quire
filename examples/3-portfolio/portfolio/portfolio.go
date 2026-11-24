// Package portfolio is the site: what it holds and how it looks. It is a
// package of its own, beside a main of a few lines, so that another program
// can run it as well.
package portfolio

import (
	"embed"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base"
	"github.com/lahickey/quire/pack/base/ui"
	"github.com/lahickey/quire/site"
)

// Identity is who the site is.
var Identity = model.Identity{
	Name:        "Writing",
	Author:      "A. Writer",
	BaseURL:     "https://example.com",
	Description: "Notes on building things, and the things built.",
}

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

// Declare says what the site holds and how it looks. It runs at bake and at
// boot, so it only declares.
func Declare(s *site.Site) error {
	// What changes how a document is drawn comes before any is read.
	s.SetWidget("swatch", swatch)
	// The default frame, with a footer of the site's own.
	s.SetFrame(ui.FrameWith(ui.Header, siteFooter))
	// site.css applies at every width, site-narrow.css only on narrow screens.
	s.AddStylesheets(static, "site.css")
	s.AddStylesheetsAt(model.WidthNarrow, static, "site-narrow.css")

	// content/writing/*.md, with the default look's listing at /writing and an
	// article for each document.
	notes, err := s.Collection(model.Spec{ID: "writing", Title: "Writing", Tags: kinds})
	if err != nil {
		return err
	}
	base.Pages(s, notes)
	// content/projects/*.md, with a listing and a page for each document that
	// this names: the default look's grid of cards at /projects, and its
	// article. A template of the site's own goes in either place.
	projects, err := s.Collection(model.Spec{ID: "projects", Title: "Projects", Tags: kinds})
	if err != nil {
		return err
	}
	s.AddCollectionPages(projects, ui.CardListing(projects), func(d model.Document) templ.Component { return ui.Article(projects, d) })
	// A page is its head data and its body. This one takes the place of the
	// default home page.
	s.AddPage(model.PageSpec{Title: "Writing", Path: "/"}, homePage(notes, projects))
	// A page of the site's own whose words are in content/contact.md, so they
	// are edited as markdown and not as a template.
	contact, err := s.Markdown("contact.md")
	if err != nil {
		return err
	}
	s.AddPage(model.PageSpec{Title: "Contact", Path: "/contact"}, contactPage(contact))
	s.SetNotFound(notFoundPage())
	return nil
}

// Options are what has to be settled before Declare runs.
func Options() []site.Option {
	return []site.Option{
		// Dark unless the visitor or their system says otherwise.
		site.WithTheme(model.ThemeDark),
		site.WithPalette(model.PaletteInk),
		// A control in the navigation switches between Ink, the site's own
		// palette and two more of the built-in ones.
		site.WithPalettePicker(harbour, model.PaletteMoss, model.PaletteDusk),
		site.WithIcons(static, "favicon.svg"),
	}
}
