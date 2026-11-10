package quire

import (
	"cmp"
	"context"
	"fmt"
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
)

func (e *Engine) addImage(name string) (model.Image, error) {
	name = path.Clean(filepath.ToSlash(name))
	if im, ok := e.images[name]; ok {
		return im, nil
	}
	r, err := e.from.Master(name)
	if err != nil {
		return model.Image{}, err
	}
	if len(r.Formats) == 0 {
		return model.Image{}, fmt.Errorf("quire: image %s names no formats", name)
	}
	im := model.NewImage(r, e.scale)
	e.images[name] = im
	return im, nil
}

func (e *Engine) picture(name, alt, caption string, figure bool) (string, error) {
	im, err := e.addImage(name)
	if err != nil || im.Hash == "" {
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

func (e *Engine) embed(widgets map[string]model.Widget) func(journal.Embed) (string, error) {
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
			return e.picture(em.Picture, em.Alt, em.Caption, em.Figure)
		case "mermaid":
			svg, err := e.from.Diagram(em.Body)
			return `<figure class="diagram">` + svg + "</figure>", err
		}
		return "", fmt.Errorf("no widget %q; rebake", em.Fence)
	}
}

func (e *Engine) documents(pattern string, widgets map[string]model.Widget) ([]journal.Document, error) {
	docs, err := e.from.Documents(pattern, slices.Sorted(maps.Keys(widgets)))
	if err != nil {
		return nil, err
	}
	for i := range docs {
		if docs[i].HTML, err = docs[i].Fill(e.embed(widgets)); err != nil {
			return nil, err
		}
	}
	return docs, nil
}

func (e *Engine) Markdown(name string) (templ.Component, error) {
	docs, err := e.documents(name, nil)
	if err != nil {
		return nil, err
	}
	if len(docs) != 1 {
		return nil, fmt.Errorf("quire: markdown %s: no such file under the content root", name)
	}
	return templ.Raw(docs[0].HTML), nil
}

func (e *Engine) AddCollection[T model.Metadata](spec model.Spec[T]) (*model.Collection[T], error) {
	baked, err := e.documents(cmp.Or(spec.Markdown, spec.ID+"/*.md"), spec.Widgets)
	if err != nil {
		return nil, err
	}
	col, err := model.Collect(spec, baked, e.addImage, e.chrome.Site, e.drafts)
	if err != nil {
		return nil, err
	}
	docs := col.Docs()
	if len(docs) == 0 {
		return col, nil
	}
	e.addNavItem(spec.ID)

	index, item := spec.Index, spec.Item
	if index == nil {
		index = ui.Listing[T]
	}
	if item == nil {
		item = ui.Article[T]
	}
	e.preparePage("/"+spec.ID, index(e.Shell, col))

	e.sitemap = append(e.sitemap, serve.Loc{Path: "/" + spec.ID})
	e.route("GET /"+spec.ID+"/{slug}", ownerSite, e.pages.Cached(func(w http.ResponseWriter, r *http.Request) {
		doc, ok := col.Get(r.PathValue("slug"))
		if !ok {
			e.notFound404.ServeHTTP(w, r)
			return
		}
		ui.Render(item(e.Shell, col, doc))(w, r)
	}))
	for _, d := range docs {
		if !d.Meta.Base().Draft {
			e.sitemap = append(e.sitemap, serve.Loc{Path: col.Href(d), LastMod: d.Meta.Base().LastMod().ISO()})
		}
	}
	return col, nil
}
