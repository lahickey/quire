package quire

import (
	"context"
	"io"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/internal/fonts"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/ui"
)

func (e *Engine) sectionNav() []model.NavItem {
	out := make([]model.NavItem, 0, len(e.navItems))
	for _, n := range e.navItems {
		if n.Href != "/" {
			out = append(out, n)
		}
	}
	return out
}

const (
	stylePath  = assetPrefix + "style.css"
	scriptPath = assetPrefix + "quire.js"
)

func (e *Engine) buildAssets() {
	if e.linked.built {
		return
	}
	e.linked.built = true
	p := e.linked.parts
	p.Scale, p.Palette = e.scale, e.palette
	p.Fonts, e.fontPreload = fonts.Faces(e.fonts, fonts.Default(), func(file string, body []byte) string {
		return e.serveBytes(assetPrefix+"fonts/"+file, "font/woff2", body)
	})
	e.linked.styles = []string{e.serveBytes(stylePath, css, assets.Stylesheet(p))}
	e.linked.script = e.serveBytes(scriptPath, javascript, assets.Script())
	e.linked.modules = []string{e.serveBytes(assetPrefix+"datastar.js", javascript, assets.Datastar())}
}

func (e *Engine) cardURL(name string) (string, bool) {
	rec, _ := e.bundle.Record(name)
	file, ok := rec.File()
	if !ok {
		return "", false
	}
	return model.ImageURL(file), true
}

func (e *Engine) buildChrome() model.Chrome {
	e.buildAssets()
	c := model.Chrome{
		Site:        e.site,
		Theme:       e.theme,
		Header:      e.header,
		Footer:      e.footer,
		ScriptURL:   e.linked.script,
		ModuleURLs:  e.linked.modules,
		Controls:    slices.Clone(e.navEnd),
		StyleURLs:   e.linked.styles,
		FontPreload: e.fontPreload,
		LiveReload:  e.liveReload(),
		Domain:      e.site.Host(),
		Nav:         slices.Clone(e.navItems),
		Sections:    e.sectionNav(),
	}
	if c.Footer == nil {
		c.Footer = ui.Footer(c.Name)
	}
	for _, i := range e.icons {
		c.Icons = append(c.Icons, i.Icon)
	}
	if u, ok := e.cardURL(journal.SiteCard); ok {
		c.OGImageURL = c.AbsURL(u)
	}
	return c
}

func (e *Engine) currentChrome() model.Chrome {
	if !e.chromeOK {
		e.chrome = e.buildChrome()
		e.chromeOK = true
	}
	return e.chrome
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
		e.chromeOK = false
		return
	}
	e.navItems = append(e.navItems, model.NavItem{Label: label, Href: href})
	e.chromeOK = false
}

func (e *Engine) Shell(p model.PageSpec) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Shell(e.currentChrome(), p).Render(ctx, w)
	})
}

func (e *Engine) late(page func(model.Chrome) templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(e.currentChrome()).Render(ctx, w)
	})
}

type linked struct {
	parts   assets.Parts
	script  string
	styles  []string
	modules []string
	hashes  map[string]string
	built   bool
}

func (e *Engine) loadFonts() {
	set, err := fonts.ReadFontSet(fonts.Default())
	if err != nil {
		panic("quire: fonts: " + err.Error())
	}
	e.fonts = set
}
