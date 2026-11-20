package quire

import (
	"fmt"
	"net/http"

	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
)

// Live is what only a served site has. It is handed to a blueprint's Live.
type Live struct{ s *Site }

type liveReload struct {
	s    *Site
	boot string
}

func (s *Site) attach(p plugin.Plugin) error {
	mustBeOpen(s.built, "plugin "+p.Name())
	a, err := plugin.Attach(p, s.Shell)
	if err != nil {
		return err
	}
	for _, r := range a.Routes {
		if err := s.claim(r.Method+" "+pagePattern(r.Path), a.Name, r.Handler); err != nil {
			return fmt.Errorf("plugin %s: %w", a.Name, err)
		}
	}
	for _, sh := range a.Sheets {
		s.addSheets("plugin", sh.At, sh.FS, sh.Names)
	}
	for _, sc := range a.Scripts {
		for _, n := range sc.Names {
			s.scripts = append(s.scripts, assets.File{FS: sc.FS, Name: n})
		}
	}
	s.chrome.NavEnd = append(s.chrome.NavEnd, a.NavEnd...)
	s.sessions = s.sessions || a.Sessions
	return nil
}

func (liveReload) Name() string { return ownerQuire }

func (l liveReload) Attach(m *plugin.Mount) error {
	streams := m.NewStreams()
	l.s.chrome.LiveReload = model.LiveReload{On: true, BootID: l.boot, Init: m.Listen("reload")}
	m.Route("GET", "reload", func(w http.ResponseWriter, r *http.Request) {
		streams.Subscribe(w, r, "boot", func(s *plugin.Stream) error { return s.Signals(map[string]string{"boot": l.boot}) })
	})
	return nil
}

func (l *Live) Use(plugins ...plugin.Plugin) error {
	for _, p := range plugins {
		if p.Name() == ownerQuire || p.Name() == ownerSite {
			return fmt.Errorf("plugin name %q is reserved", p.Name())
		}
		if err := l.s.attach(p); err != nil {
			return err
		}
	}
	return nil
}

func (l *Live) AddRoute(method, path string, h http.HandlerFunc) {
	l.s.route(method+" "+path, ownerSite, h)
}
