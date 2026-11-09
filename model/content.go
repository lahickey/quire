// Package model is the data a Quire site is made of: the site, collections, documents, images and the look.
package model

import (
	"cmp"
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
	IsDraft() bool
	IsFeatured() bool
	DocDate() Date
	LastMod() Date
	DocTitle() string
	DocSummary() string
	DocTags() []string
	DocImage() string
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
	items  bool
	site   Site
	kicker func(Document[T]) string
	tags   *Classifier
}

type Spec[T Metadata] struct {
	ID          string
	Title       string
	Description string
	Markdown    string
	YAML        string
	Widgets     map[string]Widget
	Classifier  *Classifier
	Cmp         func(a, b Document[T]) int
	Kicker      func(Document[T]) string
	Index       func(Shell, *Collection[T]) templ.Component
	Item        func(Shell, *Collection[T], Document[T]) templ.Component
	ListOnly    bool

	ImageMax Slot
}

type Basic = Spec[Meta]

type Resolver interface {
	Documents(pattern string, widgets map[string]Widget) ([]journal.Document, error)
	Data(name string) ([]byte, error)
	Image(name string, max Slot) (Image, error)
}

func published[T Metadata](docs []Document[T], drafts bool) []Document[T] {
	if drafts {
		return docs
	}
	n := 0
	for _, d := range docs {
		if d.Meta.IsDraft() {
			continue
		}
		docs[n] = d
		n++
	}
	return docs[:n]
}

func sortByDateDesc[T Metadata](docs []Document[T]) {
	slices.SortStableFunc(docs, func(a, b Document[T]) int {
		return b.Meta.DocDate().Compare(a.Meta.DocDate().Time)
	})
}

func newCollection[T Metadata](id string, docs []Document[T], items bool) *Collection[T] {
	by := make(map[string]Document[T], len(docs))
	for _, d := range docs {
		by[d.Slug] = d
	}
	return &Collection[T]{id: id, docs: docs, bySlug: by, items: items}
}

func checkTags[T Metadata](id string, docs []Document[T], c *Classifier) error {
	if c == nil {
		return nil
	}
	var tags []string
	for _, d := range docs {
		tags = append(tags, d.Meta.DocTags()...)
	}
	if err := c.check(tags...); err != nil {
		return fmt.Errorf("%s: %w", id, err)
	}
	return nil
}

func modified[T Metadata](m T) string {
	if mod := m.LastMod(); !mod.Equal(m.DocDate().Time) {
		return mod.ISO()
	}
	return ""
}

func (spec Spec[T]) imageMax() Slot {
	if spec.ImageMax >= SlotThumb {
		return spec.ImageMax
	}
	if spec.ListOnly {
		return SlotThumb
	}
	return SlotDefault
}

func images[T Metadata](r Resolver, id string, docs []Document[T], max Slot) error {
	seen := map[string]Image{}
	for i, d := range docs {
		name := d.Meta.DocImage()
		if name == "" {
			continue
		}
		name = journal.Resolve(d.Path, name)
		im, ok := seen[name]
		if !ok {
			var err error
			if im, err = r.Image(name, max); err != nil {
				return fmt.Errorf("%s/%s: %w", id, d.Slug, err)
			}
			seen[name] = im
		}
		docs[i].Image = im
	}
	return nil
}

func (m Meta) IsDraft() bool { return m.Draft }

func (m Meta) IsFeatured() bool { return m.Featured }

func (m Meta) Badges() []Marker { return nil }

func (m Meta) DocDate() Date { return m.Date }

func (m Meta) DocTitle() string { return m.Title }

func (m Meta) DocSummary() string { return m.Summary }

func (m Meta) DocTags() []string { return m.Tags }

func (m Meta) DocImage() string { return m.Image }

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

func (c *Collection[T]) Href(d Document[T]) string {
	if !c.items {
		return "/" + c.id
	}
	return "/" + c.id + "/" + d.Slug
}

func (c *Collection[T]) LongDate(d Date) string { return d.Formatted(c.site.Dates.Long) }

func (c *Collection[T]) ShortDate(d Date) string { return d.Formatted(c.site.Dates.Short) }

func (c *Collection[T]) ShortMonthYear(d Date) string {
	return d.Formatted(c.site.Dates.ShortMonthYear)
}

func (c *Collection[T]) Featured() []Document[T] {
	var out []Document[T]
	for _, d := range c.docs {
		if d.Meta.IsFeatured() {
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
	p := PageSpec{
		Title:       d.Meta.DocTitle(),
		Description: d.Meta.DocSummary(),
		Path:        "/" + c.id + "/" + d.Slug,
		Article: &ArticleSEO{
			Published: d.Meta.DocDate().ISO(),
			Modified:  modified(d.Meta),
			Draft:     d.Meta.IsDraft(),
		},
	}
	if u := d.Image.shareURL(); u != "" {
		p.ShareImage = c.site.AbsURL(u)
	}
	return p
}

func (c *Collection[T]) Kicker(d Document[T]) string { return c.kicker(d) }

func (c *Collection[T]) Tags(d Document[T]) []Marker { return c.tags.Markers(d.Meta.DocTags()) }

func Collect[T Metadata](spec Spec[T], r Resolver, site Site, drafts bool) (*Collection[T], error) {
	if spec.ID == "" {
		return nil, fmt.Errorf("collection: empty ID")
	}
	if spec.Markdown != "" && spec.YAML != "" {
		return nil, fmt.Errorf("%s: set Markdown or YAML, not both", spec.ID)
	}
	var docs []Document[T]
	var err error
	if spec.YAML != "" {
		docs, err = loadSeq[T](r, spec.YAML)
	} else {
		docs, err = load[T](r, cmp.Or(spec.Markdown, spec.ID+"/*.md"), spec.Widgets)
	}
	if err != nil {
		return nil, err
	}
	if err := checkTags(spec.ID, docs, spec.Classifier); err != nil {
		return nil, err
	}
	if err := images(r, spec.ID, docs, spec.imageMax()); err != nil {
		return nil, err
	}
	docs = published(docs, drafts)
	if spec.Cmp != nil {
		slices.SortStableFunc(docs, spec.Cmp)
	} else {
		sortByDateDesc(docs)
	}
	col := newCollection(spec.ID, docs, !spec.ListOnly)
	col.title, col.desc, col.site = spec.Title, spec.Description, site
	col.kicker, col.tags = spec.Kicker, spec.Classifier
	if col.kicker == nil {
		col.kicker = func(d Document[T]) string { return col.ShortMonthYear(d.Meta.DocDate()) }
	}
	return col, nil
}
