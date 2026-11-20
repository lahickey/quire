package quire

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/ui"
	"go.yaml.in/yaml/v3"
)

func load[T any](baked []journal.Document) ([]model.Document[T], error) {
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
		headings := make([]model.Heading, len(b.Headings))
		for i, h := range b.Headings {
			headings[i] = model.Heading(h)
		}
		out = append(out, model.Document[T]{Slug: slug, Path: b.Path, Meta: meta, HTML: b.HTML, Words: b.Words, Headings: headings})
	}
	return out, nil
}

func (s *Site) addImage(name string) (model.Image, error) {
	name = path.Clean(filepath.ToSlash(name))
	if im, ok := s.images[name]; ok {
		return im, nil
	}
	r, err := s.from.Master(name)
	if err != nil {
		return model.Image{}, err
	}
	files := map[int]string{}
	for _, v := range r.Files {
		files[v.W] = v.Name
	}
	m, base := r.Picture.Master, path.Base(name)
	im := model.Image{Name: strings.TrimSuffix(base, path.Ext(base)), Hash: m.Hash, Width: m.W, Height: m.H, Alpha: m.Alpha, Files: files, Scale: s.scale}
	s.images[name] = im
	return im, nil
}

func (s *Site) picture(name, alt, caption string, figure bool) (string, error) {
	im, err := s.addImage(name)
	if err != nil {
		return "", err
	}
	c := ui.Img(im, model.SlotPage, alt)
	if figure {
		c = ui.LazyFigure(im, alt, caption)
	}
	var b strings.Builder
	err = c.Render(context.Background(), &b)
	return b.String(), err
}

func (s *Site) video(name, alt, caption string) (string, error) {
	rec, err := s.from.Video(name)
	if err != nil {
		return "", err
	}
	poster, err := s.addImage(strings.TrimSuffix(name, path.Ext(name)) + ".png")
	if err != nil {
		return "", fmt.Errorf("%s needs a still of the same name as its poster: %w", name, err)
	}
	file, _ := rec.File()
	var b strings.Builder
	err = ui.VideoFigure(model.ImageURL(file), poster, alt, caption).Render(context.Background(), &b)
	return b.String(), err
}

func cards(body string) (string, error) {
	var written []struct{ Href, Kicker, Title, Description, CTA string }
	dec := yaml.NewDecoder(strings.NewReader(body))
	dec.KnownFields(true)
	if err := dec.Decode(&written); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	list := make([]ui.CardData, len(written))
	for i, c := range written {
		list[i] = ui.CardData{Href: c.Href, Kicker: c.Kicker, Title: c.Title, Description: c.Description, CTA: c.CTA}
	}
	var b strings.Builder
	err := ui.CardGrid(list).Render(context.Background(), &b)
	return b.String(), err
}

func (s *Site) embed(widgets map[string]model.Widget) func(journal.Embed) (string, error) {
	return func(em journal.Embed) (string, error) {
		if w, ok := widgets[em.Fence]; ok {
			html, err := w(em.Info, em.Body)
			if err != nil {
				return "", fmt.Errorf("%s: %w", em.Fence, err)
			}
			return html, nil
		}
		switch em.Fence {
		case "":
			if journal.VideoType(em.Picture) == "" {
				return s.picture(em.Picture, em.Alt, em.Caption, em.Figure)
			}
			if !em.Figure {
				return "", fmt.Errorf("%s: a video goes in a paragraph of its own", em.Picture)
			}
			return s.video(em.Picture, em.Alt, em.Caption)
		case "cards":
			html, err := cards(em.Body)
			if err != nil {
				return "", fmt.Errorf("cards: %w", err)
			}
			return html, nil
		case "mermaid":
			svg, err := s.from.Diagram(em.Body)
			return `<figure class="diagram">` + svg + "</figure>", err
		}
		return "", fmt.Errorf("no widget %q; rebake", em.Fence)
	}
}

func (s *Site) documents(pattern string, widgets map[string]model.Widget) ([]journal.Document, error) {
	docs, err := s.from.Documents(pattern, slices.Sorted(maps.Keys(widgets)))
	if err != nil {
		return nil, err
	}
	for i := range docs {
		if docs[i].HTML, err = docs[i].Fill(s.embed(widgets)); err != nil {
			return nil, err
		}
	}
	return docs, nil
}

func (s *Site) Document[T any](name string) (model.Document[T], error) {
	baked, err := s.documents(name, nil)
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
	baked, err := s.documents(cmp.Or(spec.Markdown, spec.ID+"/*.md"), spec.Widgets)
	if err != nil {
		return nil, err
	}
	docs, err := load[T](baked)
	if err != nil {
		return nil, err
	}
	for i, d := range docs {
		name := d.Meta.Base().Image
		if name == "" {
			continue
		}
		if docs[i].Image, err = s.addImage(journal.Resolve(d.Path, name)); err != nil {
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
