package quire

import (
	"fmt"
	"net/http"

	"github.com/lahickey/quire/internal/assets"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
)

type liveReload struct {
	e    *Engine
	boot string
}

func (e *Engine) attach(p plugin.Plugin) error {
	mustBeOpen(e.mounted, "plugin "+p.Name())
	a, err := plugin.Attach(p, e.Shell)
	if err != nil {
		return err
	}
	for _, r := range a.Routes {
		if err := e.claim(r.Method+" "+pagePattern(r.Path), a.Name, r.Handler); err != nil {
			return fmt.Errorf("plugin %s: %w", a.Name, err)
		}
	}
	for _, s := range a.Sheets {
		e.addSheets("plugin", s.At, s.FS, s.Names)
	}
	for _, s := range a.Scripts {
		for _, n := range s.Names {
			e.scripts = append(e.scripts, assets.File{FS: s.FS, Name: n})
		}
	}
	e.chrome.NavEnd = append(e.chrome.NavEnd, a.NavEnd...)
	e.sessions = e.sessions || a.Sessions
	return nil
}

func (liveReload) Name() string { return ownerQuire }

func (l liveReload) Attach(m *plugin.Mount) error {
	streams := m.NewStreams()
	l.e.chrome.LiveReload = model.LiveReload{On: true, BootID: l.boot, Init: m.Listen("reload")}
	m.Route("GET", "reload", func(w http.ResponseWriter, r *http.Request) {
		streams.Subscribe(w, r, "boot", func(s *plugin.Stream) error { return s.Signals(map[string]string{"boot": l.boot}) })
	})
	return nil
}

func (e *Engine) Use(plugins ...plugin.Plugin) error {
	for _, p := range plugins {
		if p.Name() == ownerQuire || p.Name() == ownerSite {
			return fmt.Errorf("plugin name %q is reserved", p.Name())
		}
		if err := e.attach(p); err != nil {
			return err
		}
	}
	return nil
}
