// Package server serves sites and runs their services, and owns what claims the whole address space: the page cache, the catch-all, robots and the health check.
package server

import (
	"cmp"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"

	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/site"
)

const pageCache = 4 << 20

type service struct {
	site.Service
	env model.Env
}

// Server serves one site or several, each where it was placed. The first is the box's own: the health check, robots.txt,
// the www redirect and the page for an address no site has are its.
type Server struct {
	mux      *http.ServeMux
	pages    *serve.Pages
	services []service
	apex     string
	health   string
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

// same says two services are one value, so a service handed to several sites is started once.
func same(a, b site.Service) bool {
	t := reflect.TypeOf(a)
	return t == reflect.TypeOf(b) && t.Comparable() && a == b
}

func (v *Server) stop(ctx context.Context, services []service) error {
	var errs []error
	for _, svc := range slices.Backward(services) {
		errs = append(errs, svc.Stop(ctx))
	}
	return errors.Join(errs...)
}

func New(sites ...*site.Site) (*Server, error) {
	if len(sites) == 0 {
		return nil, errors.New("quire: a server takes at least one site")
	}
	own := sites[0]
	v := &Server{mux: http.NewServeMux(), pages: serve.NewPages(pageCache, own.Dev()), health: own.Chrome().Out(model.EnginePath + "health")}
	if !own.Dev() {
		v.apex = own.Identity().Host()
	}
	var err error
	taken := map[string]bool{}
	add := func(s *site.Site, pattern string, h http.HandlerFunc) {
		if taken[pattern] {
			err = cmp.Or(err, fmt.Errorf("two routes are at %s", pattern))
			return
		}
		taken[pattern] = true
		v.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", s.CacheControl(r))
			h(w, r)
		})
	}
	var maps []string
	for _, s := range sites {
		at := s.Chrome()
		for _, r := range s.Routes() {
			if r.Shared {
				r.Handler = v.pages.Cached(r.Handler)
			}
			add(s, r.Pattern, r.Handler)
		}
		if at.Root != "" {
			add(s, "GET "+at.Root+"/{path...}", s.NotFound().ServeHTTP)
		}
		if at.BaseURL != "" {
			maps = append(maps, "\nSitemap: "+at.AbsURL("/sitemap.xml")+"\n")
			add(s, "GET "+at.Out("/sitemap.xml"), serve.Static("application/xml; charset=utf-8", []byte(sitemap(at.AbsURL, s.Locs()))).ServeHTTP)
		}
		for _, svc := range s.Services() {
			if !slices.ContainsFunc(v.services, func(x service) bool { return same(x.Service, svc) }) {
				v.services = append(v.services, service{svc, s.Env()})
			}
		}
	}
	add(own, "GET /{path...}", own.NotFound().ServeHTTP)
	if len(maps) > 0 {
		robots := "User-agent: *\nAllow: /\n" + strings.Join(maps, "")
		add(own, "GET "+own.Chrome().Out("/robots.txt"), serve.Static("text/plain; charset=utf-8", []byte(robots)).ServeHTTP)
	}
	v.mux.HandleFunc("GET "+v.health, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		serve.Health(w, r)
	})
	return v, err
}

// Start starts every site's services in the order they were added, each told the environment of the site that added it first.
// If one fails, those already started are stopped.
func (v *Server) Start(ctx context.Context) error {
	for i, svc := range v.services {
		if err := svc.Start(ctx, svc.env); err != nil {
			return errors.Join(err, v.stop(ctx, v.services[:i]))
		}
	}
	return nil
}

// Stop stops the services, last started first.
func (v *Server) Stop(ctx context.Context) error { return v.stop(ctx, v.services) }

func (v *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if target, ok := serve.Redirect(r, v.apex); ok {
		w.Header().Set("Cache-Control", "no-cache")
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	}
	v.mux.ServeHTTP(w, r)
}
