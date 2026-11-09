package quire

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/bake"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
	"github.com/lahickey/quire/ui"
)

type host struct{ e *Engine }

func (m host) picture(name, alt, caption string, figure bool) (string, error) {
	im, err := m.e.addImage(name)
	if _, baking := m.e.from.(*bake.Run); err != nil || baking {
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

func (m host) embed(widgets map[string]model.Widget) func(journal.Embed) (string, error) {
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
			return m.picture(em.Picture, em.Alt, em.Caption, em.Figure)
		case "mermaid":
			svg, err := m.e.from.Diagram(em.Body)
			return `<figure class="diagram">` + svg + "</figure>", err
		}
		return "", fmt.Errorf("no widget %q; rebake", em.Fence)
	}
}

func (m host) Documents(pattern string, widgets map[string]model.Widget) ([]journal.Document, error) {
	docs, err := m.e.from.Documents(pattern, slices.Sorted(maps.Keys(widgets)))
	if err != nil {
		return nil, err
	}
	for i := range docs {
		if docs[i].HTML, err = docs[i].Fill(m.embed(widgets)); err != nil {
			return nil, err
		}
	}
	return docs, nil
}

func (m host) Image(name string, max model.Slot) (model.Image, error) { return m.e.addImage(name, max) }

func (m host) Data(name string) ([]byte, error) { return m.e.from.Data(name) }

func reserved(path string) bool {
	return strings.HasPrefix(path, "/_")
}

func (h host) Route(method, path string, f http.HandlerFunc) { h.e.AddRoute(method, path, f) }

func (h host) Claim(method, path, owner string, f http.HandlerFunc) error {
	if reserved(path) && !strings.HasPrefix(path+"/", "/_"+owner+"/") {
		return fmt.Errorf("%s belongs to another namespace", path)
	}
	pattern := method + " " + pagePattern(path)
	if o, taken := h.e.claims[pattern]; taken {
		return fmt.Errorf("%s is already mounted by %s", path, o)
	}
	if h.e.claimed(pattern) {
		return fmt.Errorf("%s is already mounted by the site", path)
	}
	h.e.claims[pattern] = owner
	h.e.AddRoute(method, pagePattern(path), f)
	return nil
}

func (h host) AddStylesheets(at model.Width, fsys fs.FS, names ...string) {
	mustBeOpen(h.e.linked.built, "stylesheet")
	h.e.addSheets("plugin", at, fsys, names)
}

func (h host) AddNavEnd(c templ.Component) {
	mustBeOpen(h.e.mounted, "nav control")
	if c != nil {
		h.e.navEnd = append(h.e.navEnd, c)
		h.e.chromeOK = false
	}
}

func (h host) UseSessions() { h.e.sessions = true }

func (h host) Shell(p model.PageSpec) templ.Component { return h.e.Shell(p) }

func (e *Engine) Use(plugins ...plugin.Plugin) error {
	for _, p := range plugins {
		if err := plugin.Attach(host{e}, p); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) secure() bool {
	return e.mode != Dev && strings.HasPrefix(e.site.BaseURL, "https://")
}
