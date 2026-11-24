// A program with a web layer of its own that uses Quire for part of it. It has
// its own mux and its own page at /, and a Quire site mounted under /docs.
// Nothing here is baked: the site's pages are all Go, so it needs no bundle.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base"
	"github.com/lahickey/quire/site"
)

// docs is the site. It is written as if it were at /: its pages are at / and
// /install, and its links say so. Where it is put is not its business.
func docs(s *site.Site) error {
	s.AddPage(model.PageSpec{Title: "Docs", Path: "/"}, templ.Raw(
		`<article class="page"><h1>Docs</h1><p>Start with <a href="/install">installing it</a>.</p></article>`))
	s.AddPage(model.PageSpec{Title: "Install", Path: "/install"}, templ.Raw(
		`<article class="page"><h1>Install</h1><p>Back to <a href="/">the docs</a>.</p></article>`))
	return nil
}

func main() {
	addr := flag.String("addr", ":8090", "listen address")
	flag.Parse()

	// The place is the host's to choose. Root goes in front of the site's
	// pages. Engine is where its stylesheet, scripts and fonts go: under /docs
	// too, because this program owns everything else.
	at := model.Place{Root: "/docs", Engine: "/docs/_quire"}
	built, err := site.Describe(model.Identity{Name: "Docs"}, site.FeatureFunc(docs),
		site.WithFeatures(base.Pack{}), site.WithPlace(at)).Build(nil)
	if err != nil {
		log.Fatal(err)
	}

	// A site has no server in it. Its routes are a table, already placed, and
	// any mux can hold them.
	mux := http.NewServeMux()
	for _, r := range built.Routes() {
		mux.HandleFunc(r.Pattern, r.Handler)
	}

	// The host's own page, outside the site, drawn with the site's document,
	// frame and stylesheet.
	home := built.Page(model.PageSpec{Title: "The host"}, templ.Raw(
		`<article class="page"><h1>The host's own page</h1><p>The <a href="/">docs</a> are a Quire site.</p></article>`))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := home.Render(r.Context(), w); err != nil {
			log.Print(err)
		}
	})

	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
