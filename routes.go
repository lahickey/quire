package quire

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"
	"github.com/andybalholm/brotli"
	"github.com/lahickey/quire/internal/compress"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/ui"
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

func (e *Engine) AddRoute(method, path string, h http.HandlerFunc) {
	pattern := method + " " + path
	mustBeOpen(e.mounted, "route "+pattern)
	e.routes[pattern] = false
	e.mux.HandleFunc(pattern, h)
}

func (e *Engine) addCachedRoute(pattern string, h http.HandlerFunc) {
	mustBeOpen(e.mounted, "route "+pattern)
	e.routes[pattern] = true
	e.mux.HandleFunc(pattern, h)
}

func (e *Engine) claimed(pattern string) bool {
	_, ok := e.routes[pattern]
	return ok
}

func (e *Engine) versioned(url, hash string) string {
	e.linked.hashes[url] = hash
	return url + "?h=" + hash
}

func (e *Engine) serveBytes(url, contentType string, body []byte) string {
	v := compress.Prepare(body, contentType, brotli.BestCompression)
	e.AddRoute("GET", url, func(w http.ResponseWriter, r *http.Request) {
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
	e.AddRoute("GET", pagePattern(path), e.prepare(path, c, http.StatusOK).ServeHTTP)
}

func (e *Engine) addPageLoc(path string) {
	e.pageLocs = append(e.pageLocs, path)
}

func (e *Engine) AddPage(path string, c templ.Component) {
	if c == nil {
		return
	}
	if reserved(path) {
		panic("quire: " + path + " is reserved for the engine and its plugins")
	}
	e.preparePage(path, c)
	e.addPageLoc(path)
	e.addNavItem(path)
}

func (e *Engine) AddFile(urlPath string, fsys fs.FS, name string) string {
	if _, err := fs.Stat(fsys, name); err != nil {
		panic("quire: file " + urlPath + ": " + err.Error())
	}
	e.AddRoute("GET", urlPath, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, fsys, name)
	})
	return e.versioned(urlPath, serve.FileHash(fsys, name))
}

func (e *Engine) serveNotFound(w http.ResponseWriter, r *http.Request) {
	if e.notFound404 == nil {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (e *Engine) SetNotFound(c templ.Component) {
	mustBeOpen(e.mounted, "the not-found page")
	e.notFound = c
}

func (e *Engine) bakeNotFound() {
	page := e.notFound
	if page == nil {
		page = e.late(ui.NotFoundPage)
	}
	e.notFound404 = e.prepare("404", page, http.StatusNotFound)
	e.addCachedRoute("GET /{path...}", e.serveNotFound)
}

func (e *Engine) robotsTxt() string { return serve.Robots(e.site.AbsURL("/sitemap.xml")) }

func (e *Engine) sitemapXML() string {
	locs := make([]serve.Loc, 0, len(e.pageLocs)+len(e.sitemap))
	for _, p := range e.pageLocs {
		locs = append(locs, serve.Loc{Path: p})
	}
	return serve.Sitemap(e.site.AbsURL, append(locs, e.sitemap...))
}

func (e *Engine) ensureSiteFiles() {
	if e.siteFiles || e.site.BaseURL == "" {
		return
	}
	e.siteFiles = true
	e.AddRoute("GET", "/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(e.robotsTxt()))
	})
	e.AddRoute("GET", "/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write([]byte(e.sitemapXML()))
	})
}

func (e *Engine) mountDefaults() {
	if e.mounted {
		return
	}
	e.buildAssets()
	e.ensureSiteFiles()
	e.AddRoute("GET", model.ImagePath+"{file}", e.bundle.ServeHTTP)
	if !e.claimed("GET " + pagePattern("/")) {
		e.AddPage("/", e.late(ui.Home))
	}
	e.bakeNotFound()
	e.mounted = true
	for _, p := range e.fixed {
		p.Build()
	}
}

func (e *Engine) addImage(name string, max ...model.Slot) (model.Image, error) {
	name = path.Clean(filepath.ToSlash(name))
	want := model.SlotDefault
	if len(max) > 0 && max[0] >= model.SlotThumb {
		want = max[0]
	}
	if im, ok := e.bundle.Mounted(name, want); ok {
		return im, nil
	}
	r, err := e.from.Master(name)
	if err != nil {
		return model.Image{}, err
	}
	if len(r.Formats) == 0 {
		return model.Image{}, fmt.Errorf("quire: image %s names no formats", name)
	}
	im := model.NewImage(r, e.scale, want, e.slots)
	e.bundle.Mount(name, im, want)
	return im, nil
}

func (e *Engine) addSitemap(locs ...serve.Loc) {
	e.sitemap = append(e.sitemap, locs...)
}

func (e *Engine) AddCollection[T model.Metadata](spec model.Spec[T]) (*model.Collection[T], error) {
	col, err := model.Collect(spec, host{e}, e.site, e.drafts)
	if err != nil {
		return nil, err
	}
	docs := col.Docs()
	if len(docs) == 0 {
		return col, nil
	}
	e.addNavItem(spec.ID)

	index, item := spec.Index, spec.Item
	if index == nil {
		index = ui.Listing[T]
	}
	if item == nil {
		item = ui.Article[T]
	}
	e.preparePage("/"+spec.ID, index(e.Shell, col))

	e.addSitemap(serve.Loc{Path: "/" + spec.ID})
	if spec.ListOnly {
		return col, nil
	}
	e.addCachedRoute("GET /"+spec.ID+"/{slug}", func(w http.ResponseWriter, r *http.Request) {
		doc, ok := col.Get(r.PathValue("slug"))
		if !ok {
			e.serveNotFound(w, r)
			return
		}
		ui.Render(item(e.Shell, col, doc))(w, r)
	})
	for _, d := range docs {
		if !d.Meta.IsDraft() {
			e.addSitemap(serve.Loc{Path: col.Href(d), LastMod: d.Meta.LastMod().ISO()})
		}
	}
	return col, nil
}
