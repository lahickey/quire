package quire

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"path"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/model"
)

const (
	stylePath  = assetPrefix + "style.css"
	scriptPath = assetPrefix + "quire.js"
	javascript = "application/javascript; charset=utf-8"
	css        = "text/css; charset=utf-8"
)

func (s *Site) addSheets(layer string, at model.Width, fsys fs.FS, names []string) {
	for _, n := range names {
		s.sheets = append(s.sheets, assets.File{FS: fsys, Name: n, Layer: layer, At: at})
	}
}

func (s *Site) addAssets() {
	p := assets.Parts{Scale: s.chrome.Scale, Sheets: s.sheets, Scripts: s.scripts}
	if s.fontFiles != nil {
		p.Fonts, s.chrome.FontPreload = fontCSS(s.fonts, s.fontFiles, func(file string, body []byte) string {
			return s.addBytes(assetPrefix+"fonts/"+file, "font/woff2", body)
		})
	}
	if sheet := assets.Stylesheet(p); len(sheet) > 0 {
		s.chrome.StyleURLs = []string{s.addBytes(stylePath, css, sheet)}
	}
	if script := assets.Script(p); len(script) > 0 {
		s.chrome.ScriptURL = s.addBytes(scriptPath, javascript, script)
	}
	if s.datastar || len(s.plugins) > 0 {
		s.chrome.ModuleURLs = []string{s.addBytes(assetPrefix+"datastar.js", javascript, assets.Datastar())}
	}
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

// Page is a body as a whole page: the site's document round the frame round the body, with the chrome in reach of everything inside.
func (s *Site) Page(p model.PageSpec, body templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return document(s.chrome, p, s.frame.v).Render(templ.WithChildren(model.WithChrome(ctx, &s.chrome), body), w)
	})
}

func (s *Site) Identity() model.Identity { return s.chrome.Identity }

// Chrome is the site as a page sees it, as it stands now. A pack reads the scale and the palettes on offer from it.
func (s *Site) Chrome() model.Chrome { return s.chrome }

// Palette is the site's own palette.
func (s *Site) Palette() model.Palette { return s.palette }

// ShareCard is what the site's share card says: what WithSiteCard gave, or the site's name and description, on its palette.
func (s *Site) ShareCard() model.ShareCard {
	c := model.ShareCard{Title: s.chrome.Name, Lines: []string{s.chrome.Description}}
	if s.siteCard != nil {
		c = *s.siteCard
	}
	c.Domain = cmp.Or(c.Domain, s.chrome.Host())
	if c.Palette == (model.Palette{}) {
		c.Palette = s.palette
	}
	c.Palette.Check()
	return c
}

// SetShareImage is the bundle file every page without a picture of its own shares.
func (s *Site) SetShareImage(file string) {
	mustBeOpen(s.built, "the share image")
	s.chrome.OGImageURL = s.chrome.AbsURL(model.BundleURL(file))
}

// AddStylesheets adds CSS. The site's own is laid over a pack's, whichever came first.
func (s *Site) AddStylesheets(fsys fs.FS, names ...string) { s.AddStylesheetsAt(0, fsys, names...) }

func (s *Site) AddStylesheetsAt(w model.Width, fsys fs.FS, names ...string) {
	mustBeOpen(s.built, "a stylesheet")
	layer := "site"
	if s.depth > 0 {
		layer = "pack"
	}
	s.addSheets(layer, w, fsys, names)
}

func (s *Site) AddNavEnd(cs ...templ.Component) {
	mustBeOpen(s.built, "a nav control")
	s.chrome.NavEnd = append(s.chrome.NavEnd, cs...)
}

func (s *Site) SetFooterText(text string) {
	mustBeOpen(s.built, "the footer")
	s.footerText.set(text, s.depth == 0)
}

func (s *Site) AddFooterLinks(links ...model.Link) {
	mustBeOpen(s.built, "a footer link")
	s.chrome.FooterLinks = append(s.chrome.FooterLinks, links...)
}

// AddBaseCSS adds CSS to the bottom layer, beneath every plugin, pack and the site: the look everything else is laid over.
// It takes the CSS itself, since some of it is made and not read, such as what a palette becomes.
func (s *Site) AddBaseCSS(name string, body []byte) {
	mustBeOpen(s.built, "a stylesheet")
	s.sheets = append(s.sheets, assets.File{Name: name, Body: body, Layer: "base"})
}

// AddScripts adds JavaScript to the site's one script, each file in a scope of its own.
func (s *Site) AddScripts(fsys fs.FS, names ...string) {
	mustBeOpen(s.built, "a script")
	for _, n := range names {
		s.scripts = append(s.scripts, assets.File{FS: fsys, Name: n})
	}
}

// SetFonts is the font set the site draws in: a description, fonts.json, and the faces it names.
func (s *Site) SetFonts(fsys fs.FS) {
	mustBeOpen(s.built, "the fonts")
	set, err := model.ReadFontSet(fsys)
	if err != nil {
		s.err = cmp.Or(s.err, fmt.Errorf("fonts: %w", err))
		return
	}
	s.fonts, s.fontFiles = set, fsys
}

// SetDatastar links Datastar on every page, for a look whose controls need it. A site with a plugin has it anyway.
func (s *Site) SetDatastar(on bool) { s.datastar = on }

// SetFrame draws what goes round every page's content. Without one a page is its content and nothing else.
func (s *Site) SetFrame(f Frame) {
	mustBeOpen(s.built, "the frame")
	s.frame.set(f, s.depth == 0)
}
