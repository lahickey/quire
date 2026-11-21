// Package quire builds a site from a blueprint and a source, and serves it. Docs: https://quire.hiddenbridge.io
package quire

import (
	"crypto/rand"
	"io/fs"
	"net/http"
	"slices"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/bundle"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
)

const (
	assetPrefix      = "/_quire/"
	imageMemory      = 12 << 20
	imageMemoryLimit = 64 << 10
)

// Source is what a site asks for content: a baker answers by converting, and a bundle by reading.
// Names lists the content files matching a pattern. File is a content file kept as it is, and Converted the output
// of the named converter for a body and its arguments; each answers with the bundle file to link to, which Body reads.
// A baker encodes pictures when it finishes, so its Picture answers with nothing but its agreement.
type Source interface {
	Names(pattern string) ([]string, error)
	Document(name string) (model.BakedDocument, error)
	Picture(name string) (model.Picture, error)
	File(name string) (string, error)
	Converted(converter string, body []byte, args ...string) (file string, err error)
	Body(file string) ([]byte, error)
}

// Blueprint says what a site is made of. It is run in a bake as well as when serving, so it only declares.
type Blueprint interface {
	Declare(*Site) error
}

type BlueprintFunc func(*Site) error

// Frame draws what goes round a page's content inside the body: a header, a footer. The content is its children.
type Frame = func(model.Chrome, model.PageSpec) templ.Component

type settings struct {
	chrome   model.Chrome
	dev      bool
	drafts   bool
	palette  model.Palette
	siteCard *model.ShareCard
	icons    []assets.File
	packs    []Blueprint
}

type route struct {
	pattern string
	handler http.HandlerFunc
	cached  bool
}

// said is something a site or a pack may set. What the site itself says is never replaced by a pack, whichever spoke first.
type said[T any] struct {
	v   T
	own bool
}

// Description is everything that makes a site but its source: who it is, its blueprint and its options.
// The same one is collected over a baker and built over the bundle that bake leaves.
type Description struct {
	settings
	blueprint Blueprint
}

// Site is a blueprint run over one source. It has no server: its routes are a table for one to read.
type Site struct {
	settings
	from     Source
	depth    int
	baked    bool
	read     bool
	pictures map[string]model.Picture
	plugins  []plugin.Plugin
	sheets   []assets.File
	scripts  []assets.File

	fonts     model.FontSet
	fontFiles fs.FS
	datastar  bool

	widgets      [2]map[string]model.Widget
	frame        said[Frame]
	home         said[templ.Component]
	notFound     said[templ.Component]
	footerText   said[string]
	placePicture said[func(model.Picture, model.PictureSlot) templ.Component]
	placeVideo   said[func(src string, poster model.Picture, sl model.PictureSlot) templ.Component]

	routes      map[string]string
	handlers    []route
	versions    map[string]string
	sessions    bool
	sitemap     []serve.Loc
	fixed       []*serve.Prepared
	notFound404 *serve.Prepared
	built       bool
	err         error
}

func (x *said[T]) set(v T, own bool) {
	if own || !x.own {
		x.v, x.own = v, own
	}
}

func (d *Description) site(from Source) *Site {
	s := &Site{
		settings: d.settings, from: from, baked: from != nil, routes: map[string]string{}, pictures: map[string]model.Picture{}, versions: map[string]string{},
		widgets: [2]map[string]model.Widget{{}, {}},
	}
	s.chrome.NavEnd, s.chrome.FooterLinks = slices.Clip(s.chrome.NavEnd), slices.Clip(s.chrome.FooterLinks)
	s.frame.v, s.notFound.v, s.placePicture.v, s.placeVideo.v = bareFrame, notFoundBody(), s.barePicture, s.bareVideo
	if from == nil {
		s.from = &bundle.Bundle{}
	}
	return s
}

func (s *Site) declare(b Blueprint, packs []Blueprint) error {
	for _, p := range packs {
		if err := s.Include(p); err != nil {
			return err
		}
	}
	if err := b.Declare(s); err != nil {
		return err
	}
	return s.err
}

func (s *Site) close() error {
	s.chrome.FooterText = s.footerText.v
	s.addAssets()
	s.addIcons()
	if f, ok := s.from.(serve.Filer); ok {
		s.route("GET "+model.BundlePath+"{file}", ownerQuire, serve.Files(f, imageMemory, imageMemoryLimit, s.dev).ServeHTTP)
	}
	if _, taken := s.routes["GET "+pagePattern("/")]; !taken && s.home.v != nil {
		s.AddPage(model.PageSpec{Title: s.chrome.Name, Path: "/"}, s.home.v)
	}
	s.notFound404 = s.prepare("404", s.Page(model.PageSpec{Title: "Not found"}, s.notFound.v), http.StatusNotFound)
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

// Describe gathers what makes a site. Packs given with WithPacks are declared beneath the blueprint, which is the site's own word.
func Describe(id model.Identity, b Blueprint, opts ...Option) *Description {
	d := &Description{blueprint: b, settings: settings{
		chrome:  model.Chrome{Identity: id.WithDefaults(), Brand: id.Name, Scale: model.DefaultScale()},
		palette: model.PaletteEmber,
	}}
	for _, o := range opts {
		o(&d.settings)
	}
	if d.chrome.Palettes == nil {
		d.palette.Check()
	} else {
		d.chrome.Palettes = model.PickerPalettes(d.palette, d.chrome.Palettes)
	}
	return d
}

// Collect runs the blueprint over a baker, so the site asks for all it needs. It attaches no plugin and serves nothing.
// The baker's Finish then writes the bundle.
func (d *Description) Collect(baker Source) error {
	return d.site(baker).declare(d.blueprint, d.packs)
}

// Build runs the blueprint over a source, attaches the plugins it uses, and returns the site.
// A nil source is an empty one, for a site with nothing baked.
func (d *Description) Build(from Source) (*Site, error) {
	s := d.site(from)
	if s.dev {
		s.plugins = append(s.plugins, liveReload{s, rand.Text()})
	}
	if err := s.declare(d.blueprint, d.packs); err != nil {
		return nil, err
	}
	for _, p := range s.plugins {
		if err := s.attach(p); err != nil {
			return nil, err
		}
	}
	return s, s.close()
}

func (f BlueprintFunc) Declare(s *Site) error { return f(s) }

// Include runs a pack's blueprint on the site. What it sets gives way to what the site's own blueprint sets.
func (s *Site) Include(b Blueprint) error {
	s.depth++
	defer func() { s.depth-- }()
	if err := b.Declare(s); err != nil {
		return err
	}
	return s.err
}
