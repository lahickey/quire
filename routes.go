package quire

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/ui"
)

const (
	ownerQuire = "quire"
	ownerSite  = "site"
)

func pagePattern(path string) string {
	if path == "/" {
		return "/{$}"
	}
	return strings.TrimSuffix(path, "/")
}

func mustBeOpen(closed bool, what string) {
	if closed {
		panic("quire: " + what + " added after the site was built")
	}
}

func (e *Engine) claim(pattern, owner string, h http.HandlerFunc) error {
	mustBeOpen(e.mounted, "route "+pattern)
	_, path, _ := strings.Cut(pattern, " ")
	if strings.HasPrefix(path, "/_") && !strings.HasPrefix(path+"/", "/_"+owner+"/") {
		return fmt.Errorf("%s belongs to another namespace", path)
	}
	if o, taken := e.routes[pattern]; taken {
		return fmt.Errorf("%s is already mounted by %s", path, o)
	}
	e.routes[pattern] = owner
	e.mux.HandleFunc(pattern, h)
	return nil
}

func (e *Engine) route(pattern, owner string, h http.HandlerFunc) {
	if err := e.claim(pattern, owner, h); err != nil {
		panic("quire: " + err.Error())
	}
}

func (e *Engine) versioned(url, hash string) string {
	e.versions[url] = hash
	return url + "?h=" + hash
}

func (e *Engine) addBytes(url, contentType string, body []byte) string {
	e.route("GET "+url, ownerQuire, serve.Static(contentType, body).ServeHTTP)
	return e.versioned(url, journal.Hash(body))
}

func (e *Engine) prepare(path string, c templ.Component, code int) *serve.Prepared {
	p := serve.Prepare(path, c, code)
	e.fixed = append(e.fixed, p)
	return p
}

func (e *Engine) preparePage(path string, c templ.Component) {
	e.route("GET "+pagePattern(path), ownerSite, e.prepare(path, c, http.StatusOK).ServeHTTP)
}

func (e *Engine) addNotFound() {
	page := e.notFound
	if page == nil {
		page = e.late(ui.NotFound)
	}
	e.notFound404 = e.prepare("404", page, http.StatusNotFound)
	e.route("GET /{path...}", ownerQuire, e.notFound404.ServeHTTP)
}

func (e *Engine) addSiteFiles() {
	if e.chrome.BaseURL == "" {
		return
	}
	robots := serve.Robots(e.chrome.AbsURL("/sitemap.xml"))
	e.route("GET /robots.txt", ownerQuire, serve.Static("text/plain; charset=utf-8", []byte(robots)).ServeHTTP)
	sitemap := serve.Sitemap(e.chrome.AbsURL, e.sitemap)
	e.route("GET /sitemap.xml", ownerQuire, serve.Static("application/xml; charset=utf-8", []byte(sitemap)).ServeHTTP)
}

func (e *Engine) AddRoute(method, path string, h http.HandlerFunc) {
	e.route(method+" "+path, ownerSite, h)
}

func (e *Engine) AddPage(path string, c templ.Component) {
	if c == nil {
		panic("quire: page " + path + ": nil component")
	}
	e.preparePage(path, c)
	e.sitemap = append(e.sitemap, serve.Loc{Path: path})
	e.addNavItem(path)
}

func (e *Engine) AddFile(urlPath string, fsys fs.FS, name string) string {
	body, err := fs.ReadFile(fsys, name)
	if err != nil {
		panic("quire: file " + urlPath + ": " + err.Error())
	}
	e.route("GET "+urlPath, ownerSite, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, fsys, name)
	})
	return e.versioned(urlPath, journal.Hash(body))
}

func (e *Engine) SetNotFound(c templ.Component) {
	mustBeOpen(e.mounted, "the not-found page")
	e.notFound = c
}
