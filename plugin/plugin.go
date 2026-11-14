// Package plugin is how a capability with its own routes, styles and state joins a Quire site.
package plugin

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/session"
	"github.com/lahickey/quire/model"
)

var (
	preludes = regexp.MustCompile(`[^{};]*\{`)
	classes  = regexp.MustCompile(`\.(-?[A-Za-z_][\w-]*)`)
	comments = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

type Route struct {
	Method, Path string
	Handler      http.HandlerFunc
}

type Sheet struct {
	At    model.Width
	FS    fs.FS
	Names []string
}

type Attached struct {
	Name     string
	Routes   []Route
	Sheets   []Sheet
	Scripts  []Sheet
	NavEnd   []templ.Component
	Sessions bool
}

type Mount struct {
	got   Attached
	shell model.Shell
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

func (m *Mount) namespaced(fsys fs.FS, names []string) {
	for _, n := range names {
		css, err := fs.ReadFile(fsys, n)
		if err != nil {
			m.err = cmp.Or(m.err, err)
			continue
		}
		for _, pre := range preludes.FindAllString(comments.ReplaceAllString(string(css), ""), -1) {
			for _, c := range classes.FindAllStringSubmatch(pre, -1) {
				if !strings.HasPrefix(c[1], m.got.Name+"-") {
					m.err = cmp.Or(m.err, fmt.Errorf("%s: class .%s is outside the %s- namespace", n, c[1], m.got.Name))
				}
			}
		}
	}
}

func (p moved) Attach(m *Mount) error {
	m.pages = p.at
	return p.Plugin.Attach(m)
}

func At(path string, p Plugin) Plugin { return moved{p, path} }

func Attach(p Plugin, shell model.Shell) (Attached, error) {
	name := p.Name()
	if name == "" || strings.ContainsAny(name, "/_") {
		return Attached{}, fmt.Errorf("plugin name %q: one word, no slash or underscore", name)
	}
	m := &Mount{got: Attached{Name: name}, shell: shell}
	if err := cmp.Or(p.Attach(m), m.err); err != nil {
		return Attached{}, fmt.Errorf("plugin %s: %w", name, err)
	}
	return m.got, nil
}

func Session(r *http.Request) string { return session.ID(r) }

func (m *Mount) URL(path string) string {
	return "/_" + m.got.Name + "/" + strings.TrimPrefix(path, "/")
}

func (m *Mount) Route(method, path string, h http.HandlerFunc) {
	m.got.Routes = append(m.got.Routes, Route{method, m.URL(path), h})
}

func (m *Mount) PageURL(path string) string {
	base := cmp.Or(m.pages, "/_"+m.got.Name)
	if path = strings.Trim(path, "/"); path == "" {
		return base
	}
	return strings.TrimSuffix(base, "/") + "/" + path
}

func (m *Mount) Page(method, path string, h http.HandlerFunc) {
	m.got.Routes = append(m.got.Routes, Route{method, m.PageURL(path), h})
}

func (m *Mount) AddStylesheets(fsys fs.FS, names ...string) { m.AddStylesheetsAt(0, fsys, names...) }

func (m *Mount) AddStylesheetsAt(w model.Width, fsys fs.FS, names ...string) {
	m.namespaced(fsys, names)
	m.got.Sheets = append(m.got.Sheets, Sheet{w, fsys, names})
}

func (m *Mount) AddScripts(fsys fs.FS, names ...string) {
	for _, n := range names {
		if _, err := fs.Stat(fsys, n); err != nil {
			m.err = cmp.Or(m.err, err)
		}
	}
	m.got.Scripts = append(m.got.Scripts, Sheet{FS: fsys, Names: names})
}

func (m *Mount) AddNavEnd(c templ.Component) {
	if c == nil {
		m.err = cmp.Or(m.err, errors.New("nil nav control"))
	}
	m.got.NavEnd = append(m.got.NavEnd, c)
}

func (m *Mount) UseSessions() { m.got.Sessions = true }

func (m *Mount) Shell(p model.PageSpec) templ.Component { return m.shell(p) }
