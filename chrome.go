package quire

import (
	"context"
	"io"
	"io/fs"
	"mime"
	"path"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/internal/fonts"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/ui"
)

const (
	stylePath  = assetPrefix + "style.css"
	scriptPath = assetPrefix + "quire.js"
	javascript = "application/javascript; charset=utf-8"
	css        = "text/css; charset=utf-8"
)

func (s *settings) addSheets(layer string, at model.Width, fsys fs.FS, names []string) {
	for _, n := range names {
		s.sheets = append(s.sheets, assets.File{FS: fsys, Name: n, Layer: layer, At: at})
	}
}

func (e *Engine) addAssets() {
	p := assets.Parts{Scale: e.scale, Palette: e.palette, Palettes: e.chrome.Palettes, Sheets: e.sheets, Scripts: e.scripts}
	p.Fonts, e.chrome.FontPreload = fonts.Faces(e.fonts, fonts.Default(), func(file string, body []byte) string {
		return e.addBytes(assetPrefix+"fonts/"+file, "font/woff2", body)
	})
	e.chrome.StyleURLs = []string{e.addBytes(stylePath, css, assets.Stylesheet(p))}
	e.chrome.ScriptURL = e.addBytes(scriptPath, javascript, assets.Script(p))
	e.chrome.ModuleURLs = []string{e.addBytes(assetPrefix+"datastar.js", javascript, assets.Datastar())}
}

func (e *Engine) addIcons() {
	for _, i := range e.icons {
		rel := "icon"
		if strings.HasPrefix(path.Base(i.Name), "apple-touch-icon") {
			rel = "apple-touch-icon"
		}
		e.chrome.Icons = append(e.chrome.Icons, model.Icon{Rel: rel, Type: mime.TypeByExtension(path.Ext(i.Name)), Href: e.AddFile("/"+i.Name, i.FS, i.Name)})
	}
}

func (e *Engine) addNavItem(id string) {
	href, label := "/"+id, model.Label(id)
	if id == "/" {
		href, label = "/", "Home"
	} else if rest, ok := strings.CutPrefix(id, "/"); ok {
		href, label = id, model.Label(rest)
	}
	item := model.NavItem{Label: label, Href: href}
	switch {
	case slices.ContainsFunc(e.chrome.Nav, func(n model.NavItem) bool { return n.Href == href }):
	case href == "/":
		e.chrome.Nav = slices.Insert(e.chrome.Nav, 0, item)
	default:
		e.chrome.Nav = append(e.chrome.Nav, item)
	}
}

func (e *Engine) late(page func(model.Chrome) templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(e.chrome).Render(ctx, w)
	})
}

func (e *Engine) Site() model.Site { return e.chrome.Site }

func (e *Engine) AddStylesheets(fsys fs.FS, names ...string) { e.AddStylesheetsAt(0, fsys, names...) }

func (e *Engine) AddStylesheetsAt(w model.Width, fsys fs.FS, names ...string) {
	mustBeOpen(e.mounted, "a stylesheet")
	e.addSheets("pack", w, fsys, names)
}

func (e *Engine) AddNavEnd(cs ...templ.Component) {
	mustBeOpen(e.mounted, "a nav control")
	e.chrome.NavEnd = append(e.chrome.NavEnd, cs...)
}

func (e *Engine) SetFooter(f func(model.Chrome, model.PageSpec) templ.Component) {
	mustBeOpen(e.mounted, "the footer")
	e.chrome.Footer = f
}

func (e *Engine) SetFooterText(text string) {
	mustBeOpen(e.mounted, "the footer")
	e.chrome.FooterText = text
}

func (e *Engine) AddFooterLinks(links ...model.Link) {
	mustBeOpen(e.mounted, "a footer link")
	e.chrome.FooterLinks = append(e.chrome.FooterLinks, links...)
}

func (e *Engine) Shell(p model.PageSpec) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Shell(e.chrome, p).Render(ctx, w)
	})
}
