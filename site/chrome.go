package site

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/model"
)

const (
	stylePath  = model.EnginePath + "style.css"
	scriptPath = model.EnginePath + "quire.js"
	javascript = "application/javascript; charset=utf-8"
	css        = "text/css; charset=utf-8"
)

var (
	preludes = regexp.MustCompile(`[^{};]*\{`)
	classes  = regexp.MustCompile(`\.(-?[A-Za-z_][\w-]*)`)
	comments = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

// namespaced fails a sheet with a class that does not begin with the feature's name.
func namespaced(name string, fsys fs.FS, sheet string) error {
	css, err := fs.ReadFile(fsys, sheet)
	if err != nil {
		return err
	}
	for _, pre := range preludes.FindAllString(comments.ReplaceAllString(string(css), ""), -1) {
		for _, c := range classes.FindAllStringSubmatch(pre, -1) {
			if !strings.HasPrefix(c[1], name+"-") {
				return fmt.Errorf("%s: class .%s is outside the %s- namespace", sheet, c[1], name)
			}
		}
	}
	return nil
}

func (s *Site) addSheets(layer string, at model.Width, fsys fs.FS, names []string) {
	for _, n := range names {
		s.sheets = append(s.sheets, assets.File{FS: fsys, Name: n, Layer: layer, At: at})
	}
}

func (s *Site) addAssets() {
	p := assets.Parts{Scale: s.chrome.Scale, Sheets: s.sheets, Scripts: s.scripts}
	if s.fontFiles != nil {
		p.Fonts, s.chrome.FontPreload = fontCSS(s.fonts, s.fontFiles, func(file string, body []byte) string {
			return s.addBytes(model.EnginePath+"fonts/"+file, "font/woff2", body)
		})
	}
	if sheet := assets.Stylesheet(p); len(sheet) > 0 {
		s.chrome.StyleURLs = []string{s.addBytes(stylePath, css, placedCSS(s.chrome.Place, sheet))}
	}
	if script := assets.Script(p); len(script) > 0 {
		s.chrome.ScriptURL = s.addBytes(scriptPath, javascript, script)
	}
	if s.datastar {
		s.chrome.ModuleURLs = []string{s.addBytes(model.EnginePath+"datastar.js", javascript, assets.Datastar())}
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

func (s *Site) page(p model.PageSpec, body templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return document(s.chrome, p, s.frame.v, s.bodyEnd).Render(templ.WithChildren(model.WithChrome(ctx, &s.chrome), body), w)
	})
}

// Page is a body as a whole page: the site's document round the frame round the body, with the chrome in reach of everything inside.
// Its addresses are placed, so it can be drawn anywhere.
func (s *Site) Page(p model.PageSpec, body templ.Component) templ.Component {
	page := s.page(p, body)
	if s.chrome.Place == (model.Place{}) {
		return page
	}
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var markup bytes.Buffer
		if err := page.Render(ctx, &markup); err != nil {
			return err
		}
		_, err := w.Write(placedHTML(s.chrome.Place, markup.Bytes()))
		return err
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

// AddStylesheets adds CSS. The site's own is laid over a pack's, and a pack's over a named feature's, whichever came first.
func (s *Site) AddStylesheets(fsys fs.FS, names ...string) { s.AddStylesheetsAt(0, fsys, names...) }

func (s *Site) AddStylesheetsAt(w model.Width, fsys fs.FS, names ...string) {
	mustBeOpen(s.built, "a stylesheet")
	layer := "site"
	switch {
	case s.owner != "":
		layer = "feature"
		for _, n := range names {
			if err := namespaced(s.owner, fsys, n); err != nil {
				s.fail("%w", err)
			}
		}
	case s.depth > 0:
		layer = "pack"
	}
	s.addSheets(layer, w, fsys, names)
}

func (s *Site) AddNavEnd(cs ...templ.Component) {
	mustBeOpen(s.built, "a nav control")
	if slices.Contains(cs, nil) {
		s.fail("nil nav control")
	}
	s.chrome.NavEnd = append(s.chrome.NavEnd, cs...)
}

// AddBodyEnd adds what every page's body ends with, after the frame: something a feature needs on the page and nobody sees.
func (s *Site) AddBodyEnd(cs ...templ.Component) {
	mustBeOpen(s.built, "the end of the body")
	s.bodyEnd = append(s.bodyEnd, cs...)
}

func (s *Site) SetFooterText(text string) { s.footerText.set(text, s.own("the footer")) }

func (s *Site) AddFooterLinks(links ...model.Link) {
	mustBeOpen(s.built, "a footer link")
	s.chrome.FooterLinks = append(s.chrome.FooterLinks, links...)
}

// AddBaseCSS adds CSS to the bottom layer, beneath every feature, pack and the site: the look everything else is laid over.
// It takes the CSS itself, since some of it is made and not read, such as what a palette becomes.
func (s *Site) AddBaseCSS(name string, body []byte) {
	s.own("the base stylesheet")
	s.sheets = append(s.sheets, assets.File{Name: name, Body: body, Layer: "base"})
}

// AddScripts adds JavaScript to the site's one script, each file in a scope of its own.
func (s *Site) AddScripts(fsys fs.FS, names ...string) {
	mustBeOpen(s.built, "a script")
	for _, n := range names {
		if _, err := fs.Stat(fsys, n); err != nil {
			s.fail("%w", err)
		}
		s.scripts = append(s.scripts, assets.File{FS: fsys, Name: n})
	}
}

// SetFonts is the font set the site draws in: a description, fonts.json, and the faces it names.
func (s *Site) SetFonts(fsys fs.FS) {
	s.own("the fonts")
	set, err := model.ReadFontSet(fsys)
	if err != nil {
		s.fail("fonts: %w", err)
		return
	}
	s.fonts, s.fontFiles = set, fsys
}

// UseDatastar links Datastar on every page, for markup that needs it. A site with streams has it anyway.
func (s *Site) UseDatastar() { s.datastar = true }

// SetFrame draws what goes round every page's content. Without one a page is its content and nothing else.
func (s *Site) SetFrame(f Frame) { s.frame.set(f, s.own("the frame")) }
