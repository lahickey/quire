package site

import (
	"net/http"
	"path"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
)

// Group is a part of a site behind one gate, at the address of its name: Admin is /admin.
// Features add to a group and the site gives it its gate. A group with no gate is not served.
// A group of a site's own is one more constant of this type.
type Group string

const (
	Members Group = "members"
	Admin   Group = "admin"
)

// Gate decides who a group answers: it wraps the handler of every route in it.
type Gate = func(http.Handler) http.Handler

// Area is a group as one feature adds to it. A named feature's addresses are under the group's and then its own name: /admin/basket/.
type Area struct {
	s     *Site
	group Group
	owner string
}

type gated struct {
	group          Group
	pattern, owner string
	handler        http.HandlerFunc
}

func (s *Site) group(g Group) Group {
	if g == "" || strings.ContainsAny(string(g), "/_ ") {
		s.fail("group name %q: one word, no slash or underscore", g)
	}
	return g
}

// gate serves each group that was given a gate, and leaves the rest unserved.
func (s *Site) gate() {
	for g := range s.gates {
		for pattern := range s.routes {
			_, at, _ := strings.Cut(pattern, " ")
			if strings.HasPrefix(at+"/", "/"+string(g)+"/") {
				s.fail("%s is inside the group %s, and outside its gate: add it with In", at, g)
			}
		}
	}
	for _, r := range s.gated {
		gate, ok := s.gates[r.group]
		if !ok {
			continue
		}
		if err := s.claim(r.pattern, r.owner, gate.v(r.handler).ServeHTTP); err != nil {
			s.fail("%w", err)
			continue
		}
		s.handlers[len(s.handlers)-1].Group = r.group
	}
}

// SetGroup gives a group its gate, which is what makes it served. It is the site's to say, and the site's word beats a pack's.
func (s *Site) SetGroup(g Group, gate Gate) {
	own := s.own("the gate of group " + string(g))
	if gate == nil {
		s.fail("group %s: no gate", g)
		return
	}
	x := s.gates[s.group(g)]
	x.set(gate, own)
	s.gates[g] = x
}

// In is a group for the feature now declaring to add to.
func (s *Site) In(g Group) Area {
	mustBeOpen(s.built, "a group's route")
	return Area{s, s.group(g), s.owner}
}

func (a Area) URL(at string) string { return path.Join("/"+string(a.group), a.owner, at) }

// AddRoute is a route in the group, behind its gate.
func (a Area) AddRoute(method, at string, h http.HandlerFunc) {
	mustBeOpen(a.s.built, "a group's route")
	a.s.gated = append(a.s.gated, gated{a.group, method + " " + a.URL(at), a.owner, h})
}

// AddLivePage is a page in the group, drawn for each request. It is in no navigation and no sitemap.
func (a Area) AddLivePage(at string, page func(*http.Request) (model.PageSpec, templ.Component)) {
	a.AddRoute("GET", at, a.s.live(page))
}
