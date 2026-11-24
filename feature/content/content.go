// Package content makes a site from its content folder: index.md is the home page, a file beside it is a page, and a folder is a collection.
// It draws nothing: a look hands it how each kind of page is drawn.
package content

import (
	"cmp"
	"fmt"
	"maps"
	"path"
	"slices"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/site"
)

// Pages is how each kind of page is drawn, and the feature that adds them.
type Pages struct {
	Page     func(model.Document) templ.Component
	Listing  func(*model.Collection) templ.Component
	Document func(*model.Collection, model.Document) templ.Component
}

// home is the frontmatter of index.md: a document's, and the order of the navigation.
type home struct {
	model.Meta `yaml:",inline"`
	Nav        []string `yaml:"nav"`
}

func has(s *site.Site, name string) bool {
	found, _ := s.Names(name)
	return len(found) == 1
}

func (p Pages) home(s *site.Site) ([]string, error) {
	if !has(s, "index.md") {
		return nil, nil
	}
	h, body, err := s.Document[home]("index.md")
	if err != nil {
		return nil, err
	}
	d := model.Document{Slug: "index", Path: "index.md", Meta: h.Meta, HTML: body}
	if h.Picture != "" {
		if d.Picture, err = s.Picture(model.Resolve(d.Path, h.Picture)); err != nil {
			return nil, fmt.Errorf("index.md: %w", err)
		}
	}
	s.AddPage(model.PageSpec{Title: cmp.Or(h.Title, s.Identity().Name), Description: h.Summary, Path: "/"}, p.Page(d))
	return h.Nav, nil
}

func (p Pages) collection(s *site.Site, id string) error {
	spec := model.Spec{ID: id}
	if index := id + "/index.md"; has(s, index) {
		m, _, err := s.Document[model.Meta](index)
		if err != nil {
			return err
		}
		spec.Title, spec.Description = m.Title, m.Summary
	}
	c, err := s.Collection(spec)
	if err != nil {
		return err
	}
	s.AddCollectionPages(c, p.Listing(c), func(d model.Document) templ.Component { return p.Document(c, d) })
	return nil
}

func (Pages) Name() string { return "" }

func (p Pages) Declare(s *site.Site) error {
	nav, err := p.home(s)
	if err != nil {
		return err
	}
	loose, err := s.Collection(model.Spec{ID: "pages", Markdown: "*.md"})
	if err != nil {
		return err
	}
	add := map[string]func() error{}
	for _, d := range loose.Docs() {
		add[d.Slug] = func() error {
			s.AddPage(model.PageSpec{Title: d.Meta.Title, Description: d.Meta.Summary, Path: "/" + d.Slug}, p.Page(d))
			return nil
		}
	}
	inFolders, err := s.Names("*/*.md")
	if err != nil {
		return err
	}
	for _, name := range inFolders {
		id := path.Dir(name)
		if _, page := loose.Get(id); page {
			return fmt.Errorf("%s.md and the folder %s are both at /%s", id, id, id)
		}
		add[id] = func() error { return p.collection(s, id) }
	}
	for _, name := range nav {
		if add[name] == nil {
			return fmt.Errorf("index.md lists %q in nav, which is neither a page nor a folder of documents", name)
		}
	}
	for _, name := range slices.Compact(append(nav, slices.Sorted(maps.Keys(add))...)) {
		if next := add[name]; next != nil {
			delete(add, name)
			if err := next(); err != nil {
				return err
			}
		}
	}
	return nil
}
