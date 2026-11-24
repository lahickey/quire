// A shop: a collection with its own frontmatter type and templates, and a
// basket feature that adds routes, a page and live updates, and is handed
// the services it uses: a store for its carts, and sessions.
// The basket lives in its own package because a feature is reusable: any
// site could include it.
package main

import (
	"crypto/subtle"
	"embed"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/examples/4-feature/basket"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/service/sessions"
	"github.com/lahickey/quire/service/store"
	"github.com/lahickey/quire/site"
)

//go:embed site.css
var css embed.FS

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
	var items []basket.Item
	for _, d := range pieces.Docs() {
		items = append(items, basket.Item{ID: d.Slug, Title: d.Meta.Title})
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
	s.AddPage(pieces.Page(), wall(pieces))
	s.AddDocumentPages(pieces, func(d model.Document) (model.PageSpec, templ.Component) {
		return model.DocumentPage(pieces, d), piecePage(b, pieces, d)
	})
	s.AddPage(model.PageSpec{Title: "Prints", Path: "/"}, home(pieces))
	return nil
}

func main() {
	// Services are made here, once, and handed to what uses them. Making one
	// opens nothing: that waits until the site is served.
	carts, visitors := store.NewMemory(), sessions.New()
	quire.Run(model.Identity{
		Name:        "Prints",
		BaseURL:     "https://example.com",
		Description: "Screenprints, and a basket that keeps up across tabs.",
	}, site.FeatureFunc(func(s *site.Site) error { return declare(s, carts, visitors) }),
		site.WithPalette(model.PaletteDusk),
	)
}
