package quire

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/ui"
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

func (s *Site) picture(name string) (model.Picture, error) {
	name = path.Clean(filepath.ToSlash(name))
	if pic, ok := s.pictures[name]; ok {
		return pic, nil
	}
	pic, err := s.from.Picture(name)
	if err != nil {
		return pic, err
	}
	base := path.Base(name)
	pic.Name, pic.Scale = strings.TrimSuffix(base, path.Ext(base)), s.scale
	s.pictures[name] = pic
	return pic, nil
}

func (s *Site) placePicture(name, alt, caption string, figure bool) (string, error) {
	pic, err := s.picture(name)
	if err != nil {
		return "", err
	}
	c := ui.Img(pic, model.SizePage, alt)
	if figure {
		c = ui.LazyFigure(pic, alt, caption)
	}
	var b strings.Builder
	err = c.Render(context.Background(), &b)
	return b.String(), err
}

func (s *Site) placeVideo(name, alt, caption string) (string, error) {
	file, err := s.from.File(name)
	if err != nil {
		return "", err
	}
	poster, err := s.picture(strings.TrimSuffix(name, path.Ext(name)) + ".png")
	if err != nil {
		return "", fmt.Errorf("%s needs a still of the same name as its poster: %w", name, err)
	}
	var b strings.Builder
	err = ui.VideoFigure(model.BundleURL(file), poster, alt, caption).Render(context.Background(), &b)
	return b.String(), err
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

func (s *Site) AddCollection[T model.Metadata](spec model.Spec[T]) (*model.Collection[T], error) {
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
		if docs[i].Picture, err = s.picture(model.Resolve(d.Path, name)); err != nil {
			return nil, fmt.Errorf("%s/%s: %w", spec.ID, d.Slug, err)
		}
	}
	col, err := model.Collect(spec, docs, s.chrome.Identity, s.drafts)
	if err != nil {
		return nil, err
	}
	docs = col.Docs()
	if len(docs) == 0 {
		return col, nil
	}
	s.addNavItem(spec.ID)

	index, item := spec.Index, spec.Item
	if index == nil {
		index = ui.Listing[T]
	}
	if item == nil {
		item = ui.Article[T]
	}
	s.preparePage("/"+spec.ID, index(s.Shell, col))

	s.sitemap = append(s.sitemap, serve.Loc{Path: "/" + spec.ID})
	s.route("GET /"+spec.ID+"/{slug}", ownerSite, func(w http.ResponseWriter, r *http.Request) {
		doc, ok := col.Get(r.PathValue("slug"))
		if !ok {
			s.notFound404.ServeHTTP(w, r)
			return
		}
		ui.Render(item(s.Shell, col, doc))(w, r)
	})
	s.handlers[len(s.handlers)-1].cached = true
	for _, d := range docs {
		href := col.Href(d)
		if err := s.reserve("GET "+href, ownerSite); err != nil {
			return nil, err
		}
		for _, alias := range d.Meta.Base().Aliases {
			if !strings.HasPrefix(alias, "/") {
				return nil, fmt.Errorf("%s/%s: alias %q must start with /", spec.ID, d.Slug, alias)
			}
			if err := s.claim("GET "+alias, ownerSite, http.RedirectHandler(href, http.StatusMovedPermanently).ServeHTTP); err != nil {
				return nil, fmt.Errorf("%s/%s: alias %w", spec.ID, d.Slug, err)
			}
		}
		if !d.Meta.Base().Draft {
			s.sitemap = append(s.sitemap, serve.Loc{Path: col.Href(d), LastMod: d.Meta.Base().LastMod().ISO()})
		}
	}
	return col, nil
}
