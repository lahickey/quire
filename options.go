package quire

import (
	"io/fs"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/model"
)

type Option func(*settings)

func WithDev(on bool) Option {
	return func(s *settings) { s.dev = on }
}

func WithDrafts(on bool) Option {
	return func(s *settings) { s.drafts = on }
}

func WithPalette(p model.Palette) Option {
	return func(s *settings) { s.palette = p }
}

func WithPalettePicker(ps ...model.Palette) Option {
	return func(s *settings) {
		if len(ps) == 0 {
			ps = model.Palettes
		}
		s.chrome.Palettes = ps
	}
}

func WithScale(scale model.Scale) Option {
	return func(s *settings) { s.scale = scale }
}

func WithIcons(fsys fs.FS, names ...string) Option {
	return func(s *settings) {
		for _, n := range names {
			s.icons = append(s.icons, assets.File{FS: fsys, Name: n})
		}
	}
}

func WithStylesheets(fsys fs.FS, names ...string) Option {
	return func(s *settings) { s.addSheets("site", 0, fsys, names) }
}

func WithStylesheetsAt(w model.Width, fsys fs.FS, names ...string) Option {
	return func(s *settings) { s.addSheets("site", w, fsys, names) }
}

func WithBake(spec model.BakeSpec) Option {
	return func(s *settings) { s.bakeSpec = spec }
}

func WithTheme(t model.Theme) Option {
	return func(s *settings) { s.chrome.Theme = t }
}

func WithHeader(h func(model.Chrome, model.PageSpec) templ.Component) Option {
	return func(s *settings) { s.chrome.Header = h }
}

func WithBrand(name string) Option {
	return func(s *settings) { s.chrome.Brand = name }
}

func WithFooter(f func(model.Chrome, model.PageSpec) templ.Component) Option {
	return func(s *settings) { s.chrome.Footer = f }
}

func WithFooterText(text string) Option {
	return func(s *settings) { s.chrome.FooterText = text }
}

func WithFooterLinks(links ...model.Link) Option {
	return func(s *settings) { s.chrome.FooterLinks = append(s.chrome.FooterLinks, links...) }
}

func WithNavEnd(cs ...templ.Component) Option {
	return func(s *settings) { s.chrome.NavEnd = append(s.chrome.NavEnd, cs...) }
}

func WithSiteCard(c model.ShareCard) Option {
	return func(s *settings) { s.siteCard = &c }
}
