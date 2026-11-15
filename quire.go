// Package quire turns a folder of markdown into a website served from one binary. Docs: https://quire.hiddenbridge.io
package quire

import (
	"cmp"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/internal/fonts"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/internal/session"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/ui"
)

const (
	assetPrefix      = "/_quire/"
	healthPath       = assetPrefix + "health"
	pageCache        = 4 << 20
	imageMemory      = 12 << 20
	imageMemoryLimit = 64 << 10
)

var shipped fs.FS

type source interface {
	Master(name string) (journal.Record, error)
	Diagram(src string) (string, error)
	Documents(pattern string, fences []string) ([]journal.Document, error)
	Card(name string, c model.ShareCard) (journal.Record, error)
}

// Part is a site or a pack. Mount says what content and pages there are, and runs in a bake as well as when serving.
// A Part that also has Live(*Live) error adds what only a served site has; a bake never calls it.
type Part interface {
	Mount(*Engine) error
}

type MountFunc func(*Engine) error

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

// Engine is one mount of the site over one source. bundle, mux and pages are the server's, and a bake has none of them.
type Engine struct {
	settings
	from    source
	scripts []assets.File
	images  map[string]model.Image

	routes      map[string]string
	handlers    []route
	versions    map[string]string
	sessions    bool
	sitemap     []serve.Loc
	fixed       []*serve.Prepared
	notFound    templ.Component
	notFound404 *serve.Prepared
	mounted     bool

	bundle *serve.Store
	mux    *http.ServeMux
	pages  *serve.Pages
}

func (f MountFunc) Mount(e *Engine) error { return f(e) }

func (e *Engine) Mount(p Part) error {
	if err := p.Mount(e); err != nil {
		return err
	}
	if l, ok := p.(interface{ Live(*Live) error }); ok && e.bundle != nil {
		return l.Live(&Live{e})
	}
	return nil
}

func (e *Engine) cacheControl(r *http.Request) string {
	if e.dev || r.URL.Path == healthPath {
		return "no-store"
	}
	return serve.CacheControl(r, e.versions)
}

func (e *Engine) mountDefaults() {
	if e.mounted {
		return
	}
	e.route("GET "+healthPath, ownerQuire, serve.Health)
	e.addAssets()
	e.addIcons()
	e.addSiteFiles()
	e.route("GET "+model.ImagePath+"{file}", ownerQuire, e.bundle.ServeHTTP)
	if _, taken := e.routes["GET "+pagePattern("/")]; !taken {
		e.AddPage("/", e.late(ui.Home))
	}
	e.addNotFound()
	e.mounted = true
	for _, p := range e.fixed {
		p.Build()
	}
	e.mux, e.pages = http.NewServeMux(), serve.NewPages(pageCache, e.dev)
	for _, r := range e.handlers {
		if r.cached {
			r.handler = e.pages.Cached(r.handler)
		}
		e.mux.HandleFunc(r.pattern, r.handler)
	}
}

func (e *Engine) addSiteCard() error {
	c := model.ShareCard{Title: e.chrome.Name, Lines: []string{e.chrome.Description}}
	if e.siteCard != nil {
		c = *e.siteCard
	}
	c.Domain = cmp.Or(c.Domain, e.chrome.Host())
	if c.Palette == (model.Palette{}) {
		c.Palette = e.palette
	}
	c.Palette.Check()
	rec, err := e.from.Card(journal.SiteCard, c)
	if file, ok := rec.File(); ok {
		e.chrome.OGImageURL = e.chrome.AbsURL(model.ImageURL(file))
	}
	return err
}

func (e *Engine) load(bundle fs.FS) *Engine {
	if bundle == nil {
		bundle = os.DirFS(e.bakeSpec.WithDefaults().Bundle())
	}
	if err := e.bundle.Load(bundle); err != nil {
		panic("quire: bundle: " + err.Error() + "; run the site's bake")
	}
	if err := e.addSiteCard(); err != nil {
		panic(err.Error())
	}
	return e
}

func configure(site model.Site, opts ...Option) *settings {
	s := &settings{
		chrome:  model.Chrome{Site: site.WithDefaults(), Brand: site.Name, Header: ui.Header, Footer: ui.Footer},
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

func (s *settings) pass(from source) *Engine {
	e := &Engine{settings: *s, from: from, routes: map[string]string{}, images: map[string]model.Image{}, versions: map[string]string{}}
	e.chrome.NavEnd, e.chrome.FooterLinks, e.sheets = slices.Clip(e.chrome.NavEnd), slices.Clip(e.chrome.FooterLinks), slices.Clip(e.sheets)
	return e
}

func (s *settings) serving() *Engine {
	store := serve.NewStore(imageMemory, imageMemoryLimit, s.dev, s.fonts.ID)
	e := s.pass(store)
	e.bundle = store
	if e.dev {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			panic("quire: boot id: " + err.Error())
		}
		if err := e.attach(liveReload{e, hex.EncodeToString(b[:])}); err != nil {
			panic("quire: live reload: " + err.Error())
		}
	}
	return e
}

func New(site model.Site, opts ...Option) *Engine { return configure(site, opts...).serving() }

func (e *Engine) Handler() http.Handler {
	e.mountDefaults()
	apex := e.chrome.Host()
	if e.dev {
		apex = ""
	}
	secure := !e.dev && strings.HasPrefix(e.chrome.BaseURL, "https://")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if target, ok := serve.Redirect(r, apex); ok {
			w.Header().Set("Cache-Control", "no-cache")
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		w.Header().Set("Cache-Control", e.cacheControl(r))
		if e.sessions {
			r = session.Mint(w, r, secure)
		}
		e.mux.ServeHTTP(w, r)
	})
}

func Ship(files fs.FS, dir string) {
	b, err := fs.Sub(files, dir)
	if err != nil {
		panic("quire: ship: " + err.Error())
	}
	shipped = b
}

func Run(site model.Site, part Part, extra ...Option) {
	dev := flag.Bool("dev", false, "live reload")
	drafts := flag.Bool("drafts", false, "include draft documents")
	addr := flag.String("addr", ":8090", "listen address")
	flag.Parse()
	baking := flag.Arg(0) == "bake"
	if flag.NArg() > 1 || (flag.NArg() == 1 && !baking) {
		log.Fatalf("usage: %s [flags] [bake]", os.Args[0])
	}
	s := configure(site, append([]Option{WithDev(*dev), WithDrafts(*drafts)}, extra...)...)
	if baking || (shipped == nil && canBake) {
		if err := bakeSite(s, part); err != nil {
			log.Fatal(err)
		}
		if baking {
			return
		}
	}
	e := s.serving().load(shipped)
	if err := e.Mount(part); err != nil {
		log.Fatal(err)
	}
	log.Println("listening on", *addr)
	srv := &http.Server{Addr: *addr, Handler: e.Handler(), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 2 * time.Minute}
	err := srv.ListenAndServe()
	if op, ok := errors.AsType[*net.OpError](err); ok && errors.Is(op.Err, syscall.EADDRINUSE) {
		log.Fatalf("listen %s: already in use", *addr)
	}
	log.Fatal(err)
}
