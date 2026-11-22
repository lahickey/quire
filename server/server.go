// Package server serves a site and runs its services, and owns what claims the whole address space: the page cache, the catch-all, robots and the sitemap.
package server

import (
	"cmp"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/site"
)

const pageCache = 4 << 20

// Server serves one site.
type Server struct {
	site   *site.Site
	health string
	mux    *http.ServeMux
	pages  *serve.Pages
	apex   string
}

func sitemap(abs func(string) string, locs []site.Loc) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	seen := map[string]bool{}
	for _, loc := range locs {
		if seen[loc.Path] {
			continue
		}
		seen[loc.Path] = true
		b.WriteString("  <url><loc>")
		_ = xml.EscapeText(&b, []byte(abs(loc.Path)))
		b.WriteString("</loc>")
		if loc.LastMod != "" {
			b.WriteString("<lastmod>")
			_ = xml.EscapeText(&b, []byte(loc.LastMod))
			b.WriteString("</lastmod>")
		}
		b.WriteString("</url>\n")
	}
	b.WriteString("</urlset>\n")
	return b.String()
}

func New(s *site.Site) (*Server, error) {
	id, at := s.Identity(), s.Chrome()
	v := &Server{site: s, mux: http.NewServeMux(), pages: serve.NewPages(pageCache, s.Dev()), health: at.Out(model.EnginePath + "health")}
	if !s.Dev() {
		v.apex = id.Host()
	}
	taken := map[string]bool{}
	for _, r := range s.Routes() {
		taken[r.Pattern] = true
		if r.Shared {
			r.Handler = v.pages.Cached(r.Handler)
		}
		v.mux.HandleFunc(r.Pattern, r.Handler)
	}
	var err error
	own := func(pattern string, h http.HandlerFunc) {
		if taken[pattern] {
			err = cmp.Or(err, fmt.Errorf("the site already has a route at %s", pattern))
			return
		}
		v.mux.HandleFunc(pattern, h)
	}
	own("GET "+v.health, serve.Health)
	own("GET /{path...}", s.NotFound().ServeHTTP)
	if id.BaseURL != "" {
		robots := "User-agent: *\nAllow: /\n\nSitemap: " + at.AbsURL("/sitemap.xml") + "\n"
		own("GET /robots.txt", serve.Static("text/plain; charset=utf-8", []byte(robots)).ServeHTTP)
		own("GET "+at.Out("/sitemap.xml"), serve.Static("application/xml; charset=utf-8", []byte(sitemap(at.AbsURL, s.Locs()))).ServeHTTP)
	}
	return v, err
}

func (v *Server) stop(ctx context.Context, services []site.Service) error {
	var errs []error
	for _, svc := range slices.Backward(services) {
		errs = append(errs, svc.Stop(ctx))
	}
	return errors.Join(errs...)
}

// Start starts the site's services in the order they were added. If one fails, those already started are stopped.
func (v *Server) Start(ctx context.Context) error {
	env := v.site.Env()
	for i, svc := range v.site.Services() {
		if err := svc.Start(ctx, env); err != nil {
			return errors.Join(err, v.stop(ctx, v.site.Services()[:i]))
		}
	}
	return nil
}

// Stop stops the site's services, last started first.
func (v *Server) Stop(ctx context.Context) error { return v.stop(ctx, v.site.Services()) }

func (v *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if target, ok := serve.Redirect(r, v.apex); ok {
		w.Header().Set("Cache-Control", "no-cache")
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	}
	control := "no-store"
	if r.URL.Path != v.health {
		control = v.site.CacheControl(r)
	}
	w.Header().Set("Cache-Control", control)
	v.mux.ServeHTTP(w, r)
}
