// Package plugin is how a capability with its own routes, styles and state joins a Quire site.
package plugin

import (
	"cmp"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
)

type Host interface {
	Route(method, path string, h http.HandlerFunc)
	Claim(method, path, owner string, h http.HandlerFunc) error
	AddStylesheets(at model.Width, fsys fs.FS, names ...string)
	AddNavEnd(c templ.Component)
	UseSessions()
	Shell(p model.PageSpec) templ.Component
}

type Mount struct {
	h     Host
	name  string
	pages string
	err   error
}

type Plugin interface {
	Name() string
	Attach(*Mount) error
}

type moved struct {
	Plugin
	at string
}

func (p moved) Attach(m *Mount) error {
	m.pages = p.at
	return p.Plugin.Attach(m)
}

func At(path string, p Plugin) Plugin { return moved{p, path} }

func Attach(h Host, p Plugin) error {
	name := p.Name()
	if name == "" || strings.ContainsAny(name, "/_") {
		return fmt.Errorf("plugin name %q: one word, no slash or underscore", name)
	}
	m := &Mount{h: h, name: name}
	if err := cmp.Or(p.Attach(m), m.err); err != nil {
		return fmt.Errorf("plugin %s: %w", name, err)
	}
	return nil
}

var (
	preludes = regexp.MustCompile(`[^{};]*\{`)
	classes  = regexp.MustCompile(`\.(-?[A-Za-z_][\w-]*)`)
	comments = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

func (m *Mount) namespaced(fsys fs.FS, names []string) {
	for _, n := range names {
		css, err := fs.ReadFile(fsys, n)
		if err != nil {
			m.err = cmp.Or(m.err, err)
			continue
		}
		for _, pre := range preludes.FindAllString(comments.ReplaceAllString(string(css), ""), -1) {
			for _, c := range classes.FindAllStringSubmatch(pre, -1) {
				if !strings.HasPrefix(c[1], m.name+"-") {
					m.err = cmp.Or(m.err, fmt.Errorf("%s: class .%s is outside the %s- namespace", n, c[1], m.name))
				}
			}
		}
	}
}

func (m *Mount) Prefix() string { return "/_" + m.name + "/" }

func (m *Mount) Route(method, path string, h http.HandlerFunc) {
	m.h.Route(method, m.Prefix()+strings.TrimPrefix(path, "/"), h)
}

func (m *Mount) PageURL(path string) string {
	base := cmp.Or(m.pages, "/_"+m.name)
	if path = strings.Trim(path, "/"); path == "" {
		return base
	}
	return strings.TrimSuffix(base, "/") + "/" + path
}

func (m *Mount) Page(method, path string, h http.HandlerFunc) error {
	return m.h.Claim(method, m.PageURL(path), m.name, h)
}

func (m *Mount) AddStylesheets(fsys fs.FS, names ...string) {
	m.namespaced(fsys, names)
	m.h.AddStylesheets(0, fsys, names...)
}

func (m *Mount) AddStylesheetsAt(w model.Width, fsys fs.FS, names ...string) {
	m.namespaced(fsys, names)
	m.h.AddStylesheets(w, fsys, names...)
}

func (m *Mount) AddNavEnd(c templ.Component) { m.h.AddNavEnd(c) }

func (m *Mount) UseSessions() { m.h.UseSessions() }

func (m *Mount) Shell(p model.PageSpec) templ.Component { return m.h.Shell(p) }
