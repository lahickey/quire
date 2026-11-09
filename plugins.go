package quire

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
)

type pluginHost struct{ e *Engine }

func (h pluginHost) Route(method, path string, f http.HandlerFunc) { h.e.AddRoute(method, path, f) }

func (h pluginHost) Claim(method, path, owner string, f http.HandlerFunc) error {
	if reserved(path) && !strings.HasPrefix(path+"/", "/_"+owner+"/") {
		return fmt.Errorf("%s belongs to another namespace", path)
	}
	pattern := method + " " + pagePattern(path)
	if o, taken := h.e.claims[pattern]; taken {
		return fmt.Errorf("%s is already mounted by %s", path, o)
	}
	if h.e.claimed(pattern) {
		return fmt.Errorf("%s is already mounted by the site", path)
	}
	h.e.claims[pattern] = owner
	h.e.route(pattern, false, f)
	return nil
}

func (h pluginHost) AddStylesheets(at model.Width, fsys fs.FS, names ...string) {
	mustBeOpen(h.e.mounted, "stylesheet")
	h.e.addSheets("plugin", at, fsys, names)
}

func (h pluginHost) AddNavEnd(c templ.Component) {
	mustBeOpen(h.e.mounted, "nav control")
	if c != nil {
		h.e.navEnd = append(h.e.navEnd, c)
	}
}

func (h pluginHost) UseSessions() { h.e.sessions = true }

func (h pluginHost) Shell(p model.PageSpec) templ.Component { return h.e.Shell(p) }

func (e *Engine) Use(plugins ...plugin.Plugin) error {
	for _, p := range plugins {
		if err := plugin.Attach(pluginHost{e}, p); err != nil {
			return err
		}
	}
	return nil
}
