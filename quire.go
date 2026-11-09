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
	"strings"
	"syscall"
	"time"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/bake"
	"github.com/lahickey/quire/internal/bundle"
	"github.com/lahickey/quire/internal/fonts"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
	"github.com/lahickey/quire/ui"
)

const (
	assetPrefix      = "/_quire/"
	healthPath       = assetPrefix + "health"
	reloadPath       = assetPrefix + "reload"
	reloadInit       = `@get('` + reloadPath + `', {retry: 'always', openWhenHidden: true})`
	pageCache        = 4 << 20
	imageMemory      = 12 << 20
	imageMemoryLimit = 64 << 10
)

var shipped fs.FS

type source interface {
	Master(name string) (journal.Record, error)
	Diagram(src string) (string, error)
	Documents(pattern string, fences []string) ([]journal.Document, error)
	Data(name string) ([]byte, error)
}

type Engine struct {
	site     model.Site
	mode     Mode
	drafts   bool
	palette  model.Palette
	scale    model.Scale
	siteCard *model.ShareCard
	theme    model.Theme
	header   func(model.Chrome, model.PageSpec) templ.Component
	footer   func(model.Chrome, model.PageSpec) templ.Component
	icons    []icon
	bundles  []fs.FS

	slots  map[model.Slot]model.SlotSpec
	fonts  fonts.FontSet
	bundle *bundle.Store
	pages  *serve.Pages
	bootID string

	bakeSpec model.BakeSpec
	from     source

	mux         *http.ServeMux
	routes      map[string]bool
	versions    map[string]string
	claims      map[string]string
	sessions    bool
	sitemap     []serve.Loc
	fixed       []*serve.Prepared
	notFound    templ.Component
	notFound404 *serve.Prepared
	mounted     bool

	linked   linked
	navItems []model.NavItem
	navEnd   []templ.Component
	chrome   model.Chrome
}

func (e *Engine) cacheControl(r *http.Request) string {
	if e.mode == Dev || r.URL.Path == healthPath {
		return "no-store"
	}
	return serve.CacheControl(r, e.versions)
}

func (e *Engine) mountDefaults() {
	if e.mounted {
		return
	}
	e.addAssets()
	e.addSiteFiles()
	e.route("GET "+model.ImagePath+"{file}", false, e.bundle.ServeHTTP)
	if !e.claimed("GET " + pagePattern("/")) {
		e.AddPage("/", e.late(ui.Home))
	}
	e.addNotFound()
	e.chrome = e.buildChrome()
	e.mounted = true
	for _, p := range e.fixed {
		p.Build()
	}
}

func (e *Engine) siteCardText() model.ShareCard {
	c := model.ShareCard{Title: e.site.Name, Lines: []string{e.site.Description}}
	if e.siteCard != nil {
		c = *e.siteCard
	}
	c.Domain = cmp.Or(c.Domain, e.site.Host())
	if c.Ink == (model.Ink{}) {
		c.Ink = e.palette.Ink()
	}
	return c
}

func (e *Engine) bake(mount func(*Engine) error) error {
	s := e.bakeSpec.WithDefaults()
	if fi, err := os.Stat(s.Content); err != nil || !fi.IsDir() {
		s.Content = ""
	}
	b, err := bake.Start(s)
	if err != nil {
		return err
	}
	e.from = b
	if err := mount(e); err != nil {
		return err
	}
	if err := b.Card(journal.SiteCard, e.siteCardText()); err != nil {
		return err
	}
	return b.Finish()
}

func New(site model.Site, opts ...Option) *Engine {
	e := &Engine{
		mux:      http.NewServeMux(),
		routes:   map[string]bool{},
		versions: map[string]string{},
		claims:   map[string]string{},
		site:     site.WithDefaults(),
		palette:  model.PaletteEmber,
		scale:    model.DefaultScale(),
		header:   ui.Header,
		footer:   ui.Footer,
	}
	for _, o := range opts {
		o(e)
	}

	e.slots = e.scale.Slots()
	e.palette.Check()
	set, err := fonts.ReadFontSet(fonts.Default())
	if err != nil {
		panic("quire: fonts: " + err.Error())
	}
	e.fonts = set
	e.bundle = bundle.New(imageMemory, imageMemoryLimit, e.mode == Dev, e.fonts.ID)
	e.from = e.bundle

	for _, fsys := range e.bundles {
		if err := e.bundle.AddBundle(fsys); err != nil {
			panic("quire: bundle: " + err.Error() + "; run the site's bake")
		}
	}
	e.pages = serve.NewPages(pageCache, e.mode == Dev)
	e.AddRoute("GET", healthPath, serve.Health)
	for k, i := range e.icons {
		e.icons[k].Href = e.AddFile(i.Href, i.fsys, strings.TrimPrefix(i.Href, "/"))
	}
	if e.mode == Dev {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			panic("engine: boot id: " + err.Error())
		}
		e.bootID = hex.EncodeToString(b[:])
		e.AddRoute("GET", reloadPath, func(w http.ResponseWriter, r *http.Request) {
			_ = plugin.OpenStream(w).Signals(map[string]any{"quire": map[string]string{"boot": e.bootID}})
			<-r.Context().Done()
		})
	}
	return e
}

func (e *Engine) Handler() http.Handler {
	e.mountDefaults()
	apex := e.site.Host()
	if e.mode == Dev {
		apex = ""
	}
	secure := e.mode != Dev && strings.HasPrefix(e.site.BaseURL, "https://")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if target, ok := serve.Redirect(r, apex); ok {
			w.Header().Set("Cache-Control", "no-cache")
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		w.Header().Set("Cache-Control", e.cacheControl(r))
		if e.sessions {
			r = plugin.Mint(w, r, secure)
		}

		_, pattern := e.mux.Handler(r)
		if !e.routes[pattern] || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
			e.mux.ServeHTTP(w, r)
			return
		}
		e.pages.Serve(w, r, e.mux, e.notFound404)
	})
}

func Ship(files fs.FS) {
	b, err := fs.Sub(files, model.BundleDir)
	if err != nil {
		panic("quire: ship: " + err.Error())
	}
	shipped = b
}

func Run(site model.Site, mount func(*Engine) error, extra ...Option) {
	dev := flag.Bool("dev", false, "live reload")
	drafts := flag.Bool("drafts", false, "include draft documents")
	addr := flag.String("addr", ":8090", "listen address")
	flag.Parse()
	baking := flag.Arg(0) == "bake"
	if flag.NArg() > 1 || (flag.NArg() == 1 && !baking) {
		log.Fatalf("usage: %s [flags] [bake]", os.Args[0])
	}
	mode := Prod
	if *dev {
		mode = Dev
	}
	opts := append([]Option{WithMode(mode), WithDrafts(*drafts)}, extra...)
	b := New(site, opts...)
	if baking || (shipped == nil && bake.Enabled) {
		if err := b.bake(mount); err != nil {
			log.Fatal(err)
		}
		if baking {
			return
		}
	}
	bundle := shipped
	if bundle == nil {
		bundle = os.DirFS(b.bakeSpec.WithDefaults().Bundle())
	}
	e := New(site, append(opts, WithBundle(bundle))...)
	if err := mount(e); err != nil {
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
