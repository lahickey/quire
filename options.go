package quire

import (
	"io/fs"

	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/model"
)

// Option is what has to be known before a blueprint runs. Everything else a site says, it says in its blueprint.
type Option func(*settings)

func WithDev(on bool) Option {
	return func(s *settings) { s.dev = on }
}

func WithDrafts(on bool) Option {
	return func(s *settings) { s.drafts = on }
}

// WithPacks declares packs beneath the site's blueprint, in order, so what they set is in place before it runs.
func WithPacks(packs ...Blueprint) Option {
	return func(s *settings) { s.packs = append(s.packs, packs...) }
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
	return func(s *settings) { s.chrome.Scale = scale }
}

func WithIcons(fsys fs.FS, names ...string) Option {
	return func(s *settings) {
		for _, n := range names {
			s.icons = append(s.icons, assets.File{FS: fsys, Name: n})
		}
	}
}

func WithTheme(t model.Theme) Option {
	return func(s *settings) { s.chrome.Theme = t }
}

func WithBrand(name string) Option {
	return func(s *settings) { s.chrome.Brand = name }
}

func WithSiteCard(c model.ShareCard) Option {
	return func(s *settings) { s.siteCard = &c }
}
