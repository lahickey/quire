// Package docs is a pack: a documentation site made from content/index.md and content/docs/*.md.
// A landing page, the documents beside a sidebar, and search.
package docs

import (
	"cmp"
	"embed"
	"fmt"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/feature/search"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base/searchui"
	"github.com/lahickey/quire/pack/base/ui"
	"github.com/lahickey/quire/site"
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

// Landing is the frontmatter of content/index.md. Its body is drawn below the hero.
type Landing struct {
	Kicker  string `yaml:"kicker"`
	Title   string `yaml:"title"`
	Lede    string `yaml:"lede"`
	Install string `yaml:"install"`
	Repo    string `yaml:"repo"`
	Footer  string `yaml:"footer"`
	Docs    struct {
		Title    string    `yaml:"title"`
		Summary  string    `yaml:"summary"`
		Sections []Section `yaml:"sections"`
	} `yaml:"docs"`
}

// Book is the documents, and the section index.md puts each in.
type Book struct {
	*Collection
	sections map[string]string
}

// Pack's fields each replace one page; nil is the default of the same name.
type Pack struct {
	Home  func(model.Identity, model.Document[Landing], *Book) templ.Component
	Index func(*Book) templ.Component
	Item  func(*Book, Document) templ.Component
}

type group struct {
	Title string
	Docs  []Document
}

func (b *Book) Section(d Document) string { return b.sections[d.Slug] }

func groups(b *Book) []group {
	var out []group
	for _, d := range b.Docs() {
		if title := b.Section(d); len(out) == 0 || out[len(out)-1].Title != title {
			out = append(out, group{Title: title})
		}
		out[len(out)-1].Docs = append(out[len(out)-1].Docs, d)
	}
	return out
}

func (p *Pack) Name() string { return "" }

func (p *Pack) Declare(s *site.Site) error {
	landing, err := s.Document[Landing]("index.md")
	if err != nil {
		return err
	}
	l := landing.Meta
	order, sections := map[string]int{}, map[string]string{}
	for _, s := range l.Docs.Sections {
		for _, slug := range s.Pages {
			if _, twice := order[slug]; twice {
				return fmt.Errorf("index.md lists %q twice", slug)
			}
			order[slug], sections[slug] = len(order), s.Title
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
	c, err := s.Collection(model.Basic{
		ID:          "docs",
		Title:       cmp.Or(l.Docs.Title, "Documentation"),
		Description: l.Docs.Summary,
		Sort:        func(a, b Document) int { return cmp.Compare(order[a.Slug], order[b.Slug]) },
	})
	if err != nil {
		return err
	}
	for _, d := range c.Docs() {
		if _, listed := order[d.Slug]; !listed {
			return fmt.Errorf("%s is not in a section of index.md", d.Path)
		}
	}
	for slug := range order {
		if _, ok := c.Get(slug); !ok {
			return fmt.Errorf("index.md lists %q, which is not a document in docs", slug)
		}
	}
	s.AddStylesheets(css, "docs.css")
	s.AddStylesheetsAt(model.WidthWide, css, "docs-wide.css")
	if l.Repo != "" {
		s.AddNavEnd(ui.Link(source(l.Repo), "site-repo"))
		s.AddFooterLinks(source(l.Repo))
	}
	if l.Footer != "" {
		s.SetFooterText(l.Footer)
	}
	book := &Book{c, sections}
	s.AddPage(c.Page(), index(book))
	s.AddDocumentPages(c, func(d Document) (model.PageSpec, templ.Component) {
		return model.DocumentPage(c, d), item(book, d)
	})
	s.AddPage(model.PageSpec{Title: s.Identity().Name, Path: "/"}, home(s.Identity(), landing, book))
	found := search.New("/search", searchui.Look())
	found.Add(c)
	return s.Include(found)
}
