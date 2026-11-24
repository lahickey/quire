// Two sites in one server. Each is written as if it were alone at /, with a
// content folder of its own, and a root says where it is served.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base"
)

func main() {
	// RunSites is Run for several sites: it bakes each, into a bundle of its
	// own, and serves them all.
	quire.RunSites(
		// The first site is the box's own, at the top: the health check and
		// the page for an address nobody has are its.
		quire.Site{Identity: model.Identity{Name: "Front"}, Feature: base.Content(), Content: "front"},
		// The second is under /notes. Its pages link to / and /first, and are
		// served at /notes and /notes/first: nothing in notes/ says so. Its
		// stylesheet, scripts and fonts go in a part of /_quire/ of its own,
		// so everything Quire serves for itself on this box is under one path.
		quire.Site{Identity: model.Identity{Name: "Notes"}, Feature: base.Content(), Content: "notes", Root: "/notes"},
	)
}
