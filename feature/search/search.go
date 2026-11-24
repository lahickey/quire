// Package search is a feature that finds things in a site's documents: results as the visitor types, and a
// results page for a plain form submission. The index is every section of every collection it is given, held in memory.
// It finds and does not draw: how its box and its results look is a Look, which the site's look supplies.
package search

import (
	"cmp"
	"html"
	"io/fs"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/site"
)

const (
	limit = 8
	lead  = 50
	tail  = 110
)

var (
	heading = regexp.MustCompile(`(?s)<h[23] id="([^"]*)"[^>]*>(.*?)</h[23]>`)
	tags    = regexp.MustCompile(`(?s)<svg.*?</svg>|<[^>]*>`)
	// unlisted is the opening tag of an element that is kept out of the index.
	unlisted = regexp.MustCompile(`<(\w+)[^>]*\sdata-nosearch[^>]*>`)
)

type section struct{ page, heading, url, text string }

// Hit is one matching section; Match is the part of its text to mark.
type Hit struct{ Page, Heading, URL, Before, Match, After string }

// Look is how a search is drawn. Box is the control in the nav, a form that fetches live as the visitor types and
// submits to action without JavaScript. Results is what replaces the box's results, and Page the body of the results page.
type Look struct {
	Box     func(action, live string) templ.Component
	Results func(q string, hits []Hit) templ.Component
	Page    func(q string, hits []Hit) templ.Component
	Files   fs.FS
	Sheets  []string
	Scripts []string
}

type Feature struct {
	at       string
	look     Look
	pending  []func()
	indexed  sync.Once
	sections []section
}

func plain(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(tags.ReplaceAllString(s, ""))), " ")
}

func terms(q string) []*regexp.Regexp {
	var out []*regexp.Regexp
	for t := range strings.FieldsSeq(q) {
		if re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(t)); err == nil {
			out = append(out, re)
		}
	}
	return out
}

func (s section) score(terms []*regexp.Regexp) int {
	total := 0
	for _, t := range terms {
		n := 0
		if t.MatchString(cmp.Or(s.heading, s.page)) {
			n += 8
		}
		if t.MatchString(s.page) {
			n += 4
		}
		if t.MatchString(s.text) {
			n++
		}
		if n == 0 {
			return 0
		}
		total += n
	}
	return total
}

func (s section) hit(term *regexp.Regexp) Hit {
	loc := term.FindStringIndex(s.text)
	if loc == nil {
		loc = []int{0, 0}
	}
	before, after := s.text[max(0, loc[0]-lead):loc[0]], s.text[loc[1]:min(len(s.text), loc[1]+tail)]
	if loc[0] > lead {
		_, before, _ = strings.Cut(before, " ")
	}
	if len(s.text)-loc[1] > tail {
		if cut, _, ok := strings.CutLast(after, " "); ok {
			after = cut
		} else {
			after = ""
		}
	}
	return Hit{s.page, s.heading, s.url, before, s.text[loc[0]:loc[1]], after}
}

func (p *Feature) find(q string) []Hit {
	p.indexed.Do(func() {
		for _, index := range p.pending {
			index()
		}
	})
	terms := terms(q)
	if len(terms) == 0 {
		return nil
	}
	type scored struct {
		section
		score int
	}
	var found []scored
	for _, s := range p.sections {
		if n := s.score(terms); n > 0 {
			found = append(found, scored{s, n})
		}
	}
	slices.SortStableFunc(found, func(a, b scored) int { return b.score - a.score })
	var hits []Hit
	for _, s := range found[:min(limit, len(found))] {
		hits = append(hits, s.hit(terms[0]))
	}
	return hits
}

func (p *Feature) live(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("q")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = p.look.Results(q, p.find(q)).Render(r.Context(), w)
}

func (p *Feature) page(r *http.Request) (model.PageSpec, templ.Component) {
	q := r.FormValue("q")
	return model.PageSpec{Title: "Search", Path: p.at, NoIndex: true}, p.look.Page(q, p.find(q))
}

// listed is s without its elements marked data-nosearch, and all in them.
func listed(s string) string {
	for {
		m := unlisted.FindStringSubmatchIndex(s)
		if m == nil {
			return s
		}
		end, closing := len(s), "</"+s[m[2]:m[3]]+">"
		if i := strings.Index(s[m[1]:], closing); i >= 0 {
			end = m[1] + i + len(closing)
		}
		s = s[:m[0]] + s[end:]
	}
}

// New is a search drawn by look, with its results page at the address given.
func New(at string, look Look) *Feature { return &Feature{at: at, look: look} }

// Add gives the search a collection to find things in: one section for each document's opening, and one for each H2 and H3.
// An element marked data-nosearch is left out. The indexing waits for the first search, so a bake does none of it.
func (p *Feature) Add(c *model.Collection) {
	p.pending = append(p.pending, func() { index(p, c) })
}

func index(p *Feature, c *model.Collection) {
	for _, d := range c.Docs() {
		m := d.Meta
		body, at := m.Summary+"\n"+listed(d.HTML), 0
		sec := section{page: m.Title, url: c.Href(d)}
		for _, h := range heading.FindAllStringSubmatchIndex(body, -1) {
			sec.text = plain(body[at:h[0]])
			p.sections = append(p.sections, sec)
			sec = section{page: m.Title, heading: plain(body[h[4]:h[5]]), url: c.Href(d) + "#" + body[h[2]:h[3]]}
			at = h[1]
		}
		sec.text = plain(body[at:])
		p.sections = append(p.sections, sec)
	}
}

func (p *Feature) Name() string { return "search" }

func (p *Feature) Declare(s *site.Site) error {
	live := s.URL("live")
	s.UseDatastar()
	s.AddStylesheets(p.look.Files, p.look.Sheets...)
	s.AddScripts(p.look.Files, p.look.Scripts...)
	s.AddNavEnd(p.look.Box(p.at, live))
	s.AddRoute("GET", live, p.live)
	s.AddLivePage(p.at, p.page)
	return nil
}
