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

func (s *Site) addAssets() {
	p := assets.Parts{Scale: s.scale, Palette: s.palette, Palettes: s.chrome.Palettes, Sheets: s.sheets, Scripts: s.scripts}
	p.Fonts, s.chrome.FontPreload = fonts.Faces(s.fonts, fonts.Default(), func(file string, body []byte) string {
		return s.addBytes(assetPrefix+"fonts/"+file, "font/woff2", body)
	})
	s.chrome.StyleURLs = []string{s.addBytes(stylePath, css, assets.Stylesheet(p))}
	s.chrome.ScriptURL = s.addBytes(scriptPath, javascript, assets.Script(p))
	s.chrome.ModuleURLs = []string{s.addBytes(assetPrefix+"datastar.js", javascript, assets.Datastar())}
}

func (s *Site) addIcons() {
	for _, i := range s.icons {
		rel := "icon"
		if strings.HasPrefix(path.Base(i.Name), "apple-touch-icon") {
			rel = "apple-touch-icon"
		}
		s.chrome.Icons = append(s.chrome.Icons, model.Icon{Rel: rel, Type: mime.TypeByExtension(path.Ext(i.Name)), Href: s.AddFile("/"+i.Name, i.FS, i.Name)})
	}
}

func (s *Site) addNavItem(id string) {
	href, label := "/"+id, model.Label(id)
	if id == "/" {
		href, label = "/", "Home"
	} else if rest, ok := strings.CutPrefix(id, "/"); ok {
		href, label = id, model.Label(rest)
	}
	item := model.NavItem{Label: label, Href: href}
	switch {
	case slices.ContainsFunc(s.chrome.Nav, func(n model.NavItem) bool { return n.Href == href }):
	case href == "/":
		s.chrome.Nav = slices.Insert(s.chrome.Nav, 0, item)
	default:
		s.chrome.Nav = append(s.chrome.Nav, item)
	}
}

func (s *Site) late(page func(model.Chrome) templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return page(s.chrome).Render(ctx, w)
	})
}

func (s *Site) Identity() model.Identity { return s.chrome.Identity }

func (s *Site) AddStylesheets(fsys fs.FS, names ...string) { s.AddStylesheetsAt(0, fsys, names...) }

func (s *Site) AddStylesheetsAt(w model.Width, fsys fs.FS, names ...string) {
	mustBeOpen(s.built, "a stylesheet")
	s.addSheets("pack", w, fsys, names)
}

func (s *Site) AddNavEnd(cs ...templ.Component) {
	mustBeOpen(s.built, "a nav control")
	s.chrome.NavEnd = append(s.chrome.NavEnd, cs...)
}

func (s *Site) SetFooter(f func(model.Chrome, model.PageSpec) templ.Component) {
	mustBeOpen(s.built, "the footer")
	s.chrome.Footer = f
}

func (s *Site) SetFooterText(text string) {
	mustBeOpen(s.built, "the footer")
	s.chrome.FooterText = text
}

func (s *Site) AddFooterLinks(links ...model.Link) {
	mustBeOpen(s.built, "a footer link")
	s.chrome.FooterLinks = append(s.chrome.FooterLinks, links...)
}

func (s *Site) Shell(p model.PageSpec) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return ui.Shell(s.chrome, p).Render(ctx, w)
	})
}
