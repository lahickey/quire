package quire

import (
	"cmp"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
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

func (s *Site) reserve(pattern, owner string) error {
	mustBeOpen(s.built, "route "+pattern)
	_, path, _ := strings.Cut(pattern, " ")
	if strings.HasPrefix(path, "/_") && !strings.HasPrefix(path+"/", "/_"+owner+"/") {
		return fmt.Errorf("%s belongs to another namespace", path)
	}
	if o, taken := s.routes[pattern]; taken {
		return fmt.Errorf("%s is already taken by %s", path, o)
	}
	s.routes[pattern] = owner
	return nil
}

func (s *Site) claim(pattern, owner string, h http.HandlerFunc) error {
	if err := s.reserve(pattern, owner); err != nil {
		return err
	}
	s.handlers = append(s.handlers, route{pattern: pattern, handler: func(w http.ResponseWriter, r *http.Request) {
		h(w, r.WithContext(model.WithChrome(r.Context(), &s.chrome)))
	}})
	return nil
}

func (s *Site) route(pattern, owner string, h http.HandlerFunc) {
	s.err = cmp.Or(s.err, s.claim(pattern, owner, h))
}

func (s *Site) versioned(url, hash string) string {
	s.versions[url] = hash
	return url + "?h=" + hash
}

func (s *Site) addBytes(url, contentType string, body []byte) string {
	s.route("GET "+url, ownerQuire, serve.Static(contentType, body).ServeHTTP)
	return s.versioned(url, journal.Hash(body))
}

func (s *Site) prepare(path string, c templ.Component, code int) *serve.Prepared {
	p := serve.Prepare(path, c, code)
	s.fixed = append(s.fixed, p)
	return p
}

func render(c templ.Component) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := c.Render(r.Context(), w); err != nil {
			log.Printf("render: %v", err)
		}
	}
}

// AddPage adds a fixed page at the address its head data names, drawn once and kept, in the navigation and the sitemap.
func (s *Site) AddPage(p model.PageSpec, body templ.Component) {
	if body == nil {
		s.err = cmp.Or(s.err, fmt.Errorf("page %s: no body", p.Path))
		return
	}
	s.route("GET "+pagePattern(p.Path), ownerSite, s.prepare(p.Path, s.Page(p, body), http.StatusOK).ServeHTTP)
	s.sitemap = append(s.sitemap, serve.Loc{Path: p.Path})
	s.addNavItem(p.Path)
}

// AddLivePage adds a page drawn for each request, and never kept. Its head data may depend on the request.
func (s *Site) AddLivePage(path string, page func(*http.Request) (model.PageSpec, templ.Component)) {
	s.route("GET "+pagePattern(path), ownerSite, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		render(s.Page(page(r)))(w, r)
	})
}

// AddRoute is the raw mux: nothing added, never kept.
func (s *Site) AddRoute(method, path string, h http.HandlerFunc) {
	s.route(method+" "+path, ownerSite, h)
}

func (s *Site) AddFile(urlPath string, fsys fs.FS, name string) string {
	body, err := fs.ReadFile(fsys, name)
	if err != nil {
		s.err = cmp.Or(s.err, fmt.Errorf("file %s: %w", urlPath, err))
		return ""
	}
	s.route("GET "+urlPath, ownerSite, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, fsys, name)
	})
	return s.versioned(urlPath, journal.Hash(body))
}

// SetNotFound is the body of the page for an address nothing answers.
func (s *Site) SetNotFound(body templ.Component) {
	mustBeOpen(s.built, "the not-found page")
	s.notFound.set(body, s.depth == 0)
}

// SetDefaultHome is the body of the page at / when the site is built and nothing has taken it.
func (s *Site) SetDefaultHome(body templ.Component) {
	mustBeOpen(s.built, "the default home page")
	s.home.set(body, s.depth == 0)
}
