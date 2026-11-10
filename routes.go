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

func reserved(path string) bool {
	return strings.HasPrefix(path, "/_")
}

func mustBeOpen(closed bool, what string) {
	if closed {
		panic("quire: " + what + " added after the site was built")
	}
}

func (e *Engine) route(pattern, owner string, h http.HandlerFunc) {
	mustBeOpen(e.mounted, "route "+pattern)
	e.routes[pattern] = owner
	e.mux.HandleFunc(pattern, h)
}

func (e *Engine) claim(pattern, path, owner string, h http.HandlerFunc) error {
	if reserved(path) && !strings.HasPrefix(path+"/", "/_"+owner+"/") {
		return fmt.Errorf("%s belongs to another namespace", path)
	}
	if o, taken := e.routes[pattern]; taken {
		return fmt.Errorf("%s is already mounted by %s", path, o)
	}
	e.route(pattern, owner, h)
	return nil
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
		page = e.late(ui.NotFoundPage)
	}
	e.notFound404 = e.prepare("404", page, http.StatusNotFound)
	e.route("GET /{path...}", ownerQuire, e.notFound404.ServeHTTP)
}

func (e *Engine) robotsTxt() string { return serve.Robots(e.chrome.AbsURL("/sitemap.xml")) }

func (e *Engine) sitemapXML() string { return serve.Sitemap(e.chrome.AbsURL, e.sitemap) }

func (e *Engine) addSiteFiles() {
	if e.chrome.BaseURL == "" {
		return
	}
	e.route("GET /robots.txt", ownerQuire, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(e.robotsTxt()))
	})
	e.route("GET /sitemap.xml", ownerQuire, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write([]byte(e.sitemapXML()))
	})
}

func (e *Engine) AddRoute(method, path string, h http.HandlerFunc) {
	e.route(method+" "+path, ownerSite, h)
}

func (e *Engine) AddPage(path string, c templ.Component) {
	if c == nil {
		return
	}
	if reserved(path) {
		panic("quire: " + path + " is reserved for the engine and its plugins")
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
