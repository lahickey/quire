// Package site is the table a site is declared on: its features run over a source, and the routes that leaves. Docs: https://quire.hiddenbridge.io
package site

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/bundle"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
)

const (
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

// Feature says what a site is made of. It is run in a bake as well as when serving, so it only declares.
// One with a name has a namespace the site holds it to: its routes under /_name/, its classes under name-.
// One without works in the site's own, as the site's own feature and a pack's do.
type Feature interface {
	Name() string
	Declare(*Site) error
}

// FeatureFunc is a feature with no name.
type FeatureFunc func(*Site) error

// Service is something that runs while a site is served and that others call: a store, sessions, a mailer.
// It never sees the site. Its constructor only keeps what it is given, and Start is told the site's environment and opens what it needs.
type Service interface {
	Start(context.Context, model.Env) error
	Stop(context.Context) error
}

// Frame draws what goes round a page's content inside the body: a header, a footer. The content is its children.
type Frame = func(model.Chrome, model.PageSpec) templ.Component

type settings struct {
	chrome   model.Chrome
	dev      bool
	drafts   bool
	palette  model.Palette
	log      *slog.Logger
	siteCard *model.ShareCard
	icons    []assets.File
	features []Feature
}

// Route is one line of a site's table. A shared route answers every visitor alike, so its response may be kept.
// One in a group has that group's gate round its handler already.
type Route struct {
	Pattern string
	Handler http.HandlerFunc
	Shared  bool
	Group   Group
	asIs    bool
}

// Loc is an address a sitemap lists.
type Loc struct{ Path, LastMod string }

// said is something a site or a pack may set. What the site itself says is never replaced by a pack, whichever spoke first.
type said[T any] struct {
	v   T
	own bool
}

// Description is everything that makes a site but its source: who it is, its own feature and its options.
// The same one is collected over a baker and built over the bundle that bake leaves.
type Description struct {
	settings
	feature Feature
}

// Site is features run over one source. It has no server: its routes are a table for one to read.
type Site struct {
	settings
	from     Source
	owner    string
	named    []string
	depth    int
	baked    bool
	read     bool
	pictures map[string]model.Picture
	sheets   []assets.File
	scripts  []assets.File
	bodyEnd  []templ.Component
	gates    map[Group]said[Gate]
	gated    []gated
	services []Service

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
	handlers    []Route
	table       []Route
	versions    map[string]string
	sitemap     []Loc
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
		widgets: [2]map[string]model.Widget{{}, {}}, gates: map[Group]said[Gate]{},
	}
	s.chrome.NavEnd, s.chrome.FooterLinks = slices.Clip(s.chrome.NavEnd), slices.Clip(s.chrome.FooterLinks)
	for _, at := range []string{s.chrome.Root, s.chrome.Engine} {
		if at != "" && (!strings.HasPrefix(at, "/") || strings.HasSuffix(at, "/")) {
			s.fail("a site is placed at %q: a path with a leading slash and no trailing one", at)
		}
	}
	s.frame.v, s.notFound.v, s.placePicture.v, s.placeVideo.v = bareFrame, notFoundBody(), s.barePicture, s.bareVideo
	if from == nil {
		s.from = &bundle.Bundle{}
	}
	return s
}

func (s *Site) fail(format string, a ...any) {
	err := fmt.Errorf(format, a...)
	if s.owner != "" {
		err = fmt.Errorf("feature %s: %w", s.owner, err)
	}
	s.err = cmp.Or(s.err, err)
}

// own says the site itself is speaking, and refuses a named feature what belongs to the look.
func (s *Site) own(what string) bool {
	mustBeOpen(s.built, what)
	if s.owner != "" {
		s.fail("%s is the site's look, which a named feature leaves alone", what)
	}
	return s.depth == 0
}

func (s *Site) run(f Feature, depth int) error {
	name := f.Name()
	switch {
	case name == "":
	case name == ownerQuire || strings.ContainsAny(name, "/_"):
		return fmt.Errorf("feature name %q: one word, no slash or underscore, and not %s", name, ownerQuire)
	case slices.Contains(s.named, name):
		return fmt.Errorf("feature %s is included twice", name)
	default:
		s.named = append(s.named, name)
	}
	defer func(owner string, depth int) { s.owner, s.depth = owner, depth }(s.owner, s.depth)
	s.owner, s.depth = cmp.Or(name, s.owner), depth
	return cmp.Or(f.Declare(s), s.err)
}

func (d *Description) declare(s *Site) error {
	if d.feature.Name() != "" {
		return errors.New("the site's own feature has no name: include a named one from it")
	}
	for _, f := range d.features {
		if err := s.Include(f); err != nil {
			return err
		}
	}
	return s.run(d.feature, 0)
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
	s.gate()
	s.notFound404 = s.prepare("404", s.Page(model.PageSpec{Title: "Not found"}, s.notFound.v), http.StatusNotFound)
	s.built = true
	s.table = s.handlers
	if p := s.chrome.Place; p != (model.Place{}) {
		s.table = make([]Route, len(s.handlers))
		for i, r := range s.handlers {
			r.Pattern, r.Handler = placedPattern(p, r.Pattern), placedHandler(p, r.Handler, r.asIs)
			s.table[i] = r
		}
	}
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

// Describe gathers what makes a site. Features given with WithFeatures are declared beneath f, which is the site's own word.
func Describe(id model.Identity, f Feature, opts ...Option) *Description {
	d := &Description{feature: f, settings: settings{
		chrome:  model.Chrome{Identity: id.WithDefaults(), Brand: id.Name, Scale: model.DefaultScale()},
		palette: model.PaletteEmber,
		log:     slog.Default(),
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

// Collect runs the features over a baker, so the site asks for all it needs. It serves nothing.
// The baker's Finish then writes the bundle.
func (d *Description) Collect(baker Source) error {
	return d.declare(d.site(baker))
}

// Build runs the features over a source and returns the site.
// A nil source is an empty one, for a site with nothing baked.
func (d *Description) Build(from Source) (*Site, error) {
	s := d.site(from)
	if err := d.declare(s); err != nil {
		return nil, err
	}
	if err := s.close(); err != nil {
		return nil, err
	}
	return s, nil
}

func (FeatureFunc) Name() string { return "" }

func (f FeatureFunc) Declare(s *Site) error { return f(s) }

// AddServices says the site has these services. Whatever serves the site starts them in this order and stops them in reverse.
// A bake starts nothing. Every service is added, whether or not it has anything to start yet.
func (s *Site) AddServices(services ...Service) {
	mustBeOpen(s.built, "a service")
	s.services = append(s.services, services...)
}

func (s *Site) Services() []Service { return s.services }

// Env is the site's environment, as its services are told it.
func (s *Site) Env() model.Env {
	return model.Env{Identity: s.chrome.Identity, Place: s.chrome.Place, Dev: s.dev, Log: s.log}
}

// Log is where the feature now declaring writes, under its name.
func (s *Site) Log() *slog.Logger {
	if s.owner == "" {
		return s.log
	}
	return s.log.With("feature", s.owner)
}

// Include runs a feature on the site. What it sets gives way to what the site's own feature sets.
func (s *Site) Include(f Feature) error { return s.run(f, s.depth+1) }
