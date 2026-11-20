package markdown

import (
	"slices"
	"strings"
	"testing"

	"github.com/lahickey/quire/bake"
	"github.com/lahickey/quire/bundle"
)

func converted(name, raw string) (bundle.Document, error) {
	out, err := Converter{}.Convert(bake.Input{Name: name, Body: []byte(raw)})
	if err != nil {
		return bundle.Document{}, err
	}
	return bundle.DecodeDocument(out.Body)
}

func TestCallouts(t *testing.T) {
	for _, c := range []struct{ src, want, not string }{
		{"> [!Warning]\n> Mind the gap.", `<blockquote class="callout callout--warning"><p class="callout-title">Warning</p>` + "\n<p>Mind the gap.</p>", "[!"},
		{"> [!NOTE]\n>\n> Second paragraph.", `<p class="callout-title">Note</p>` + "\n<p>Second paragraph.</p>", "<p></p>"},
		{"> Just a quote.", "<blockquote>\n<p>Just a quote.</p>", "callout"},
		{"> [!NOTE]", "[!NOTE]", "callout"},
		{"> [!SHOUT]\n> Not a kind.", "[!SHOUT]", "callout"},
	} {
		html, _, _, err := render(c.src, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(html, c.want) || strings.Contains(html, c.not) {
			t.Errorf("%q:\n%s", c.src, html)
		}
	}
}

func TestBracketedSpans(t *testing.T) {
	for _, c := range []struct{ src, want, not string }{
		{"Drawn [◆]{.annotation .mark} here.", `Drawn <span class="annotation mark">◆</span> here.`, "{"},
		{"A [link](https://example.com){.x}.", `<a href="https://example.com">link</a>`, "<span"},
		{"Not [a span]{style=red}.", "[a span]{style=red}", "<span"},
		{"[<b>]{.x}", `<span class="x">&lt;b&gt;</span>`, "<b>"},
	} {
		html, _, _, err := render(c.src, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(html, c.want) || strings.Contains(html, c.not) {
			t.Errorf("%q:\n%s", c.src, html)
		}
	}
}

func TestLinksAndHeadingAnchors(t *testing.T) {
	for src, want := range map[string]string{
		"[a](./bake.md)":                `href="./bake"`,
		"[a](../posts/one.md#part)":     `href="../posts/one#part"`,
		"[a](https://example.com/x.md)": `href="https://example.com/x.md"`,
		"[a](/files/x.md)":              `href="/files/x.md"`,
		"[a](./notes.mdx)":              `href="./notes.mdx"`,
		"## Shipping it":                `<h2 id="shipping-it">Shipping it<a class="heading-anchor" href="#shipping-it"`,
	} {
		html, _, _, err := render(src, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(html, want) {
			t.Errorf("%q:\n%s", src, html)
		}
	}
}

func TestFencesAreHoles(t *testing.T) {
	box := bundle.Embed{Fence: "box", Info: "Title", Body: "inside\n"}
	for _, c := range []struct {
		name, src string
		want      []bundle.Embed
		not       string
	}{
		{"one", "Hello\n\n```box Title\ninside\n```\n\nAfter.\n", []bundle.Embed{box}, "```"},
		{"adjacent", "```box one\nbody\n```\n```box two\nbody\n```\n", []bundle.Embed{{Fence: "box", Info: "one", Body: "body\n"}, {Fence: "box", Info: "two", Body: "body\n"}}, "<p>"},
		{"in a list", "- item\n\n  ```box nested\n  x\n  ```\n", []bundle.Embed{{Fence: "box", Info: "nested", Body: "x\n"}}, "language-box"},
		{"shown, not run", "````md\n```box Title\ninside\n```\n````\n", nil, "\x00"},
	} {
		html, _, embeds, err := render(c.src, "", []string{"box"})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(embeds, c.want) || strings.Contains(html, c.not) {
			t.Errorf("%s: %+v\n%s", c.name, embeds, html)
		}
	}
}

func TestOnlyCodeGetsACopyButton(t *testing.T) {
	html, _, _, err := render("```go\nx := 1\n```\n\n```\nplain\n```\n\n```box\ny\n```\n", "", []string{"box"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(html, `class="code-copy"`) != 2 || strings.Count(html, "</div>") != 2 {
		t.Errorf("want a button round each code block and none round the widget:\n%s", html)
	}
}

func TestTables(t *testing.T) {
	html, _, _, err := render("| Key | Does |\n|-|-|\n| `a` | Annotate |\n", "", nil)
	if err != nil || !strings.Contains(html, "<th>Key</th>") || !strings.Contains(html, "<td><code>a</code></td>") {
		t.Errorf("a table with a one-dash separator did not render: %v\n%s", err, html)
	}
}

func TestFrontmatter(t *testing.T) {
	long := strings.Repeat("x", 200*1024)
	for _, c := range []struct{ name, raw, front, html string }{
		{"thematic break", "---\ntitle: HR\n---\nBefore.\n\n---\n\nAfter.\n", "title: HR", "<hr"},
		{"crlf", "---\r\ntitle: CRLF\r\n---\r\nBody line\r\nSecond\r\n", "title: CRLF", "Body line\nSecond"},
		{"long line", "---\ntitle: Long\n---\n\n![img](data:image/png;base64," + long + ")\n", "title: Long", long},
	} {
		d, err := converted(c.name+".md", c.raw)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if d.Front != c.front || !strings.Contains(d.HTML, c.html) || strings.Contains(d.HTML, "\r") {
			t.Errorf("%s: front %q html %.200s", c.name, d.Front, d.HTML)
		}
	}
}

func TestEmbedsAreHoles(t *testing.T) {
	d, err := converted("posts/a.md", "---\ntitle: A\n---\n![x](p.png)\n\nSee ![x](p.png).\n\n```mermaid\nflowchart LR\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []bundle.Embed{
		{Fence: "mermaid", Body: "flowchart LR\n"},
		{Picture: "posts/p.png", Alt: "x", Caption: "x", Figure: true},
		{Picture: "posts/p.png", Alt: "x", Caption: "x"},
	}
	if !slices.Equal(d.Embeds, want) {
		t.Fatalf("embeds %+v", d.Embeds)
	}
	if strings.Contains(d.HTML, "<img") || strings.Contains(d.HTML, "language-mermaid") {
		t.Fatalf("an embed was rendered rather than left for serving: %s", d.HTML)
	}
}

func TestPaletteDrawsRangesAndTheRest(t *testing.T) {
	html, _, _, err := render("```palette moss\nall\n```\n", "", nil)
	if err != nil || strings.Count(html, `data-palette="moss"`) != 2 || !strings.Contains(html, "var(--qr-accent-12)") || !strings.Contains(html, "var(--qr-error-text)") || strings.Contains(html, "--qr-error-1") {
		t.Errorf("a named palette, whole: %v\n%s", err, html)
	}
	if _, _, _, err := render("```palette\neverything\n```\n", "", nil); err == nil {
		t.Error("an unknown role drew")
	}
}
