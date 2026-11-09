package quire

import (
	"context"
	"io"
	"io/fs"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/internal/fonts"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/ui"
)

const (
	stylePath  = assetPrefix + "style.css"
	scriptPath = assetPrefix + "quire.js"
	javascript = "application/javascript; charset=utf-8"
	css        = "text/css; charset=utf-8"
)

type linked struct {
	parts   assets.Parts
	script  string
	styles  []string
	modules []string
	preload []string
}

func (e *Engine) addSheets(layer string, at model.Width, fsys fs.FS, names []string) {
	for _, n := range names {
		e.linked.parts.Sheets = append(e.linked.parts.Sheets, assets.Sheet{FS: fsys, Name: n, Layer: layer, At: at})
	}
}

func (e *Engine) addAssets() {
	p := e.linked.parts
	p.Scale, p.Palette = e.scale, e.palette
	p.Fonts, e.linked.preload = fonts.Faces(e.fonts, fonts.Default(), func(file string, body []byte) string {
		return e.addBytes(assetPrefix+"fonts/"+file, "font/woff2", body)
	})
	e.linked.styles = []string{e.addBytes(stylePath, css, assets.Stylesheet(p))}
	e.linked.script = e.addBytes(scriptPath, javascript, assets.Script())
	e.linked.modules = []string{e.addBytes(assetPrefix+"datastar.js", javascript, assets.Datastar())}
}

func (e *Engine) buildChrome() model.Chrome {
	c := model.Chrome{
		Site:        e.site,
		Theme:       e.theme,
		Header:      e.header,
		Footer:      e.footer,
		ScriptURL:   e.linked.script,
		ModuleURLs:  e.linked.modules,
		NavEnd:      slices.Clone(e.navEnd),
		StyleURLs:   e.linked.styles,
		FontPreload: e.linked.preload,
		Domain:      e.site.Host(),
		Nav:         slices.Clone(e.navItems),
		Sections:    slices.DeleteFunc(slices.Clone(e.navItems), func(n model.NavItem) bool { return n.Href == "/" }),
	}
	if e.mode == Dev {
		c.LiveReload = model.LiveReload{On: true, BootID: e.bootID, Init: reloadInit}
	}
	for _, i := range e.icons {
		c.Icons = append(c.Icons, i.Icon)
	}
	rec, _ := e.bundle.Record(journal.SiteCard)
	if file, ok := rec.File(); ok {
		c.OGImageURL = c.AbsURL(model.ImageURL(file))
	}
	return c
}

func (e *Engine) addNavItem(id string) {
	href, label := "/"+id, model.Label(id)
	if id == "/" {
		href, label = "/", "Home"
	} else if rest, ok := strings.CutPrefix(id, "/"); ok {
		href, label = id, model.Label(rest)
	}
	for _, n := range e.navItems {
		if n.Href == href {
			return
		}
	}
	if href == "/" {
		e.navItems = append([]model.NavItem{{Label: label, Href: href}}, e.navItems...)
		return
	}
	e.navItems = append(e.navItems, model.NavItem{Label: label, Href: href})
}

func (e *Engine) late(page func(model.Chrome) templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(e.chrome).Render(ctx, w)
	})
}

func (e *Engine) Shell(p model.PageSpec) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Shell(e.chrome, p).Render(ctx, w)
	})
}
