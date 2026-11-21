package quire

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"go.yaml.in/yaml/v3"
)

func load[T any](baked []model.BakedDocument) ([]model.Document[T], error) {
	out := make([]model.Document[T], 0, len(baked))
	bySlug := map[string]string{}
	for _, b := range baked {
		var meta T
		dec := yaml.NewDecoder(strings.NewReader(b.Front))
		dec.KnownFields(true)
		if err := dec.Decode(&meta); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s: frontmatter: %w", b.Path, err)
		}
		slug := strings.TrimSuffix(path.Base(b.Path), path.Ext(b.Path))
		if other, taken := bySlug[slug]; taken {
			return nil, fmt.Errorf("%s and %s share the slug %q", other, b.Path, slug)
		}
		bySlug[slug] = b.Path
		out = append(out, model.Document[T]{Slug: slug, Path: b.Path, Meta: meta, HTML: b.HTML, Words: b.Words, Headings: b.Headings})
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

func (s *Site) Document[T any](name string) (model.Document[T], error) {
	baked, err := s.documents(name)
	if err == nil && len(baked) != 1 {
		err = fmt.Errorf("quire: markdown %s: no such file under the content root", name)
	}
	if err != nil {
		return model.Document[T]{}, err
	}
	docs, err := load[T](baked)
	if err != nil {
		return model.Document[T]{}, err
	}
	return docs[0], nil
}

func (s *Site) Markdown(name string) (templ.Component, error) {
	d, err := s.Document[map[string]any](name)
	return templ.Raw(d.HTML), err
}

func (s *Site) Collection[T model.Metadata](spec model.Spec[T]) (*model.Collection[T], error) {
	baked, err := s.documents(cmp.Or(spec.Markdown, spec.ID+"/*.md"))
	if err != nil {
		return nil, err
	}
	docs, err := load[T](baked)
	if err != nil {
		return nil, err
	}
	for i, d := range docs {
		name := d.Meta.Base().Picture
		if name == "" {
			continue
		}
		if docs[i].Picture, err = s.Picture(model.Resolve(d.Path, name)); err != nil {
			return nil, fmt.Errorf("%s/%s: %w", spec.ID, d.Slug, err)
		}
	}
	col, err := model.Collect(spec, docs, s.drafts)
	if err != nil {
		return nil, err
	}
	return col, nil
}

// AddDocumentPages gives each of a collection's documents a page at its address, with its aliases leading there and its place in the sitemap.
// A page is drawn on its first request and kept.
func (s *Site) AddDocumentPages[T model.Metadata](col *model.Collection[T], page func(model.Document[T]) (model.PageSpec, templ.Component)) {
	s.route("GET "+col.Path()+"/{slug}", ownerSite, func(w http.ResponseWriter, r *http.Request) {
		doc, ok := col.Get(r.PathValue("slug"))
		if !ok {
			s.notFound404.ServeHTTP(w, r)
			return
		}
		render(s.Page(page(doc)))(w, r)
	})
	s.handlers[len(s.handlers)-1].cached = true
	for _, d := range col.Docs() {
		href := col.Href(d)
		s.err = cmp.Or(s.err, s.reserve("GET "+href, ownerSite))
		for _, alias := range d.Meta.Base().Aliases {
			if !strings.HasPrefix(alias, "/") {
				s.err = cmp.Or(s.err, fmt.Errorf("%s: alias %q must start with /", d.Path, alias))
				continue
			}
			if err := s.claim("GET "+alias, ownerSite, http.RedirectHandler(href, http.StatusMovedPermanently).ServeHTTP); err != nil {
				s.err = cmp.Or(s.err, fmt.Errorf("%s: alias %w", d.Path, err))
			}
		}
		if !d.Meta.Base().Draft {
			s.sitemap = append(s.sitemap, serve.Loc{Path: href, LastMod: d.Meta.Base().LastMod().ISO()})
		}
	}
}
