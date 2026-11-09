package quire

import (
	"io/fs"
	"mime"
	"path"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
)

type Mode int

const (
	Prod Mode = iota
	Dev
)

type Option func(*Engine)

type icon struct {
	fsys fs.FS
	model.Icon
}

func WithMode(m Mode) Option {
	return func(e *Engine) { e.mode = m }
}

func WithDrafts(on bool) Option {
	return func(e *Engine) { e.drafts = on }
}

func WithPalette(p model.Palette) Option {
	return func(e *Engine) { e.palette = p }
}

func WithScale(s model.Scale) Option {
	return func(e *Engine) { e.scale = s }
}

func WithWidest(w model.Width) Option {
	return func(e *Engine) { e.scale.Widest = w }
}

func WithIcons(fsys fs.FS, names ...string) Option {
	return func(e *Engine) {
		for _, n := range names {
			rel := "icon"
			if strings.HasPrefix(path.Base(n), "apple-touch-icon") {
				rel = "apple-touch-icon"
			}
			e.icons = append(e.icons, icon{fsys, model.Icon{Rel: rel, Type: mime.TypeByExtension(path.Ext(n)), Href: "/" + n}})
		}
	}
}

func WithStylesheets(fsys fs.FS, names ...string) Option {
	return func(e *Engine) { e.addSheets("site", 0, fsys, names) }
}

func WithStylesheetsAt(w model.Width, fsys fs.FS, names ...string) Option {
	return func(e *Engine) { e.addSheets("site", w, fsys, names) }
}

func WithBundle(fsys fs.FS) Option {
	return func(e *Engine) { e.bundles = append(e.bundles, fsys) }
}

func WithBake(s model.BakeSpec) Option {
	return func(e *Engine) { e.bakeSpec = s }
}

func WithTheme(t model.Theme) Option {
	return func(e *Engine) { e.theme = t }
}

func WithHeader(h func(model.Chrome, model.PageSpec) templ.Component) Option {
	return func(e *Engine) { e.header = h }
}

func WithFooter(f func(model.Chrome, model.PageSpec) templ.Component) Option {
	return func(e *Engine) { e.footer = f }
}

func WithNavEnd(cs ...templ.Component) Option {
	return func(e *Engine) { e.navEnd = append(e.navEnd, cs...) }
}

func WithSiteCard(c model.ShareCard) Option {
	return func(e *Engine) { e.siteCard = &c }
}
