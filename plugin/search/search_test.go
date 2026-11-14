package search

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/lahickey/quire/model"
)

func TestAddSplitsADocumentAtItsHeadings(t *testing.T) {
	c, err := model.Collect(model.Basic{ID: "docs"}, []model.Document[model.Meta]{{
		Slug: "look",
		Meta: model.Meta{Title: "The look", Summary: "Palette and fonts."},
		HTML: `<p>Intro &amp; more.</p><figure><svg><text>drawn</text></svg></figure><h2 id="palette">Palette</h2><pre><code><span>quire</span><span>.</span><span>Run</span></code></pre><h3 id="fonts">Fonts</h3><p>Inter.</p>`,
	}}, model.Site{}, false)
	if err != nil {
		t.Fatal(err)
	}
	p := New()
	p.Add(c)
	want := []section{
		{"The look", "", "/docs/look", "Palette and fonts. Intro & more."},
		{"The look", "Palette", "/docs/look#palette", "quire.Run"},
		{"The look", "Fonts", "/docs/look#fonts", "Inter."},
	}
	if len(p.sections) != len(want) {
		t.Fatalf("sections: %+v", p.sections)
	}
	for i, s := range p.sections {
		if s != want[i] {
			t.Errorf("section %d: %+v, want %+v", i, s, want[i])
		}
	}
}

func TestFind(t *testing.T) {
	long := strings.Repeat("découpé — ", 40)
	p := &Plugin{sections: []section{
		{"Images", "", "/images", "A master is a picture. The palette is mentioned here."},
		{"The look", "Palette", "/look#palette", "Each role is a scale."},
		{"Baking", "Controls", "/bake#controls", long + "needle " + long},
	}}
	if hits := p.find("PALETTE"); len(hits) != 2 || hits[0].URL != "/look#palette" || hits[1].Match != "palette" {
		t.Errorf("a heading does not outrank a body match: %+v", hits)
	}
	if hits := p.find("palette master"); len(hits) != 1 || hits[0].URL != "/images" {
		t.Errorf("every term must match: %+v", hits)
	}
	hits := p.find("needle")
	if len(hits) != 1 {
		t.Fatalf("needle: %+v", hits)
	}
	h := hits[0]
	if !utf8.ValidString(h.Before) || !utf8.ValidString(h.After) || strings.HasPrefix(h.Before, "écoupé") || len(h.Before) > lead || len(h.After) > tail {
		t.Errorf("snippet is not cut at word boundaries: %q | %q", h.Before, h.After)
	}
	for _, q := range []string{"", "   ", "\xff\xfe", "(", `\`} {
		if hits := p.find(q); len(hits) != 0 {
			t.Errorf("%q found %+v", q, hits)
		}
	}
}
