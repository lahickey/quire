package markdown

import (
	"slices"
	"strings"
	"testing"

	"github.com/lahickey/quire/internal/journal"
)

func TestCallouts(t *testing.T) {
	for _, c := range []struct{ src, want, not string }{
		{"> [!Warning]\n> Mind the gap.", `<blockquote class="callout callout--warning"><p class="callout-title">Warning</p>` + "\n<p>Mind the gap.</p>", "[!"},
		{"> [!NOTE]\n>\n> Second paragraph.", `<p class="callout-title">Note</p>` + "\n<p>Second paragraph.</p>", "<p></p>"},
		{"> Just a quote.", "<blockquote>\n<p>Just a quote.</p>", "callout"},
		{"> [!NOTE]", "[!NOTE]", "callout"},
		{"> [!SHOUT]\n> Not a kind.", "[!SHOUT]", "callout"},
	} {
		html, _, _, err := Render(c.src, "", nil)
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
		html, _, _, err := Render(c.src, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(html, c.want) || strings.Contains(html, c.not) {
			t.Errorf("%q:\n%s", c.src, html)
		}
	}
}

func TestFencesAreHoles(t *testing.T) {
	box := journal.Embed{Fence: "box", Info: "Title", Body: "inside\n"}
	for _, c := range []struct {
		name, src string
		want      []journal.Embed
		not       string
	}{
		{"one", "Hello\n\n```box Title\ninside\n```\n\nAfter.\n", []journal.Embed{box}, "```"},
		{"adjacent", "```box one\nbody\n```\n```box two\nbody\n```\n", []journal.Embed{{Fence: "box", Info: "one", Body: "body\n"}, {Fence: "box", Info: "two", Body: "body\n"}}, "<p>"},
		{"in a list", "- item\n\n  ```box nested\n  x\n  ```\n", []journal.Embed{{Fence: "box", Info: "nested", Body: "x\n"}}, "language-box"},
		{"shown, not run", "````md\n```box Title\ninside\n```\n````\n", nil, "\x00"},
	} {
		html, _, embeds, err := Render(c.src, "", []string{"box"})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(embeds, c.want) || strings.Contains(html, c.not) {
			t.Errorf("%s: %+v\n%s", c.name, embeds, html)
		}
	}
}

func TestFrontmatter(t *testing.T) {
	long := strings.Repeat("x", 200*1024)
	for _, c := range []struct{ name, raw, front, html string }{
		{"thematic break", "---\ntitle: HR\n---\nBefore.\n\n---\n\nAfter.\n", "title: HR", "<hr"},
		{"crlf", "---\r\ntitle: CRLF\r\n---\r\nBody line\r\nSecond\r\n", "title: CRLF", "Body line\nSecond"},
		{"long line", "---\ntitle: Long\n---\n\n![img](data:image/png;base64," + long + ")\n", "title: Long", long},
	} {
		d, err := Convert(c.name+".md", []byte(c.raw), nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if d.Front != c.front || !strings.Contains(d.HTML, c.html) || strings.Contains(d.HTML, "\r") {
			t.Errorf("%s: front %q html %.200s", c.name, d.Front, d.HTML)
		}
	}
}

func TestEmbedsAreHoles(t *testing.T) {
	d, err := Convert("posts/a.md", []byte("---\ntitle: A\n---\n![x](p.png)\n\nSee ![x](p.png).\n\n```mermaid\nflowchart LR\n```\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []journal.Embed{
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
