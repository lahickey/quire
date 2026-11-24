package site

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
	"go.yaml.in/yaml/v3"
)

func front[T any](b model.BakedDocument) (T, error) {
	var v T
	dec := yaml.NewDecoder(strings.NewReader(b.Front))
	dec.KnownFields(true)
	if err := dec.Decode(&v); err != nil && !errors.Is(err, io.EOF) {
		return v, fmt.Errorf("%s: frontmatter: %w", b.Path, err)
	}
	return v, nil
}

func load(baked []model.BakedDocument) ([]model.Document, error) {
	out := make([]model.Document, 0, len(baked))
	bySlug := map[string]string{}
	for _, b := range baked {
		meta, err := front[model.Meta](b)
		if err != nil {
			return nil, err
		}
		slug := strings.TrimSuffix(path.Base(b.Path), path.Ext(b.Path))
		if other, taken := bySlug[slug]; taken {
			return nil, fmt.Errorf("%s and %s share the slug %q", other, b.Path, slug)
		}
		bySlug[slug] = b.Path
		out = append(out, model.Document{Slug: slug, Path: b.Path, Meta: meta, HTML: b.HTML, Words: b.Words, Headings: b.Headings})
	}
	return out, nil
}

func (s *Site) documents(pattern string) ([]model.BakedDocument, error) {
	names, err := s.from.Names(pattern)
	if err != nil {
		return nil, err
	}
	s.read = true
	docs := make([]model.BakedDocument, len(names))
	for i, name := range names {
		if docs[i], err = s.from.Document(name); err != nil {
			return nil, err
		}
		if docs[i].HTML, err = docs[i].Fill(s.fill); err != nil {
			return nil, err
		}
	}
	return docs, nil
}

// Names is the documents under the content root that match a pattern.
func (s *Site) Names(pattern string) ([]string, error) { return s.from.Names(pattern) }

// Document reads one file under the content root: its frontmatter into T, and its body as HTML.
func (s *Site) Document[T any](name string) (T, string, error) {
	baked, err := s.documents(name)
	if err == nil && len(baked) != 1 {
		err = fmt.Errorf("quire: markdown %s: no such file under the content root", name)
	}
	if err != nil {
		var none T
		return none, "", err
	}
	v, err := front[T](baked[0])
	return v, baked[0].HTML, err
}

func (s *Site) Markdown(name string) (templ.Component, error) {
	_, html, err := s.Document[map[string]any](name)
	return templ.Raw(html), err
}

func (s *Site) Collection(spec model.Spec) (*model.Collection, error) {
	baked, err := s.documents(cmp.Or(spec.Markdown, spec.ID+"/*.md"))
	if err != nil {
		return nil, err
	}
	docs, err := load(slices.DeleteFunc(baked, func(b model.BakedDocument) bool { return path.Base(b.Path) == "index.md" }))
	if err != nil {
		return nil, err
	}
	for i, d := range docs {
		name := d.Meta.Picture
		if name == "" {
			continue
		}
		if docs[i].Picture, err = s.Picture(model.Resolve(d.Path, name)); err != nil {
			return nil, fmt.Errorf("%s/%s: %w", spec.ID, d.Slug, err)
		}
	}
	return model.Collect(spec, docs, s.drafts)
}

// AddCollectionPages gives a collection its listing, at its address, and a page for each document, with the head data the collection has for each.
// A collection with nothing in it, as when every document is a draft, gets neither.
func (s *Site) AddCollectionPages(c *model.Collection, listing templ.Component, document func(model.Document) templ.Component) {
	if c.Len() == 0 {
		return
	}
	s.AddPage(c.Page(), listing)
	s.AddDocumentPages(c, func(d model.Document) (model.PageSpec, templ.Component) {
		return model.DocumentPage(c, d), document(d)
	})
}

// AddDocumentPages gives each of a collection's documents a page at its address, with its aliases leading there and its place in the sitemap.
// A page is drawn on its first request and kept.
func (s *Site) AddDocumentPages(col *model.Collection, page func(model.Document) (model.PageSpec, templ.Component)) {
	s.route("GET "+col.Path()+"/{slug}", s.owner, func(w http.ResponseWriter, r *http.Request) {
		doc, ok := col.Get(r.PathValue("slug"))
		if !ok {
			if out, placed := w.(*placedWriter); placed {
				out.markup = nil
			}
			s.notFound404.ServeHTTP(w, r)
			return
		}
		s.render(s.page(page(doc)))(w, r)
	}).Shared = true
	for _, d := range col.Docs() {
		href := col.Href(d)
		if err := s.reserve("GET "+href, s.owner); err != nil {
			s.fail("%w", err)
		}
		for _, alias := range d.Meta.Aliases {
			if !strings.HasPrefix(alias, "/") {
				s.fail("%s: alias %q must start with /", d.Path, alias)
				continue
			}
			if err := s.claim("GET "+alias, s.owner, http.RedirectHandler(href, http.StatusMovedPermanently).ServeHTTP); err != nil {
				s.fail("%s: alias %w", d.Path, err)
			}
		}
		if !d.Meta.Draft {
			s.sitemap = append(s.sitemap, Loc{Path: href, LastMod: d.Meta.LastMod().ISO()})
		}
	}
}
