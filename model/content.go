// Package model is the data a Quire site is made of: the site, collections, documents, pictures and the look.
package model

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Marker struct {
	Text string
	Kind string
}

type Meta struct {
	Title    string   `yaml:"title"`
	Draft    bool     `yaml:"draft"`
	Featured bool     `yaml:"featured"`
	Date     Date     `yaml:"date"`
	Updated  Date     `yaml:"updated"`
	Summary  string   `yaml:"summary"`
	Picture  string   `yaml:"picture"`
	Tags     []string `yaml:"tag"`
	Aliases  []string `yaml:"aliases"`
	Links    []Link   `yaml:"links"`
	// Extra is whatever a site or a feature keeps in a document beyond the fields above.
	Extra map[string]any `yaml:"extra"`
}

type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
	ID    string `json:"id"`
}

type Widget = func(info, body string) (string, error)

type Document struct {
	Slug     string
	Path     string
	Meta     Meta
	HTML     string
	Words    int
	Headings []Heading

	Picture Picture
}

type Collection struct {
	id     string
	title  string
	desc   string
	docs   []Document
	bySlug map[string]Document
	kinds  map[string]string
	badges func(Document) []Marker
}

type Spec struct {
	ID          string
	Title       string
	Description string
	Markdown    string
	Tags        Tags
	Sort        func(a, b Document) int
	// Badges is what a document says of its own state, in the site's words.
	Badges func(Document) []Marker
}

func published(docs []Document, drafts bool) []Document {
	if drafts {
		return docs
	}
	n := 0
	for _, d := range docs {
		if d.Meta.Draft {
			continue
		}
		docs[n] = d
		n++
	}
	return docs[:n]
}

func modified(m Meta) string {
	if mod := m.LastMod(); !mod.Equal(m.Date.Time) {
		return mod.ISO()
	}
	return ""
}

// Text is one of a document's own fields, when it is a string.
func (m Meta) Text(key string) string {
	v, _ := m.Extra[key].(string)
	return v
}

func (m Meta) LastMod() Date {
	if !m.Updated.IsZero() {
		return m.Updated
	}
	return m.Date
}

func (c *Collection) Docs() []Document { return c.docs }

func (c *Collection) Len() int { return len(c.docs) }

func (c *Collection) Get(slug string) (Document, bool) {
	d, ok := c.bySlug[slug]
	return d, ok
}

func (c *Collection) Neighbours(d Document) (prev, next *Document) {
	i := slices.IndexFunc(c.docs, func(o Document) bool { return o.Slug == d.Slug })
	if i > 0 {
		prev = &c.docs[i-1]
	}
	if i >= 0 && i < len(c.docs)-1 {
		next = &c.docs[i+1]
	}
	return prev, next
}

func (c *Collection) ID() string { return c.id }

// Path is where the collection's pages go when it has any: its listing there, and each document beneath it.
func (c *Collection) Path() string { return "/" + c.id }

func (c *Collection) Href(d Document) string { return c.Path() + "/" + d.Slug }

func (c *Collection) Featured() []Document {
	var out []Document
	for _, d := range c.docs {
		if d.Meta.Featured {
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

func (c *Collection) Title() string {
	if c.title != "" {
		return c.title
	}
	return Label(c.id)
}

func (c *Collection) Description() string { return c.desc }

// Page is the head data of a collection's listing.
func (c *Collection) Page() PageSpec {
	return PageSpec{Title: c.Title(), Description: c.desc, Path: c.Path()}
}

// DocumentPage is the head data of a document's own page: an article, with its dates and the picture it shares.
func DocumentPage(c *Collection, d Document) PageSpec {
	m := d.Meta
	return PageSpec{
		Title:       m.Title,
		Description: m.Summary,
		Path:        c.Href(d),
		ShareImage:  d.Picture.shareURL(),
		Article:     &ArticleSEO{Published: m.Date.ISO(), Modified: modified(m), Draft: m.Draft},
	}
}

func (c *Collection) Badges(d Document) []Marker {
	if c.badges == nil {
		return nil
	}
	return c.badges(d)
}

// Extra reads every document's own fields into T, by slug. A field T does not have fails it.
func (c *Collection) Extra[T any]() (map[string]T, error) {
	out := make(map[string]T, len(c.docs))
	for _, d := range c.docs {
		raw, err := yaml.Marshal(d.Meta.Extra)
		if err != nil {
			return nil, fmt.Errorf("%s: extra: %w", d.Path, err)
		}
		var v T
		dec := yaml.NewDecoder(bytes.NewReader(raw))
		dec.KnownFields(true)
		if err := dec.Decode(&v); err != nil {
			return nil, fmt.Errorf("%s: extra: %w", d.Path, err)
		}
		out[d.Slug] = v
	}
	return out, nil
}

func (c *Collection) Tags(d Document) []Marker {
	tags := d.Meta.Tags
	out := make([]Marker, len(tags))
	for i, t := range tags {
		out[i] = Marker{Text: t, Kind: c.kinds[tagKey(t)]}
	}
	return out
}

func Collect(spec Spec, docs []Document, drafts bool) (*Collection, error) {
	if spec.ID == "" {
		return nil, fmt.Errorf("collection: empty ID")
	}
	kinds := spec.Tags.kinds()
	if spec.Tags != nil {
		var unknown []string
		for _, d := range docs {
			for _, t := range d.Meta.Tags {
				if _, ok := kinds[tagKey(t)]; !ok && tagKey(t) != "" {
					unknown = append(unknown, t)
				}
			}
		}
		if len(unknown) > 0 {
			slices.Sort(unknown)
			return nil, fmt.Errorf("%s: unclassified: %s", spec.ID, strings.Join(slices.Compact(unknown), ", "))
		}
	}
	docs = published(docs, drafts)
	sort := spec.Sort
	if sort == nil {
		sort = func(a, b Document) int { return b.Meta.Date.Compare(a.Meta.Date.Time) }
	}
	slices.SortStableFunc(docs, sort)
	col := &Collection{
		id: spec.ID, title: spec.Title, desc: spec.Description,
		docs: docs, bySlug: make(map[string]Document, len(docs)),
		kinds: kinds, badges: spec.Badges,
	}
	for _, d := range docs {
		col.bySlug[d.Slug] = d
	}
	return col, nil
}
