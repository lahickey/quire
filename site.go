// Package quire builds a site from a blueprint and a source, and serves it. Docs: https://quire.hiddenbridge.io
package quire

import (
	"cmp"
	"crypto/rand"
	"encoding/json/v2"
	"net/http"
	"slices"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/bundle"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/internal/fonts"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/ui"
)

const (
	assetPrefix      = "/_quire/"
	imageMemory      = 12 << 20
	imageMemoryLimit = 64 << 10
)

// Source is what a site asks for content: a baker answers by converting, and a bundle by reading.
// Names lists the content files matching a pattern, File is a content file kept as it is,
// Converted is the output of the named converter for a body and its arguments, and Body is the one file a record names.
// A baker makes pictures when it finishes, so a record from its Picture names the master and no more.
type Source interface {
	Names(pattern string) ([]string, error)
	Document(name string, fences []string) (bundle.Document, error)
	Picture(name string) (bundle.Record, error)
	File(name string) (bundle.Record, error)
	Converted(converter string, body []byte, args ...string) (bundle.Record, error)
	Body(bundle.Record) ([]byte, error)
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
}

type route struct {
	pattern string
	handler http.HandlerFunc
	cached  bool
}

// Site is blueprints run over one source. It has no server: its routes are a table for one to read.
type Site struct {
	settings
	from       Source
	blueprints []Blueprint
	scripts    []assets.File
	pictures   map[string]model.Picture

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

func (s *settings) site(from Source) *Site {
	site := &Site{settings: *s, from: from, routes: map[string]string{}, pictures: map[string]model.Picture{}, versions: map[string]string{}}
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
	body, err := json.Marshal(c)
	if err != nil {
		return err
	}
	rec, err := s.from.Converted("card", body, s.fonts.ID)
	if file, ok := rec.File(); ok {
		s.chrome.OGImageURL = s.chrome.AbsURL(model.BundleURL(file))
	}
	return err
}

func (s *Site) close() error {
	s.addAssets()
	s.addIcons()
	if f, ok := s.from.(serve.Filer); ok {
		s.route("GET "+model.BundlePath+"{file}", ownerQuire, serve.Files(f, imageMemory, imageMemoryLimit, s.dev).ServeHTTP)
	}
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

func (s *settings) build(from Source, b Blueprint) (*Site, error) {
	site := s.site(from)
	if from == nil {
		site.from = &bundle.Bundle{}
	} else if err := site.addSiteCard(); err != nil {
		return nil, err
	}
	if s.dev {
		if err := site.attach(liveReload{site, rand.Text()}); err != nil {
			return nil, err
		}
	}
	if err := site.Include(b); err != nil {
		return nil, err
	}
	for _, b := range site.blueprints {
		if l, ok := b.(interface{ Live(*Live) error }); ok {
			if err := l.Live(&Live{site}); err != nil {
				return nil, err
			}
		}
	}
	return site, site.close()
}

// Collect runs a blueprint over a baker, so the site asks for all it needs: what Declare reads, and the share card.
// It calls no Live and serves nothing. The baker's Finish then writes the bundle.
func Collect(id model.Identity, baker Source, b Blueprint, opts ...Option) error {
	site := configure(id, opts...).site(baker)
	if err := site.Include(b); err != nil {
		return err
	}
	return site.addSiteCard()
}

// Build runs a blueprint over a source and returns the site: what it declares, then what its Live adds.
// A nil source is an empty one, for a site with nothing baked.
func Build(id model.Identity, from Source, b Blueprint, opts ...Option) (*Site, error) {
	return configure(id, opts...).build(from, b)
}

func (f BlueprintFunc) Declare(s *Site) error { return f(s) }

func (s *Site) Include(b Blueprint) error {
	s.blueprints = append(s.blueprints, b)
	return cmp.Or(b.Declare(s), s.err)
}
