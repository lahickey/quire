// Package basket is a feature: a per-visitor basket whose count and lines
// update live in every open tab. Each button is a plain form, so it also works
// without JavaScript; Datastar takes it over when it loads.
package basket

import (
	"context"
	"embed"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sync"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/libs/streams"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/service/sessions"
	"github.com/lahickey/quire/service/store"
	"github.com/lahickey/quire/site"
)

//go:embed basket.css basket-narrow.css
var css embed.FS

// Money is an amount in pence.
type Money int

// Item is something the basket can hold.
type Item struct {
	ID, Title string
	Price     Money
}

type Feature struct {
	at       string
	items    []Item
	carts    store.Store
	visitors *sessions.Sessions
	streams  *streams.Streams
	log      *slog.Logger

	// One change to a cart at a time: a change reads the cart, alters it and writes it back.
	mu sync.Mutex
}

type row struct {
	visitor string
	count   int
}

func (p *Feature) known(id string) bool {
	return slices.ContainsFunc(p.items, func(it Item) bool { return it.ID == id })
}

// cart is a visitor's, as it is kept: JSON under their id.
func (p *Feature) cart(ctx context.Context, visitor string) map[string]int {
	cart := map[string]int{}
	if kept, ok, err := p.carts.Get(ctx, visitor); err != nil {
		p.log.Error("reading a cart", "err", err)
	} else if ok {
		if err := json.Unmarshal(kept, &cart); err != nil {
			p.log.Error("reading a cart", "err", err)
		}
	}
	return cart
}

func (p *Feature) keep(ctx context.Context, visitor string, cart map[string]int) error {
	if len(cart) == 0 {
		return p.carts.Delete(ctx, visitor)
	}
	kept, err := json.Marshal(cart)
	if err != nil {
		return err
	}
	return p.carts.Put(ctx, visitor, kept)
}

func (m Money) String() string { return fmt.Sprintf("£%d.%02d", m/100, m%100) }

func (p *Feature) total(cart map[string]int) Money {
	var t Money
	for _, it := range p.items {
		t += it.Price * Money(cart[it.ID])
	}
	return t
}

func count(cart map[string]int) int {
	n := 0
	for _, q := range cart {
		n += q
	}
	return n
}

// publish pushes a changed cart to every tab the visitor has open: the new count to the
// nav link, and the new lines to the basket page if it is open.
func (p *Feature) publish(visitor string, cart map[string]int) {
	p.streams.Publish(visitor, func(s *streams.Stream) error {
		return s.Signals(map[string]int{"count": count(cart)})
	})
	p.streams.Publish(visitor+"/lines", func(s *streams.Stream) error {
		return s.Elements(context.Background(), p.lines(cart))
	})
}

// change adds delta of an item to the visitor's cart. Datastar gets an empty reply,
// because the change arrives over the event stream; a plain form post is redirected back.
func (p *Feature) change(w http.ResponseWriter, r *http.Request, delta int) {
	visitor, id := p.visitors.ID(w, r), r.FormValue("item")
	if !p.known(id) {
		http.Error(w, "no such item", http.StatusBadRequest)
		return
	}
	p.mu.Lock()
	cart := p.cart(r.Context(), visitor)
	if cart[id] += delta; cart[id] <= 0 {
		delete(cart, id)
	}
	err := p.keep(r.Context(), visitor, cart)
	p.mu.Unlock()
	if err != nil {
		p.log.Error("keeping a cart", "err", err)
		http.Error(w, "the basket could not be kept", http.StatusInternalServerError)
		return
	}
	p.publish(visitor, cart)
	streams.Done(w, r)
}

func (p *Feature) add(w http.ResponseWriter, r *http.Request) { p.change(w, r, 1) }

func (p *Feature) dec(w http.ResponseWriter, r *http.Request) { p.change(w, r, -1) }

// events is the event stream each tab subscribes to. It sends the current state
// on connect, then whatever publish sends. ID gives a first-time visitor their
// cookie, so it is asked before the stream begins.
func (p *Feature) events(w http.ResponseWriter, r *http.Request) {
	visitor := p.visitors.ID(w, r)
	if r.URL.Query().Get("view") == "lines" {
		p.streams.Subscribe(w, r, visitor+"/lines", func(s *streams.Stream) error {
			return s.Elements(r.Context(), p.lines(p.cart(r.Context(), visitor)))
		})
		return
	}
	p.streams.Subscribe(w, r, visitor, func(s *streams.Stream) error {
		return s.Signals(map[string]int{"count": count(p.cart(r.Context(), visitor))})
	})
}

// The basket page differs per visitor, so it is drawn for each request.
func (p *Feature) livePage(r *http.Request) (model.PageSpec, templ.Component) {
	return model.PageSpec{Title: "Basket", Path: p.at, NoIndex: true}, p.page(p.cart(r.Context(), p.visitors.Read(r)))
}

// The shop's own page lists every basket there is.
func (p *Feature) adminPage(r *http.Request) (model.PageSpec, templ.Component) {
	var rows []row
	for visitor, kept := range p.carts.List(r.Context(), "") {
		var cart map[string]int
		if err := json.Unmarshal(kept, &cart); err == nil {
			rows = append(rows, row{visitor, count(cart)})
		}
	}
	return model.PageSpec{Title: "Baskets", NoIndex: true}, p.baskets(rows)
}

// New is a basket of these items, with its page at the address given. It keeps
// its carts in the store, under its own name, and asks visitors who each one is.
func New(carts store.Store, visitors *sessions.Sessions, at string, items ...Item) *Feature {
	return &Feature{at: at, items: items, carts: store.Under(carts, "basket"), visitors: visitors}
}

// Name is the feature's namespace: its routes are under /_basket/, its
// classes begin basket-, and the site refuses anything of its outside that.
func (p *Feature) Name() string { return "basket" }

// Declare runs in a bake as well as when serving, so it only declares.
// Streams is the event streams publish sends to, and URL an address under
// /_basket/. In(site.Admin) is the Admin group, which is served only if the
// site gives it a gate: there the page is at /admin/basket.
func (p *Feature) Declare(s *site.Site) error {
	p.streams, p.log = s.Streams(), s.Log()
	s.AddStylesheets(css, "basket.css")
	s.AddStylesheetsAt(model.WidthNarrow, css, "basket-narrow.css")
	s.AddNavEnd(p.nav())
	s.AddRoute("GET", s.URL("events"), p.events)
	s.AddRoute("POST", s.URL("add"), p.add)
	s.AddRoute("POST", s.URL("dec"), p.dec)
	s.AddLivePage(p.at, p.livePage)
	s.In(site.Admin).AddLivePage("", p.adminPage)
	return nil
}
