// Package docs is a pack: a documentation site made from content/index.md and content/docs/*.md.
// A landing page, the documents beside a sidebar, and search.
package docs

import (
	"cmp"
	"embed"
	"fmt"

	"github.com/a-h/templ"
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
	"github.com/lahickey/quire/plugin/search"
	"github.com/lahickey/quire/ui"
)

//go:embed *.css
var css embed.FS

type (
	Collection = model.Collection[model.Meta]
	Document   = model.Document[model.Meta]
)

// Section is one heading of the sidebar and the documents under it, by slug, in order.
type Section struct {
	Title string   `yaml:"title"`
	Pages []string `yaml:"pages"`
}

// Landing is the frontmatter of content/index.md. Its body is drawn below the features.
type Landing struct {
	Kicker   string        `yaml:"kicker"`
	Title    string        `yaml:"title"`
	Lede     string        `yaml:"lede"`
	Install  string        `yaml:"install"`
	Features []ui.CardData `yaml:"features"`
	Repo     string        `yaml:"repo"`
	Footer   string        `yaml:"footer"`
	Docs     struct {
		Title    string    `yaml:"title"`
		Summary  string    `yaml:"summary"`
		Sections []Section `yaml:"sections"`
	} `yaml:"docs"`
}

// Pack's fields each replace one page; nil is the default of the same name.
type Pack struct {
	Widgets map[string]model.Widget

	Home  func(model.Shell, model.Site, model.Document[Landing], *Collection) templ.Component
	Index func(model.Shell, *Collection) templ.Component
	Item  func(model.Shell, *Collection, Document) templ.Component
}

type group struct {
	Title string
	Docs  []Document
}

type place struct {
	order   int
	section string
}

func groups(c *Collection) []group {
	var out []group
	for _, d := range c.Docs() {
		if title := c.Kicker(d); len(out) == 0 || out[len(out)-1].Title != title {
			out = append(out, group{Title: title})
		}
		out[len(out)-1].Docs = append(out[len(out)-1].Docs, d)
	}
	return out
}

func (p Pack) Mount(e *quire.Engine) error {
	landing, err := e.Document[Landing]("index.md")
	if err != nil {
		return err
	}
	l := landing.Meta
	places := map[string]place{}
	for _, s := range l.Docs.Sections {
		for _, slug := range s.Pages {
			if _, twice := places[slug]; twice {
				return fmt.Errorf("index.md lists %q twice", slug)
			}
			places[slug] = place{len(places), s.Title}
		}
	}
	home, index, item := p.Home, p.Index, p.Item
	if home == nil {
		home = Home
	}
	if index == nil {
		index = Index
	}
	if item == nil {
		item = Item
	}
	c, err := e.AddCollection(model.Basic{
		ID:          "docs",
		Title:       cmp.Or(l.Docs.Title, "Documentation"),
		Description: l.Docs.Summary,
		Widgets:     p.Widgets,
		Kicker:      func(d Document) string { return places[d.Slug].section },
		Sort:        func(a, b Document) int { return cmp.Compare(places[a.Slug].order, places[b.Slug].order) },
		Index:       index,
		Item:        item,
	})
	if err != nil {
		return err
	}
	for _, d := range c.Docs() {
		if _, listed := places[d.Slug]; !listed {
			return fmt.Errorf("%s is not in a section of index.md", d.Path)
		}
	}
	for slug := range places {
		if _, ok := c.Get(slug); !ok {
			return fmt.Errorf("index.md lists %q, which is not a document in docs", slug)
		}
	}
	e.AddStylesheets(css, "docs.css")
	e.AddStylesheetsAt(model.WidthWide, css, "docs-wide.css")
	if l.Repo != "" {
		e.AddNavEnd(repoLink(l.Repo))
	}
	if l.Footer != "" {
		e.SetFooter(func(model.Chrome, model.PageSpec) templ.Component { return footer(l) })
	}
	e.AddPage("/", home(e.Shell, e.Site(), landing, c))
	s := search.New()
	s.Add(c)
	return e.Use(plugin.At("/search", s))
}
