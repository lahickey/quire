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
}

type Engine struct {
	chrome   model.Chrome
	mode     Mode
	drafts   bool
	palette  model.Palette
	scale    model.Scale
	siteCard *model.ShareCard
	sheets   []assets.Sheet
	shipped  fs.FS

	images map[string]model.Image
	fonts  fonts.FontSet
	bundle *serve.Store
	pages  *serve.Pages

	bakeSpec model.BakeSpec
	from     source

	mux         *http.ServeMux
	routes      map[string]string
	versions    map[string]string
	sessions    bool
	sitemap     []serve.Loc
	fixed       []*serve.Prepared
	notFound    templ.Component
	notFound404 *serve.Prepared
	mounted     bool
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
	e.route("GET "+model.ImagePath+"{file}", ownerQuire, e.bundle.ServeHTTP)
	if _, taken := e.routes["GET "+pagePattern("/")]; !taken {
		e.AddPage("/", e.late(ui.Home))
	}
	e.addNotFound()
	e.mounted = true
	for _, p := range e.fixed {
		p.Build()
	}
}

func (e *Engine) siteCardText() model.ShareCard {
	c := model.ShareCard{Title: e.chrome.Name, Lines: []string{e.chrome.Description}}
	if e.siteCard != nil {
		c = *e.siteCard
	}
	c.Domain = cmp.Or(c.Domain, e.chrome.Host())
	if c.Ink == (model.Ink{}) {
		c.Ink = e.palette.Ink()
	}
	return c
}

func New(site model.Site, opts ...Option) *Engine {
	e := &Engine{
		mux:      http.NewServeMux(),
		routes:   map[string]string{},
		images:   map[string]model.Image{},
		versions: map[string]string{},
		chrome:   model.Chrome{Site: site.WithDefaults(), Header: ui.Header, Footer: ui.Footer},
		palette:  model.PaletteEmber,
		scale:    model.DefaultScale(),
	}
	for _, o := range opts {
		o(e)
	}

	e.palette.Check()
	set, err := fonts.ReadFontSet(fonts.Default())
	if err != nil {
		panic("quire: fonts: " + err.Error())
	}
	e.fonts = set
	e.bundle = serve.NewStore(imageMemory, imageMemoryLimit, e.mode == Dev, e.fonts.ID)
	e.from = e.bundle

	if e.shipped != nil {
		if err := e.bundle.Load(e.shipped); err != nil {
			panic("quire: bundle: " + err.Error() + "; run the site's bake")
		}
	}
	rec, _ := e.bundle.Record(journal.SiteCard)
	if file, ok := rec.File(); ok {
		e.chrome.OGImageURL = e.chrome.AbsURL(model.ImageURL(file))
	}
	e.pages = serve.NewPages(pageCache, e.mode == Dev)
	e.route("GET "+healthPath, ownerQuire, serve.Health)
	if e.mode == Dev {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			panic("engine: boot id: " + err.Error())
		}
		if err := e.attach(liveReload{e, hex.EncodeToString(b[:])}); err != nil {
			panic("engine: live reload: " + err.Error())
		}
	}
	return e
}

func (e *Engine) Handler() http.Handler {
	e.mountDefaults()
	apex := e.chrome.Host()
	if e.mode == Dev {
		apex = ""
	}
	secure := e.mode != Dev && strings.HasPrefix(e.chrome.BaseURL, "https://")
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
	if baking || (shipped == nil && canBake) {
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
	e := New(site, append(opts, withBundle(bundle))...)
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
