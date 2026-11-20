package quire

import (
	"cmp"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/bundle"
	"github.com/lahickey/quire/internal/serve"
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
	s.handlers = append(s.handlers, route{pattern: pattern, handler: h})
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
	return s.versioned(url, bundle.Hash(body))
}

func (s *Site) prepare(path string, c templ.Component, code int) *serve.Prepared {
	p := serve.Prepare(path, c, code)
	s.fixed = append(s.fixed, p)
	return p
}

func (s *Site) preparePage(path string, c templ.Component) {
	s.route("GET "+pagePattern(path), ownerSite, s.prepare(path, c, http.StatusOK).ServeHTTP)
}

func (s *Site) AddPage(path string, c templ.Component) {
	if c == nil {
		s.err = cmp.Or(s.err, fmt.Errorf("page %s: nil component", path))
		return
	}
	s.preparePage(path, c)
	s.sitemap = append(s.sitemap, serve.Loc{Path: path})
	s.addNavItem(path)
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
	return s.versioned(urlPath, bundle.Hash(body))
}

func (s *Site) SetNotFound(c templ.Component) {
	mustBeOpen(s.built, "the not-found page")
	s.notFound = c
}
