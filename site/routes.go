package site

import (
	"cmp"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/libs/streams"
	"github.com/lahickey/quire/model"
)

const ownerQuire = "quire"

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
		return fmt.Errorf("%s is in a namespace that is not its own", path)
	}
	if o, taken := s.routes[pattern]; taken {
		return fmt.Errorf("%s is already taken by %s", path, cmp.Or(o, "the site"))
	}
	s.routes[pattern] = owner
	return nil
}

func (s *Site) claim(pattern, owner string, h http.HandlerFunc) error {
	if err := s.reserve(pattern, owner); err != nil {
		return err
	}
	s.handlers = append(s.handlers, Route{Pattern: pattern, Handler: func(w http.ResponseWriter, r *http.Request) {
		h(w, r.WithContext(model.WithChrome(r.Context(), &s.chrome)))
	}})
	return nil
}

func (s *Site) route(pattern, owner string, h http.HandlerFunc) {
	if err := s.claim(pattern, owner, h); err != nil {
		s.fail("%w", err)
	}
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

func (s *Site) live(page func(*http.Request) (model.PageSpec, templ.Component)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		s.render(s.Page(page(r)))(w, r)
	}
}

func (s *Site) render(c templ.Component) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := c.Render(r.Context(), w); err != nil {
			s.log.Error("render", "path", r.URL.Path, "err", err)
		}
	}
}

// AddPage adds a fixed page at the address its head data names, drawn once and kept, in the navigation and the sitemap.
func (s *Site) AddPage(p model.PageSpec, body templ.Component) {
	if body == nil {
		s.fail("page %s: no body", p.Path)
		return
	}
	s.route("GET "+pagePattern(p.Path), s.owner, s.prepare(p.Path, s.Page(p, body), http.StatusOK).ServeHTTP)
	s.sitemap = append(s.sitemap, Loc{Path: p.Path})
	s.addNavItem(p.Path)
}

// AddLivePage adds a page drawn for each request, and never kept. Its head data may depend on the request.
func (s *Site) AddLivePage(path string, page func(*http.Request) (model.PageSpec, templ.Component)) {
	s.route("GET "+pagePattern(path), s.owner, s.live(page))
}

// AddRoute is the raw mux: nothing added, never kept.
func (s *Site) AddRoute(method, path string, h http.HandlerFunc) {
	s.route(method+" "+path, s.owner, h)
}

func (s *Site) AddFile(urlPath string, fsys fs.FS, name string) string {
	body, err := fs.ReadFile(fsys, name)
	if err != nil {
		s.fail("file %s: %w", urlPath, err)
		return ""
	}
	s.route("GET "+urlPath, s.owner, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, fsys, name)
	})
	return s.versioned(urlPath, journal.Hash(body))
}

// SetNotFound is the body of the page for an address nothing answers.
func (s *Site) SetNotFound(body templ.Component) { s.notFound.set(body, s.own("the not-found page")) }

// SetDefaultHome is the body of the page at / when the site is built and nothing has taken it.
func (s *Site) SetDefaultHome(body templ.Component) { s.home.set(body, s.own("the default home page")) }

// Routes is the site's table, for a server or a mux of somebody else's.
func (s *Site) Routes() []Route { return s.handlers }

// NotFound answers an address nothing in the table does.
func (s *Site) NotFound() http.Handler { return s.notFound404 }

func (s *Site) Locs() []Loc { return s.sitemap }

// CacheControl is how long a response to the request may be kept: for good at a hashed address, and not at all in dev.
func (s *Site) CacheControl(r *http.Request) string {
	if s.dev {
		return "no-store"
	}
	return serve.CacheControl(r, s.versions)
}

func (s *Site) Dev() bool { return s.dev }

// URL is an address in the namespace of the feature now declaring, which has to have a name.
func (s *Site) URL(path string) string {
	mustBeOpen(s.built, "an address")
	if s.owner == "" {
		s.fail("a namespace is a named feature's, and this one has no name")
	}
	return "/_" + s.owner + "/" + strings.TrimPrefix(path, "/")
}

// Streams is the live updates of the feature now declaring, scoped by its name. It links Datastar on every page.
func (s *Site) Streams() *streams.Streams {
	s.datastar = true
	return streams.New(s.owner, s.URL(""))
}
