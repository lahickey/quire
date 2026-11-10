package quire

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/ui"
)

type resolver struct{ e *Engine }

func (e *Engine) addImage(name string, max ...model.Slot) (model.Image, error) {
	name = path.Clean(filepath.ToSlash(name))
	want := model.SlotDefault
	if len(max) > 0 && max[0] >= model.SlotThumb {
		want = max[0]
	}
	if im, ok := e.bundle.Mounted(name, want); ok {
		return im, nil
	}
	r, err := e.from.Master(name)
	if err != nil {
		return model.Image{}, err
	}
	if len(r.Formats) == 0 {
		return model.Image{}, fmt.Errorf("quire: image %s names no formats", name)
	}
	im := model.NewImage(r, e.scale, want, e.slots)
	e.bundle.Mount(name, im, want)
	return im, nil
}

func (r resolver) picture(name, alt, caption string, figure bool) (string, error) {
	im, err := r.e.addImage(name)
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

func (r resolver) embed(widgets map[string]model.Widget) func(journal.Embed) (string, error) {
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
			return r.picture(em.Picture, em.Alt, em.Caption, em.Figure)
		case "mermaid":
			svg, err := r.e.from.Diagram(em.Body)
			return `<figure class="diagram">` + svg + "</figure>", err
		}
		return "", fmt.Errorf("no widget %q; rebake", em.Fence)
	}
}

func (r resolver) Documents(pattern string, widgets map[string]model.Widget) ([]journal.Document, error) {
	docs, err := r.e.from.Documents(pattern, slices.Sorted(maps.Keys(widgets)))
	if err != nil {
		return nil, err
	}
	for i := range docs {
		if docs[i].HTML, err = docs[i].Fill(r.embed(widgets)); err != nil {
			return nil, err
		}
	}
	return docs, nil
}

func (r resolver) Image(name string, max model.Slot) (model.Image, error) {
	return r.e.addImage(name, max)
}

func (r resolver) Data(name string) ([]byte, error) { return r.e.from.Data(name) }

func (e *Engine) AddCollection[T model.Metadata](spec model.Spec[T]) (*model.Collection[T], error) {
	col, err := model.Collect(spec, resolver{e}, e.chrome.Site, e.drafts)
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
	if spec.ListOnly {
		return col, nil
	}
	e.route("GET /"+spec.ID+"/{slug}", ownerSite, e.pages.Cached(func(w http.ResponseWriter, r *http.Request) {
		doc, ok := col.Get(r.PathValue("slug"))
		if !ok {
			e.notFound404.ServeHTTP(w, r)
			return
		}
		ui.Render(item(e.Shell, col, doc))(w, r)
	}))
	for _, d := range docs {
		if !d.Meta.IsDraft() {
			e.sitemap = append(e.sitemap, serve.Loc{Path: col.Href(d), LastMod: d.Meta.LastMod().ISO()})
		}
	}
	return col, nil
}
