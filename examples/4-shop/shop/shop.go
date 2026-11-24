// Package shop is the site: its prints, its pages and its basket. It is a
// package of its own, beside a main of a few lines, so that another program
// can run it as well.
package shop

import (
	"crypto/subtle"
	"embed"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/examples/4-shop/basket"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/service/sessions"
	"github.com/lahickey/quire/service/store"
	"github.com/lahickey/quire/site"
)

// Identity is who the site is.
var Identity = model.Identity{
	Name:        "Prints",
	BaseURL:     "https://example.com",
	Description: "Screenprints, and a basket that keeps up across tabs.",
}

//go:embed site.css
var css embed.FS

// piece is what a print's frontmatter says beyond the stock fields.
type piece struct {
	Medium string       `yaml:"medium"`
	Price  basket.Money `yaml:"price"`
}

// password is a gate: it lets a request through only with the password, by HTTP basic auth.
func password(want string) site.Gate {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, got, ok := r.BasicAuth(); !ok || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="the shop"`)
				http.Error(w, "the shop's own", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// declare says what the site holds: its content, its pages and its basket.
// It runs in a bake and when serving, so it only declares.
func declare(s *site.Site, carts *store.Memory, visitors *sessions.Sessions) error {
	// Every service is added, so whatever serves the site starts and stops it.
	s.AddServices(carts, visitors)
	s.AddStylesheets(css, "site.css")
	pieces, err := s.Collection(model.Spec{ID: "pieces", Title: "Pieces"})
	if err != nil {
		return err
	}
	// A piece's own fields are under extra in its frontmatter. Extra reads them
	// into a type of the site's, once, and a field it does not have fails the build.
	extra, err := pieces.Extra[piece]()
	if err != nil {
		return err
	}
	var items []basket.Item
	for _, d := range pieces.Docs() {
		items = append(items, basket.Item{ID: d.Slug, Title: d.Meta.Title, Price: extra[d.Slug].Price})
	}
	// Include runs the basket on the site: its endpoints, stylesheets, nav
	// link and its page at /basket. It is handed the services it uses.
	b := basket.New(carts, visitors, "/basket", items...)
	if err := s.Include(b); err != nil {
		return err
	}
	// The basket also has a page for whoever runs the shop, in the Admin
	// group. A group is served only once the site gives it a gate, so with no
	// password set there is nothing at /admin at all.
	if pw := os.Getenv("SHOP_ADMIN"); pw != "" {
		s.SetGroup(site.Admin, password(pw))
	}

	// A collection is its documents and nothing more; these are its pages.
	s.AddCollectionPages(pieces, wall(pieces), func(d model.Document) templ.Component { return piecePage(b, pieces, d) })
	s.AddPage(model.PageSpec{Title: "Prints", Path: "/"}, home(pieces))
	return nil
}

// Feature is the site's own. Its services are made here, once, and handed to
// what uses them. Making one opens nothing: that waits until the site is served.
func Feature() site.Feature {
	carts, visitors := store.NewMemory(), sessions.New()
	return site.FeatureFunc(func(s *site.Site) error { return declare(s, carts, visitors) })
}

// Options are what has to be settled before the feature runs.
func Options() []site.Option { return []site.Option{site.WithPalette(model.PaletteDusk)} }
