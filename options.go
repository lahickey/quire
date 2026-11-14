package quire

import (
	"io/fs"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/model"
)

type Option func(*Engine)

func WithDev(on bool) Option {
	return func(e *Engine) { e.dev = on }
}

func WithDrafts(on bool) Option {
	return func(e *Engine) { e.drafts = on }
}

func WithPalette(p model.Palette) Option {
	return func(e *Engine) { e.palette = p }
}

func WithPalettePicker(ps ...model.Palette) Option {
	return func(e *Engine) {
		if len(ps) == 0 {
			ps = model.Palettes
		}
		e.chrome.Palettes = ps
	}
}

func WithScale(s model.Scale) Option {
	return func(e *Engine) { e.scale = s }
}

func WithIcons(fsys fs.FS, names ...string) Option {
	return func(e *Engine) {
		for _, n := range names {
			e.icons = append(e.icons, assets.File{FS: fsys, Name: n})
		}
	}
}

func WithStylesheets(fsys fs.FS, names ...string) Option {
	return func(e *Engine) { e.addSheets("site", 0, fsys, names) }
}

func WithStylesheetsAt(w model.Width, fsys fs.FS, names ...string) Option {
	return func(e *Engine) { e.addSheets("site", w, fsys, names) }
}

func WithBake(s model.BakeSpec) Option {
	return func(e *Engine) { e.bakeSpec = s }
}

func WithTheme(t model.Theme) Option {
	return func(e *Engine) { e.chrome.Theme = t }
}

func WithHeader(h func(model.Chrome, model.PageSpec) templ.Component) Option {
	return func(e *Engine) { e.chrome.Header = h }
}

func WithBrand(name string) Option {
	return func(e *Engine) { e.chrome.Brand = name }
}

func WithFooter(f func(model.Chrome, model.PageSpec) templ.Component) Option {
	return func(e *Engine) { e.chrome.Footer = f }
}

func WithFooterText(text string) Option {
	return func(e *Engine) { e.chrome.FooterText = text }
}

func WithFooterLinks(links ...model.Link) Option {
	return func(e *Engine) { e.chrome.FooterLinks = append(e.chrome.FooterLinks, links...) }
}

func WithNavEnd(cs ...templ.Component) Option {
	return func(e *Engine) { e.chrome.NavEnd = append(e.chrome.NavEnd, cs...) }
}

func WithSiteCard(c model.ShareCard) Option {
	return func(e *Engine) { e.siteCard = &c }
}
