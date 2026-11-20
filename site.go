package quire

import (
	"cmp"
	"crypto/rand"
	"net/http"
	"slices"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/internal/fonts"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/ui"
)

const assetPrefix = "/_quire/"

type source interface {
	Master(name string) (journal.Record, error)
	Video(name string) (journal.Record, error)
	Diagram(src string) (string, error)
	Documents(pattern string, fences []string) ([]journal.Document, error)
	Card(name string, c model.ShareCard) (journal.Record, error)
}

// Blueprint says what a site is made of, and is run in a bake as well as when serving.
// One that also has Live(*Live) error adds what only a served site has; a bake never calls it.
type Blueprint interface {
	Declare(*Site) error
}

type BlueprintFunc func(*Site) error

type settings struct {
	chrome   model.Chrome
	dev      bool
	drafts   bool
	palette  model.Palette
	scale    model.Scale
	siteCard *model.ShareCard
	sheets   []assets.File
	icons    []assets.File
	fonts    fonts.FontSet
	bakeSpec model.BakeSpec
}

type route struct {
	pattern string
	handler http.HandlerFunc
	cached  bool
}

// Site is blueprints run over one source. It has no server: its routes are a table for one to read.
type Site struct {
	settings
	from       source
	blueprints []Blueprint
	scripts    []assets.File
	images     map[string]model.Image

	routes      map[string]string
	handlers    []route
	versions    map[string]string
	sessions    bool
	sitemap     []serve.Loc
	fixed       []*serve.Prepared
	notFound    templ.Component
	notFound404 *serve.Prepared
	built       bool
	err         error
}

func configure(id model.Identity, opts ...Option) *settings {
	s := &settings{
		chrome:  model.Chrome{Identity: id.WithDefaults(), Brand: id.Name, Header: ui.Header, Footer: ui.Footer},
		palette: model.PaletteEmber,
		scale:   model.DefaultScale(),
	}
	for _, o := range opts {
		o(s)
	}
	if s.chrome.Palettes == nil {
		s.palette.Check()
	} else {
		s.chrome.Palettes = model.PickerPalettes(s.palette, s.chrome.Palettes)
	}
	set, err := fonts.ReadFontSet(fonts.Default())
	if err != nil {
		panic("quire: fonts: " + err.Error())
	}
	s.fonts = set
	return s
}

func (s *settings) site(from source) *Site {
	site := &Site{settings: *s, from: from, routes: map[string]string{}, images: map[string]model.Image{}, versions: map[string]string{}}
	site.chrome.NavEnd, site.chrome.FooterLinks, site.sheets = slices.Clip(site.chrome.NavEnd), slices.Clip(site.chrome.FooterLinks), slices.Clip(site.sheets)
	return site
}

func (s *Site) addSiteCard() error {
	c := model.ShareCard{Title: s.chrome.Name, Lines: []string{s.chrome.Description}}
	if s.siteCard != nil {
		c = *s.siteCard
	}
	c.Domain = cmp.Or(c.Domain, s.chrome.Host())
	if c.Palette == (model.Palette{}) {
		c.Palette = s.palette
	}
	c.Palette.Check()
	rec, err := s.from.Card(journal.SiteCard, c)
	if file, ok := rec.File(); ok {
		s.chrome.OGImageURL = s.chrome.AbsURL(model.ImageURL(file))
	}
	return err
}

func (s *Site) close() error {
	s.addAssets()
	s.addIcons()
	if _, taken := s.routes["GET "+pagePattern("/")]; !taken {
		s.AddPage("/", s.late(ui.Home))
	}
	page := s.notFound
	if page == nil {
		page = s.late(ui.NotFound)
	}
	s.notFound404 = s.prepare("404", page, http.StatusNotFound)
	s.built = true
	if s.err != nil {
		return s.err
	}
	for _, p := range s.fixed {
		if err := p.Build(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Site) build(b Blueprint) error {
	if s.dev {
		if err := s.attach(liveReload{s, rand.Text()}); err != nil {
			return err
		}
	}
	if err := s.Include(b); err != nil {
		return err
	}
	for _, b := range s.blueprints {
		if l, ok := b.(interface{ Live(*Live) error }); ok {
			if err := l.Live(&Live{s}); err != nil {
				return err
			}
		}
	}
	return s.close()
}

func (f BlueprintFunc) Declare(s *Site) error { return f(s) }

func (s *Site) Include(b Blueprint) error {
	s.blueprints = append(s.blueprints, b)
	return cmp.Or(b.Declare(s), s.err)
}
