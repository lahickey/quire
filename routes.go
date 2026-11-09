package quire

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/andybalholm/brotli"
	"github.com/lahickey/quire/internal/compress"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/ui"
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

func (e *Engine) route(pattern string, cached bool, h http.HandlerFunc) {
	mustBeOpen(e.mounted, "route "+pattern)
	e.routes[pattern] = cached
	e.mux.HandleFunc(pattern, h)
}

func (e *Engine) claimed(pattern string) bool {
	_, ok := e.routes[pattern]
	return ok
}

func (e *Engine) versioned(url, hash string) string {
	e.versions[url] = hash
	return url + "?h=" + hash
}

func (e *Engine) addBytes(url, contentType string, body []byte) string {
	v := compress.Prepare(body, contentType, brotli.BestCompression)
	e.route("GET "+url, false, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(compress.Negotiate(w, r, body, v))
	})
	return e.versioned(url, serve.Hash(body))
}

func (e *Engine) prepare(path string, c templ.Component, code int) *serve.Prepared {
	p := serve.Prepare(path, c, code)
	e.fixed = append(e.fixed, p)
	return p
}

func (e *Engine) preparePage(path string, c templ.Component) {
	e.route("GET "+pagePattern(path), false, e.prepare(path, c, http.StatusOK).ServeHTTP)
}

func (e *Engine) serveNotFound(w http.ResponseWriter, r *http.Request) {
	if e.notFound404 == nil {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (e *Engine) addNotFound() {
	page := e.notFound
	if page == nil {
		page = e.late(ui.NotFoundPage)
	}
	e.notFound404 = e.prepare("404", page, http.StatusNotFound)
	e.route("GET /{path...}", true, e.serveNotFound)
}

func (e *Engine) robotsTxt() string { return serve.Robots(e.site.AbsURL("/sitemap.xml")) }

func (e *Engine) sitemapXML() string { return serve.Sitemap(e.site.AbsURL, e.sitemap) }

func (e *Engine) addSiteFiles() {
	if e.site.BaseURL == "" {
		return
	}
	e.route("GET /robots.txt", false, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(e.robotsTxt()))
	})
	e.route("GET /sitemap.xml", false, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write([]byte(e.sitemapXML()))
	})
}

func (e *Engine) AddRoute(method, path string, h http.HandlerFunc) {
	e.route(method+" "+path, false, h)
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
	if _, err := fs.Stat(fsys, name); err != nil {
		panic("quire: file " + urlPath + ": " + err.Error())
	}
	e.route("GET "+urlPath, false, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, fsys, name)
	})
	return e.versioned(urlPath, serve.FileHash(fsys, name))
}

func (e *Engine) SetNotFound(c templ.Component) {
	mustBeOpen(e.mounted, "the not-found page")
	e.notFound = c
}
