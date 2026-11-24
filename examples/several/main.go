// Two sites in one server. Each is written as if it were alone at /, and a
// place says where it is served. Nothing here is baked: the pages are all Go.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base"
	"github.com/lahickey/quire/server"
	"github.com/lahickey/quire/site"
)

func front(s *site.Site) error {
	s.AddPage(model.PageSpec{Title: "Front", Path: "/"}, templ.Raw(
		`<article class="page"><h1>The front site</h1><p>There are <a href="/notes">notes</a> too, which are a site of their own.</p></article>`))
	return nil
}

// notes links to its own pages as / and /first. Served under /notes, they are
// /notes and /notes/first, and nothing here says so.
func notes(s *site.Site) error {
	s.AddPage(model.PageSpec{Title: "Notes", Path: "/"}, templ.Raw(
		`<article class="page"><h1>Notes</h1><p>Read <a href="/first">the first</a>.</p></article>`))
	s.AddPage(model.PageSpec{Title: "First", Path: "/first"}, templ.Raw(
		`<article class="page"><h1>The first note</h1><p>Back to <a href="/">the notes</a>.</p></article>`))
	return nil
}

func build(name string, f site.FeatureFunc, at model.Place) *site.Site {
	built, err := site.Describe(model.Identity{Name: name}, f, site.WithFeatures(base.Pack{}), site.WithPlace(at)).Build(nil)
	if err != nil {
		log.Fatal(err)
	}
	return built
}

func main() {
	addr := flag.String("addr", ":8090", "listen address")
	flag.Parse()

	// The first site is the box's own, at the top: the health check and the
	// page for an address nobody has are its. The second is under /notes. Its
	// stylesheet, scripts and fonts go in a part of /_quire/ of its own, so
	// everything Quire serves for itself on this box is under the one path.
	v, err := server.New(
		build("Front", front, model.Place{}),
		build("Notes", notes, model.Place{Root: "/notes", Engine: "/_quire/notes"}),
	)
	if err != nil {
		log.Fatal(err)
	}
	// Start starts every site's services. These two have none, and it is
	// still the call that makes a server ready.
	if err := v.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	log.Println("listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, v))
}
