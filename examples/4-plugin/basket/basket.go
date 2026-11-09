// Package basket is a Quire plugin: a per-visitor basket whose count and lines
// update live in every open tab. Each button is a plain form, so it also works
// without JavaScript; Datastar takes it over when it loads.
package basket

import (
	"context"
	"embed"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
	"github.com/lahickey/quire/ui"
)

//go:embed basket.css basket-narrow.css
var css embed.FS

// Item is something the basket can hold.
type Item struct{ ID, Title string }

type Plugin struct {
	mount   *plugin.Mount
	streams *plugin.Streams
	items   []Item

	// Carts are in memory, keyed by session and then item: a restart empties them.
	mu    sync.Mutex
	carts map[string]map[string]int
}

func New(items ...Item) *Plugin { return &Plugin{items: items, carts: map[string]map[string]int{}} }

// Name is where the plugin is mounted: its routes are under /_basket/.
func (p *Plugin) Name() string { return "basket" }

func (p *Plugin) url(endpoint string) string { return p.mount.Prefix() + endpoint }

// stream is the Datastar action that keeps an event stream open, reconnecting if it drops.
func (p *Plugin) stream(endpoint string) string {
	return "@get('" + p.url(endpoint) + "', {retry: 'always', requestCancellation: 'cleanup'})"
}

// post is the Datastar action that sends a form without leaving the page.
func (p *Plugin) post(endpoint string) string {
	return "@post('" + p.url(endpoint) + "', {contentType: 'form'})"
}

func (p *Plugin) known(id string) bool {
	return slices.ContainsFunc(p.items, func(it Item) bool { return it.ID == id })
}

func (p *Plugin) cart(session string) map[string]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return maps.Clone(p.carts[session])
}

func count(cart map[string]int) int {
	n := 0
	for _, q := range cart {
		n += q
	}
	return n
}

// back is where a form post without JavaScript returns to: the page it came from, if on this site.
func (p *Plugin) back(r *http.Request) string {
	u, err := url.Parse(r.Header.Get("Referer"))
	if err != nil || (u.Host != "" && u.Host != r.Host) || !strings.HasPrefix(u.Path, "/") {
		return p.mount.PageURL("")
	}
	return u.Path
}

// publish pushes a changed cart to every tab of the session: the new count to the
// nav link, and the new lines to the basket page if it is open.
func (p *Plugin) publish(session string, cart map[string]int) {
	p.streams.Publish(session, func(s *plugin.Stream) error {
		return s.Signals(map[string]int{"count": count(cart)})
	})
	p.streams.Publish(session+"/lines", func(s *plugin.Stream) error {
		return s.Elements(context.Background(), p.lines(cart))
	})
}

// change adds delta of an item to the visitor's cart. Datastar gets an empty reply,
// because the change arrives over the event stream; a plain form post is redirected back.
func (p *Plugin) change(w http.ResponseWriter, r *http.Request, delta int) {
	session, id := plugin.Session(r), r.FormValue("item")
	if session == "" || !p.known(id) {
		http.Error(w, "no session or no such item", http.StatusBadRequest)
		return
	}
	p.mu.Lock()
	c := p.carts[session]
	if c == nil {
		c = map[string]int{}
		p.carts[session] = c
	}
	c[id] += delta
	if c[id] <= 0 {
		delete(c, id)
	}
	cart := maps.Clone(c)
	p.mu.Unlock()
	p.publish(session, cart)
	if plugin.IsDatastar(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, p.back(r), http.StatusSeeOther)
}

func (p *Plugin) add(w http.ResponseWriter, r *http.Request) { p.change(w, r, 1) }

func (p *Plugin) dec(w http.ResponseWriter, r *http.Request) { p.change(w, r, -1) }

// events is the event stream each tab subscribes to. It sends the current state
// on connect, then whatever publish sends.
func (p *Plugin) events(w http.ResponseWriter, r *http.Request) {
	session := plugin.Session(r)
	if r.URL.Query().Get("view") == "lines" {
		p.streams.Subscribe(w, r, session+"/lines", func(s *plugin.Stream) error {
			return s.Elements(r.Context(), p.lines(p.cart(session)))
		})
		return
	}
	p.streams.Subscribe(w, r, session, func(s *plugin.Stream) error {
		return s.Signals(map[string]int{"count": count(p.cart(session))})
	})
}

// show renders the basket page. It differs per visitor, so it must not be cached.
func (p *Plugin) show(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ui.Render(p.page(p.cart(plugin.Session(r))))(w, r)
}

// Attach is called by e.Use. UseSessions gives each visitor a session cookie,
// NewStreams the event streams publish sends to. Routes are endpoints under
// /_basket/; Page is the basket page visitors see, at /_basket unless the site
// moves it with plugin.At.
func (p *Plugin) Attach(m *plugin.Mount) error {
	p.mount = m
	p.streams = m.NewStreams()
	m.UseSessions()
	m.AddStylesheets(css, "basket.css")
	m.AddStylesheetsAt(model.WidthNarrow, css, "basket-narrow.css")
	m.AddNavEnd(p.nav())
	m.Route("GET", "events", p.events)
	m.Route("POST", "add", p.add)
	m.Route("POST", "dec", p.dec)
	return m.Page("GET", "", p.show)
}
