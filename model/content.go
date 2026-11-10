// Package model is the data a Quire site is made of: the site, collections, documents, images and the look.
package model

import (
	"fmt"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/journal"
)

type Marker struct {
	Text string
	Kind string
}

type Metadata interface {
	Base() Meta
	Badges() []Marker
}

type Meta struct {
	Title    string   `yaml:"title"`
	Draft    bool     `yaml:"draft"`
	Featured bool     `yaml:"featured"`
	Date     Date     `yaml:"date"`
	Updated  Date     `yaml:"updated"`
	Summary  string   `yaml:"summary"`
	Image    string   `yaml:"image"`
	Tags     []string `yaml:"tag"`
}

type Heading = journal.Heading

type Widget = func(info, body string) (string, error)

type Document[T any] struct {
	Slug     string
	Path     string
	Meta     T
	HTML     string
	Words    int
	Headings []Heading

	Image Image
}

type Collection[T Metadata] struct {
	id     string
	title  string
	desc   string
	docs   []Document[T]
	bySlug map[string]Document[T]
	site   Site
	kicker func(Document[T]) string
	tags   *Classifier
}

type Spec[T Metadata] struct {
	ID          string
	Title       string
	Description string
	Markdown    string
	Widgets     map[string]Widget
	Classifier  *Classifier
	Kicker      func(Document[T]) string
	Sort        func(a, b Document[T]) int
	Index       func(Shell, *Collection[T]) templ.Component
	Item        func(Shell, *Collection[T], Document[T]) templ.Component
}

type Basic = Spec[Meta]

func published[T Metadata](docs []Document[T], drafts bool) []Document[T] {
	if drafts {
		return docs
	}
	n := 0
	for _, d := range docs {
		if d.Meta.Base().Draft {
			continue
		}
		docs[n] = d
		n++
	}
	return docs[:n]
}

func checkTags[T Metadata](id string, docs []Document[T], c *Classifier) error {
	if c == nil {
		return nil
	}
	var tags []string
	for _, d := range docs {
		tags = append(tags, d.Meta.Base().Tags...)
	}
	if err := c.check(tags...); err != nil {
		return fmt.Errorf("%s: %w", id, err)
	}
	return nil
}

func modified(m Meta) string {
	if mod := m.LastMod(); !mod.Equal(m.Date.Time) {
		return mod.ISO()
	}
	return ""
}

func images[T Metadata](image func(string) (Image, error), id string, docs []Document[T]) error {
	seen := map[string]Image{}
	for i, d := range docs {
		name := d.Meta.Base().Image
		if name == "" {
			continue
		}
		name = journal.Resolve(d.Path, name)
		im, ok := seen[name]
		if !ok {
			var err error
			if im, err = image(name); err != nil {
				return fmt.Errorf("%s/%s: %w", id, d.Slug, err)
			}
			seen[name] = im
		}
		docs[i].Image = im
	}
	return nil
}

func (m Meta) Base() Meta { return m }

func (m Meta) Badges() []Marker { return nil }

func (m Meta) LastMod() Date {
	if !m.Updated.IsZero() {
		return m.Updated
	}
	return m.Date
}

func (c *Collection[T]) Docs() []Document[T] { return c.docs }

func (c *Collection[T]) Len() int { return len(c.docs) }

func (c *Collection[T]) Get(slug string) (Document[T], bool) {
	d, ok := c.bySlug[slug]
	return d, ok
}

func (c *Collection[T]) Neighbours(d Document[T]) (prev, next *Document[T]) {
	i := slices.IndexFunc(c.docs, func(o Document[T]) bool { return o.Slug == d.Slug })
	if i > 0 {
		prev = &c.docs[i-1]
	}
	if i >= 0 && i < len(c.docs)-1 {
		next = &c.docs[i+1]
	}
	return prev, next
}

func (c *Collection[T]) Href(d Document[T]) string { return "/" + c.id + "/" + d.Slug }

func (c *Collection[T]) LongDate(d Date) string { return d.Formatted(c.site.Dates.Long) }

func (c *Collection[T]) ShortDate(d Date) string { return d.Formatted(c.site.Dates.Short) }

func (c *Collection[T]) ShortMonthYear(d Date) string {
	return d.Formatted(c.site.Dates.ShortMonthYear)
}

func (c *Collection[T]) Featured() []Document[T] {
	var out []Document[T]
	for _, d := range c.docs {
		if d.Meta.Base().Featured {
			out = append(out, d)
		}
	}
	return out
}

func Label(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (c *Collection[T]) Title() string {
	if c.title != "" {
		return c.title
	}
	return Label(c.id)
}

func (c *Collection[T]) Description() string { return c.desc }

func (c *Collection[T]) IndexPage() PageSpec {
	return PageSpec{Title: c.Title(), Description: c.desc, Path: "/" + c.id}
}

func (c *Collection[T]) ItemPage(d Document[T]) PageSpec {
	m := d.Meta.Base()
	p := PageSpec{
		Title:       m.Title,
		Description: m.Summary,
		Path:        c.Href(d),
		Article: &ArticleSEO{
			Published: m.Date.ISO(),
			Modified:  modified(m),
			Draft:     m.Draft,
		},
	}
	if u := d.Image.shareURL(); u != "" {
		p.ShareImage = c.site.AbsURL(u)
	}
	return p
}

func (c *Collection[T]) Kicker(d Document[T]) string { return c.kicker(d) }

func (c *Collection[T]) Tags(d Document[T]) []Marker { return c.tags.Markers(d.Meta.Base().Tags) }

func Collect[T Metadata](spec Spec[T], baked []journal.Document, image func(string) (Image, error), site Site, drafts bool) (*Collection[T], error) {
	if spec.ID == "" {
		return nil, fmt.Errorf("collection: empty ID")
	}
	docs, err := load[T](baked)
	if err != nil {
		return nil, err
	}
	if err := checkTags(spec.ID, docs, spec.Classifier); err != nil {
		return nil, err
	}
	if err := images(image, spec.ID, docs); err != nil {
		return nil, err
	}
	docs = published(docs, drafts)
	sort := spec.Sort
	if sort == nil {
		sort = func(a, b Document[T]) int { return b.Meta.Base().Date.Compare(a.Meta.Base().Date.Time) }
	}
	slices.SortStableFunc(docs, sort)
	col := &Collection[T]{
		id: spec.ID, title: spec.Title, desc: spec.Description, site: site,
		docs: docs, bySlug: make(map[string]Document[T], len(docs)),
		kicker: spec.Kicker, tags: spec.Classifier,
	}
	for _, d := range docs {
		col.bySlug[d.Slug] = d
	}
	if col.kicker == nil {
		col.kicker = func(d Document[T]) string { return col.ShortMonthYear(d.Meta.Base().Date) }
	}
	return col, nil
}
