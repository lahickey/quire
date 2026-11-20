package quire

import (
	"cmp"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/internal/session"
)

const (
	healthPath = assetPrefix + "health"
	pageCache  = 4 << 20
)

// Server serves one site, and owns what claims the whole address space.
type Server struct {
	site   *Site
	mux    *http.ServeMux
	pages  *serve.Pages
	apex   string
	secure bool
}

func NewServer(site *Site) (*Server, error) {
	if !site.built {
		return nil, errors.New("quire: a server takes a site from Build")
	}
	v := &Server{site: site, mux: http.NewServeMux(), pages: serve.NewPages(pageCache, site.dev)}
	if !site.dev {
		v.apex, v.secure = site.chrome.Host(), strings.HasPrefix(site.chrome.BaseURL, "https://")
	}
	for _, r := range site.handlers {
		if r.cached {
			r.handler = v.pages.Cached(r.handler)
		}
		v.mux.HandleFunc(r.pattern, r.handler)
	}
	var err error
	own := func(pattern string, h http.HandlerFunc) {
		if owner, taken := site.routes[pattern]; taken {
			err = cmp.Or(err, fmt.Errorf("%s is already taken by %s", pattern, owner))
			return
		}
		v.mux.HandleFunc(pattern, h)
	}
	own("GET "+healthPath, serve.Health)
	own("GET /{path...}", site.notFound404.ServeHTTP)
	if site.chrome.BaseURL != "" {
		robots := serve.Robots(site.chrome.AbsURL("/sitemap.xml"))
		own("GET /robots.txt", serve.Static("text/plain; charset=utf-8", []byte(robots)).ServeHTTP)
		sitemap := serve.Sitemap(site.chrome.AbsURL, site.sitemap)
		own("GET /sitemap.xml", serve.Static("application/xml; charset=utf-8", []byte(sitemap)).ServeHTTP)
	}
	return v, err
}

func (v *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if target, ok := serve.Redirect(r, v.apex); ok {
		w.Header().Set("Cache-Control", "no-cache")
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	}
	control := "no-store"
	if !v.site.dev && r.URL.Path != healthPath {
		control = serve.CacheControl(r, v.site.versions)
	}
	w.Header().Set("Cache-Control", control)
	if v.site.sessions {
		r = session.Mint(w, r, v.secure)
	}
	v.mux.ServeHTTP(w, r)
}
