// Quire's own documentation site: a landing page, the docs in a beside a sidebar, and search.
package main

import (
	"cmp"
	"embed"
	"fmt"
	"slices"

	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
	"github.com/lahickey/quire/plugin/search"
)

const repo = "https://github.com/lahickey/quire"

//go:embed *.css
var css embed.FS

type docs = *model.Collection[model.Meta]

type doc = model.Document[model.Meta]

type section struct {
	Title string
	Slugs []string
}

// The sidebar: the sections in order, and the documents in each. A document
// that is not listed, or a slug with no document, fails the load.
var sections = []section{
	{"Start here", []string{"getting-started"}},
	{"Authoring", []string{"content", "images", "look"}},
	{"Building", []string{"wiring", "plugins", "bake"}},
	{"Background", []string{"architecture"}},
}

func place(slug string) (order int, title string) {
	n := 0
	for _, s := range sections {
		if i := slices.Index(s.Slugs, slug); i >= 0 {
			return n + i, s.Title
		}
		n += len(s.Slugs)
	}
	return -1, ""
}

func listed(c docs) error {
	for _, d := range c.Docs() {
		if i, _ := place(d.Slug); i < 0 {
			return fmt.Errorf("%s is not in the sidebar", d.Path)
		}
	}
	for _, s := range sections {
		for _, slug := range s.Slugs {
			if _, ok := c.Get(slug); !ok {
				return fmt.Errorf("the sidebar lists %q, which is not a document", slug)
			}
		}
	}
	return nil
}

func within(c docs, s section) []doc {
	var out []doc
	for _, slug := range s.Slugs {
		d, _ := c.Get(slug)
		out = append(out, d)
	}
	return out
}

func mount(e *quire.Engine) error {
	pages, err := e.AddCollection(model.Basic{
		ID:          "docs",
		Title:       "Documentation",
		Description: "How to build a site with Quire, and why it is shaped the way it is.",
		Kicker: func(d doc) string {
			_, title := place(d.Slug)
			return title
		},
		Sort: func(a, b doc) int {
			i, _ := place(a.Slug)
			j, _ := place(b.Slug)
			return cmp.Compare(i, j)
		},
		Index: docsIndex,
		Item:  docsPage,
	})
	if err != nil {
		return err
	}
	if err := listed(pages); err != nil {
		return err
	}
	sample, err := e.Markdown("home/sample.md")
	if err != nil {
		return err
	}
	e.AddPage("/", home(e.Shell, sample))
	s := search.New()
	s.Add(pages)
	return e.Use(plugin.At("/search", s))
}

func main() {
	quire.Run(model.Site{
		Name:        "Quire",
		BaseURL:     "https://quire.hiddenbridge.io",
		Description: "A Go engine that turns a folder of markdown into a finished website, served from a single binary.",
	}, mount,
		quire.WithNavEnd(repoLink()),
		quire.WithFooter(footer),
		quire.WithStylesheets(css, "site.css"),
		quire.WithStylesheetsAt(model.WidthWide, css, "site-wide.css"),
	)
}
