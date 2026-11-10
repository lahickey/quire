// Package search is a Quire plugin: a search box in the nav whose results arrive
// as the visitor types, and a results page for a plain form submission.
// The index is every section of every collection it is given, held in memory.
package search

import (
	"cmp"
	"embed"
	"html"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
	"github.com/lahickey/quire/ui"
)

const (
	limit = 8
	lead  = 50
	tail  = 110
)

//go:embed search.css
var css embed.FS

var (
	heading = regexp.MustCompile(`(?s)<h[23] id="([^"]*)"[^>]*>(.*?)</h[23]>`)
	tags    = regexp.MustCompile(`(?s)<svg.*?</svg>|<[^>]*>`)
)

type section struct{ page, heading, url, text string }

// Hit is one matching section; Match is the part of its text to mark.
type Hit struct{ Page, Heading, URL, Before, Match, After string }

type Plugin struct {
	mount    *plugin.Mount
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

func (p *Plugin) find(q string) []Hit {
	terms := terms(q)
	if len(terms) == 0 {
		return nil
	}
	found := slices.DeleteFunc(slices.Clone(p.sections), func(s section) bool { return s.score(terms) == 0 })
	slices.SortStableFunc(found, func(a, b section) int { return b.score(terms) - a.score(terms) })
	var hits []Hit
	for _, s := range found[:min(limit, len(found))] {
		hits = append(hits, s.hit(terms[0]))
	}
	return hits
}

func (p *Plugin) live(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("q")
	ui.Render(p.results(q, p.find(q)))(w, r)
}

func (p *Plugin) show(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("q")
	w.Header().Set("Cache-Control", "no-store")
	ui.Render(p.page(q, p.find(q)))(w, r)
}

func New() *Plugin { return &Plugin{} }

// Add indexes a collection: one section for each document's opening, and one for each H2 and H3.
func (p *Plugin) Add[T model.Metadata](c *model.Collection[T]) {
	for _, d := range c.Docs() {
		m := d.Meta.Base()
		body, at := m.Summary+"\n"+d.HTML, 0
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

func (p *Plugin) Name() string { return "search" }

func (p *Plugin) Attach(m *plugin.Mount) error {
	p.mount = m
	m.AddStylesheets(css, "search.css")
	m.AddNavEnd(p.box())
	m.Route("GET", "live", p.live)
	m.Page("GET", "", p.show)
	return nil
}
