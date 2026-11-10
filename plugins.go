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

const core = "quire"

type pluginHost struct{ e *Engine }

type liveReload struct {
	e    *Engine
	boot string
}

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

func (liveReload) Name() string { return core }

func (l liveReload) Attach(m *plugin.Mount) error {
	streams := m.NewStreams()
	l.e.reload = model.LiveReload{On: true, BootID: l.boot, Init: m.Listen("reload")}
	m.Route("GET", "reload", func(w http.ResponseWriter, r *http.Request) {
		streams.Subscribe(w, r, "boot", func(s *plugin.Stream) error { return s.Signals(map[string]string{"boot": l.boot}) })
	})
	return nil
}

func (e *Engine) Use(plugins ...plugin.Plugin) error {
	for _, p := range plugins {
		if p.Name() == core {
			return fmt.Errorf("plugin name %q is reserved for Quire's own plugins", core)
		}
		if err := plugin.Attach(pluginHost{e}, p); err != nil {
			return err
		}
	}
	return nil
}
