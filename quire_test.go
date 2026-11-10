//go:build !quire_nobake

package quire

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/bake/baketest"
	"github.com/lahickey/quire/internal/bake/markdown"
	"github.com/lahickey/quire/internal/bake/picture"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/testkit"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/plugin"
)

var withFallback = []string{"webp", "jpg"}

type note struct {
	model.Meta `yaml:",inline"`
}

type testPlugin struct {
	name   string
	attach func(*plugin.Mount) error
}

func body(s string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

func bakeFS(t *testing.T, masters fs.FS, codecs ...string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, masters); err != nil {
		t.Fatal(err)
	}
	// AVIF at its fastest: these tests are about bookkeeping, not bytes.
	spec := model.BakeSpec{Content: dir, Dir: t.TempDir(), Codecs: codecs, AVIFSpeed: 9}
	if err := baketest.BakeMasters(spec); err != nil {
		t.Fatal(err)
	}
	return spec.Bundle()
}

func baked(t *testing.T, masters fs.FS) Option {
	t.Helper()
	return WithBundle(os.DirFS(bakeFS(t, masters, withFallback...)))
}

func served(t *testing.T, files fs.FS, site model.Site, mount func(*Engine) error, opts ...Option) (*Engine, error) {
	t.Helper()
	content := t.TempDir()
	if err := os.CopyFS(content, files); err != nil {
		t.Fatal(err)
	}
	spec := model.BakeSpec{Content: content, Dir: t.TempDir(), Codecs: withFallback, AVIFSpeed: 9}
	if err := New(site, append(opts, WithBake(spec))...).bake(mount); err != nil {
		return nil, err
	}
	e := New(site, append(opts, WithBundle(os.DirFS(spec.Bundle())))...)
	return e, mount(e)
}

func mounted(t *testing.T, name string, src []byte, opts ...Option) (model.Image, *Engine) {
	t.Helper()
	dir := t.TempDir()
	_, file, err := picture.Ingest(dir, name, src, false)
	if err != nil {
		t.Fatal(err)
	}
	e := New(model.Site{}, append([]Option{baked(t, os.DirFS(dir))}, opts...)...)
	im, err := e.addImage(file)
	if err != nil {
		t.Fatal(err)
	}
	return im, e
}

func rungFile(r journal.Record, w int) string {
	for _, v := range r.Formats[len(r.Formats)-1].Out {
		if v.W == w {
			return v.Name
		}
	}
	return ""
}

func posts(e *Engine) error {
	_, err := e.AddCollection(model.Basic{ID: "posts"})
	return err
}

func TestBakeSharesExistingRungsAndOneCard(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string][]byte{
		"content/posts/pic.md":    []byte("---\ntitle: Pic\nimage: ridge.png\n---\nA ridge.\n"),
		"content/posts/plain.md":  []byte("---\ntitle: Plain\nsummary: No picture.\n---\nText.\n"),
		"content/posts/ridge.png": testkit.Photo(t, 1600, 900, false),
	} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	site := model.Site{Name: "Example", BaseURL: "https://example.com", Description: "A site."}
	spec := model.BakeSpec{Content: filepath.Join(dir, "content"), Dir: filepath.Join(dir, ".quire")}
	if err := New(site, WithBake(spec)).bake(posts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".quire/.gitignore")); err != nil {
		t.Errorf("the bake left .quire without its .gitignore: %v", err)
	}
	bundle := os.DirFS(spec.Bundle())
	e := New(site, WithBundle(bundle))
	col, err := e.AddCollection(model.Basic{ID: "posts"})
	if err != nil {
		t.Fatal(err)
	}
	pic, _ := col.Get("pic")
	plain, _ := col.Get("plain")
	if u, want := col.ItemPage(pic).ShareImage, e.site.AbsURL(model.ImageURL(rungFile(testkit.ReadJournal(t, spec.Bundle())["posts/ridge.png"], 1096))); u != want {
		t.Errorf("a document with an image shares %q, want its own 1096 rung %q", u, want)
	}
	if u := col.ItemPage(plain).ShareImage; u != "" {
		t.Errorf("a document without an image shares %q of its own, want the site card", u)
	}
	h := e.Handler()
	card := e.chrome.OGImageURL
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, strings.TrimPrefix(card, "https://example.com"), nil))
	cfg, format, err := image.DecodeConfig(rec.Body)
	if err != nil || format != "png" || cfg.Width != 1200 || cfg.Height != 630 {
		t.Errorf("site card %s: %s %d×%d %v, want a 1200×630 png", card, format, cfg.Width, cfg.Height, err)
	}
}

func TestYAMLCollectionsBake(t *testing.T) {
	fsys := fstest.MapFS{"team.yaml": {Data: []byte("- title: A\n- title: B\n")}}
	var col *model.Collection[model.Meta]
	_, err := served(t, fsys, model.Site{}, func(e *Engine) (err error) {
		col, err = e.AddCollection(model.Basic{ID: "team", YAML: "team.yaml", ListOnly: true})
		return err
	})
	if err != nil || col.Len() != 2 {
		t.Fatalf("a YAML collection served %v from the bundle: %v", col, err)
	}
}

func TestDraftsMountOnlyWithTheFlag(t *testing.T) {
	fsys := fstest.MapFS{"posts/d.md": {Data: []byte("---\ntitle: D\ndate: 2026-01-01\ndraft: true\n---\n")}}
	for _, drafts := range []bool{false, true} {
		e, err := served(t, fsys, model.Site{}, posts, WithDrafts(drafts))
		if err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		e.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts", nil))
		inNav := slices.ContainsFunc(e.navItems, func(it model.NavItem) bool { return it.Href == "/posts" })
		if (rec.Code == http.StatusOK) != drafts || inNav != drafts {
			t.Errorf("drafts %v: /posts %d, in nav %v", drafts, rec.Code, inNav)
		}
	}
}

func TestArticleHead(t *testing.T) {
	e := New(model.Site{BaseURL: "https://example.com"})
	e.Handler()
	head := func(a model.ArticleSEO) string {
		var b strings.Builder
		if err := e.Shell(model.PageSpec{Title: "T", Path: "/posts/t", Article: &a}).Render(context.Background(), &b); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	draft := head(model.ArticleSEO{Published: "2026-07-20", Draft: true})
	if !strings.Contains(draft, `name="robots" content="noindex"`) || strings.Contains(draft, `rel="canonical"`) {
		t.Error("a draft must be noindex without a canonical")
	}
	page := head(model.ArticleSEO{Published: "2026-07-20", Modified: "2026-08-19"})
	for _, want := range []string{`property="og:type" content="article"`, `article:published_time" content="2026-07-20"`, `article:modified_time" content="2026-08-19"`, `rel="canonical"`} {
		if !strings.Contains(page, want) {
			t.Errorf("article head lacks %s", want)
		}
	}
}

// A custom 404 is a fixed page like the default: rendered once at boot and never
// handed the request, so no URL can make it vary or fill the page cache.
func TestNotFoundIsPreparedOnce(t *testing.T) {
	e := New(model.Site{})
	var renders atomic.Int32
	e.SetNotFound(templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		renders.Add(1)
		_, err := io.WriteString(w, "<html>"+strings.Repeat("gone ", 400)+"</html>")
		return err
	}))
	e.mountDefaults()
	for i := range 200 {
		r := httptest.NewRequest("GET", fmt.Sprintf("/junk-%d", i), nil)
		r.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()
		e.Handler().ServeHTTP(rec, r)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("junk-%d: status %d", i, rec.Code)
		}
		if rec.Header().Get("Content-Encoding") != "gzip" {
			t.Fatalf("junk-%d: served uncompressed", i)
		}
	}
	if n := e.pages.Len(); n != 0 {
		t.Errorf("404 took %d cache entries; it is prepared, not cached", n)
	}
	if n := renders.Load(); n != 1 {
		t.Errorf("custom 404 rendered %d times; it is rendered once at boot", n)
	}
}

func TestRobotsAndSitemap(t *testing.T) {
	fsys := fstest.MapFS{
		"a.md": &fstest.MapFile{Data: []byte("---\ntitle: A\ndate: 2026-01-01\n---\nx\n")},
		"d.md": &fstest.MapFile{Data: []byte("---\ntitle: D\ndate: 2026-02-01\ndraft: true\n---\nx\n")},
	}
	e, err := served(t, fsys, model.Site{BaseURL: "https://example.com"}, func(e *Engine) error {
		_, err := e.AddCollection(model.Spec[note]{
			ID:       "writing",
			Markdown: "*.md",
			Index:    func(model.Shell, *model.Collection[note]) templ.Component { return body("index") },
			Item:     func(model.Shell, *model.Collection[note], model.Document[note]) templ.Component { return body("item") },
		})
		e.AddPage("/contact", body("contact"))
		e.AddPage("/", body("home"))
		return err
	}, WithMode(Dev))
	if err != nil {
		t.Fatal(err)
	}

	if g := e.robotsTxt(); !strings.Contains(g, "Sitemap: https://example.com/sitemap.xml") {
		t.Fatalf("robots: %s", g)
	}
	sm := e.sitemapXML()
	for _, loc := range []string{
		"https://example.com/",
		"https://example.com/contact",
		"https://example.com/writing",
		"https://example.com/writing/a",
	} {
		if !strings.Contains(sm, "<loc>"+loc+"</loc>") {
			t.Fatalf("missing %s\n%s", loc, sm)
		}
	}
	if strings.Contains(sm, "/writing/d") {
		t.Fatal("draft in sitemap")
	}

	rec := httptest.NewRecorder()
	e.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<urlset") {
		t.Fatalf("sitemap route %d %s", rec.Code, rec.Body.String())
	}
}

// A custom header replaces Quire's nav, so it must still be handed what the nav
// draws: the page's path and the controls plugins add.
func TestCustomHeaderSeesNavAndControls(t *testing.T) {
	e := New(model.Site{}, WithHeader(func(c model.Chrome, p model.PageSpec) templ.Component {
		return body(fmt.Sprintf("<header>%s %d</header>", p.Path, len(c.NavEnd)))
	}))
	if err := e.Use(testPlugin{"basket", func(m *plugin.Mount) error {
		m.AddNavEnd(body("basket"))
		return nil
	}}); err != nil {
		t.Fatal(err)
	}
	e.AddPage("/about", e.Shell(model.PageSpec{Path: "/about"}))
	rec := httptest.NewRecorder()
	e.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/about", nil))
	if !strings.Contains(rec.Body.String(), "<header>/about 1</header>") {
		t.Fatalf("header was not handed the path and the plugin's control: %s", rec.Body.String())
	}
}

func TestOnlyPathDependentRoutesAreCached(t *testing.T) {
	e := New(model.Site{BaseURL: "https://example.com"})
	var live, posts atomic.Int32
	e.AddRoute("GET", "/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "<html>%d</html>", live.Add(1))
	})
	e.AddRoute("POST", "/act", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html>posted</html>"))
	})
	h := e.Handler()

	get := func(path string) string {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Body.String()
	}
	if a, b := get("/live"), get("/live"); a == b {
		t.Errorf("AddRoute served the same body twice (%q); it is per request", a)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/act", nil))
	if rec.Code != http.StatusOK || posts.Load() != 1 {
		t.Fatalf("POST status %d, handled %d", rec.Code, posts.Load())
	}
	if e.pages.Len() != 0 {
		t.Errorf("%d entries in the page cache; neither route may be stored", e.pages.Len())
	}
}

func (p testPlugin) Name() string { return p.name }

func (p testPlugin) Attach(m *plugin.Mount) error { return p.attach(m) }

func TestPluginScoping(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("x")) }

	e := New(model.Site{BaseURL: "https://example.com"})
	var prefix string
	err := e.Use(testPlugin{"basket", func(m *plugin.Mount) error {
		prefix = m.URL("")
		m.Route("GET", "events", ok)
		return m.Page("GET", "", ok)
	}}, plugin.At("/shop", testPlugin{"tickets", func(m *plugin.Mount) error {
		return errors.Join(m.Page("GET", "", ok), m.Page("GET", "/checkout", ok))
	}}))
	if err != nil {
		t.Fatal(err)
	}
	if prefix != "/_basket/" {
		t.Fatalf("prefix %q", prefix)
	}
	h := e.Handler()
	for _, path := range []string{"/_basket/events", "/_basket", "/shop", "/shop/checkout"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status %d", path, rec.Code)
		}
	}

	clash := New(model.Site{})
	err = clash.Use(
		plugin.At("/checkout", testPlugin{"basket", func(m *plugin.Mount) error { return m.Page("GET", "", ok) }}),
		plugin.At("/checkout", testPlugin{"tickets", func(m *plugin.Mount) error { return m.Page("GET", "", ok) }}),
	)
	if err == nil || !strings.Contains(err.Error(), "basket") || !strings.Contains(err.Error(), "tickets") {
		t.Fatalf("a second claim on /checkout must name both plugins, got %v", err)
	}

	reserved := New(model.Site{})
	if err := reserved.Use(plugin.At("/_quire", testPlugin{"basket", func(m *plugin.Mount) error {
		return m.Page("GET", "style.css", ok)
	}})); err == nil {
		t.Error("a plugin mounted a public page inside the reserved namespace")
	}

	if err := New(model.Site{}).Use(testPlugin{"has/slash", func(*plugin.Mount) error { return nil }}); err == nil {
		t.Error("a name that would break the prefix was accepted")
	}
	if err := New(model.Site{}, WithMode(Dev)).Use(plugin.At("/q", testPlugin{"quire", func(*plugin.Mount) error { return nil }})); err == nil {
		t.Error("a site plugin took the name reserved for Quire's own")
	}

	css := fstest.MapFS{"ok.css": {Data: []byte("/* .line */ .basket-line { background: url(a.svg) } a.basket-x:hover {}")}, "bad.css": {Data: []byte(".basket-line, .line {}")}}
	for name, want := range map[string]bool{"ok.css": true, "bad.css": false} {
		err := New(model.Site{}).Use(testPlugin{"basket", func(m *plugin.Mount) error {
			m.AddStylesheets(css, name)
			return nil
		}})
		if (err == nil) != want {
			t.Errorf("%s: %v", name, err)
		}
	}

	taken := New(model.Site{})
	taken.AddPage("/basket", body("site"))
	if err := taken.Use(plugin.At("/basket", testPlugin{"basket", func(m *plugin.Mount) error {
		return m.Page("GET", "", ok)
	}})); err == nil || !strings.Contains(err.Error(), "site") {
		t.Fatalf("a plugin claiming a site route must error, got %v", err)
	}
}

func TestSrcSetIsPerSlot(t *testing.T) {
	body := testkit.Photo(t, 2400, 1600, false)
	fsys := fstest.MapFS{"r.png": {Data: body}}
	e := New(model.Site{}, baked(t, fsys))
	im, err := e.addImage("r.png", model.SlotPage)
	if err != nil {
		t.Fatal(err)
	}
	thumb, page := im.SrcSet(model.SlotThumb), im.SrcSet(model.SlotPage)
	if strings.Count(thumb, ",") != 2 {
		t.Errorf("thumb srcset is not three rungs: %s", thumb)
	}
	if strings.Contains(thumb, " 1704w") {
		t.Errorf("a 4rem thumb offers a page-width rung: %s", thumb)
	}
	if !strings.Contains(page, " 1704w") {
		t.Errorf("page srcset stops short: %s", page)
	}
	if strings.Contains(page, " 64w") {
		t.Errorf("page srcset carries a thumbnail rung: %s", page)
	}
	if im.SrcSet(model.SlotBleed) != "" {
		t.Error("an image offers a slot above its ceiling")
	}
}

func TestBodyMarkdownImages(t *testing.T) {
	body := testkit.Photo(t, 1200, 800, false)
	fsys := fstest.MapFS{
		"posts/hello.md":   {Data: []byte("---\ntitle: Hello\ndate: 2026-02-01\n---\n\nA ridge.\n\n![the ridge](../images/ridge.png \"Dawn\")\n\nSee ![it](../images/ridge.png) here.\n\n![The ridge at dusk](/images/ridge.png)\n\n![away](https://example.com/x.jpg)\n")},
		"images/ridge.png": {Data: body},
	}
	var col *model.Collection[model.Meta]
	e, err := served(t, fsys, model.Site{}, func(e *Engine) (err error) {
		col, err = e.AddCollection(model.Basic{ID: "posts"})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	d, _ := col.Get("hello")
	if !strings.Contains(d.HTML, `<figure class="figure">`) {
		t.Fatalf("standalone image is not a figure: %s", d.HTML)
	}
	if !strings.Contains(d.HTML, "<figcaption>Dawn</figcaption>") {
		t.Fatal("title did not become a caption")
	}
	if !strings.Contains(d.HTML, "<figcaption>The ridge at dusk</figcaption>") {
		t.Fatalf("an untitled figure did not caption itself with its text: %s", d.HTML)
	}
	if !strings.Contains(d.HTML, `loading="lazy"`) {
		t.Fatal("body image is not lazy")
	}
	if strings.Contains(d.HTML, `fetchpriority="high"`) {
		t.Fatal("body image stole hero priority")
	}
	im, ok := e.bundle.Mounted("images/ridge.png", model.SlotThumb)
	if !ok {
		t.Fatal("e.Image missed the body path")
	}
	if !strings.Contains(d.HTML, im.URL(model.SlotPage)) {
		t.Fatal("figure src is not the hero variant")
	}
	if strings.Count(d.HTML, `<figure class="figure">`) != 2 {
		t.Fatalf("inline image became a figure: %s", d.HTML)
	}
	if strings.Contains(d.HTML, `src="images/ridge.png"`) {
		t.Fatalf("inline image still points at the master: %s", d.HTML)
	}
	if !strings.Contains(d.HTML, `src="https://example.com/x.jpg"`) {
		t.Fatalf("remote image was rewritten: %s", d.HTML)
	}
}

func TestMissingBodyImageFailsLoad(t *testing.T) {
	fsys := fstest.MapFS{
		"posts/hello.md": {Data: []byte("---\ntitle: Hello\ndate: 2026-02-01\n---\n\n![](nope.png)\n")},
	}
	if _, err := served(t, fsys, model.Site{}, posts); err == nil {
		t.Fatal("a missing body image baked silently")
	}
}

func TestNarrowMasterCollapsesSlots(t *testing.T) {
	im, _ := mounted(t, "small.png", testkit.Flat(t, 600, 400, false))
	if im.URL(model.SlotPage) != im.URL(model.SlotColumn) {
		t.Fatalf("600px master serves two files: %s", im.SrcSet(model.SlotPage))
	}
	if n := strings.Count(im.SrcSet(model.SlotPage), " 600w"); n != 1 {
		t.Fatalf("width 600 listed %d times: %s", n, im.SrcSet(model.SlotPage))
	}
	if im.Sizes(model.SlotPage) == im.Sizes(model.SlotColumn) {
		t.Error("a collapsed file must not collapse the sizes")
	}
	if !strings.Contains(im.SrcSet(model.SlotPage), im.URL(model.SlotPage)+" 600w") {
		t.Fatalf("src is not in srcset: %s vs %s", im.URL(model.SlotPage), im.SrcSet(model.SlotPage))
	}
}

func TestDiagrams(t *testing.T) {
	content := t.TempDir()
	post := "---\ntitle: A\ndate: 2026-02-01\n---\n\n```mermaid\nflowchart LR\n  a[client] --> b[(store)]\n```\n"
	if err := os.MkdirAll(filepath.Join(content, "posts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "posts", "a.md"), []byte(post), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := model.BakeSpec{Content: content, Dir: t.TempDir()}
	out := spec.Bundle()
	if err := New(model.Site{}, WithBake(spec)).bake(posts); err != nil {
		t.Fatal(err)
	}
	load := func(bundle fs.FS) (string, error) {
		col, err := New(model.Site{}, WithBundle(bundle)).AddCollection(model.Basic{ID: "posts"})
		if err != nil {
			return "", err
		}
		doc, _ := col.Get("a")
		return doc.HTML, nil
	}

	baked, err := load(os.DirFS(out))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(baked, `<figure class="diagram"><svg`) || !strings.Contains(baked, `class="pele`) {
		t.Fatalf("diagram not inlined:\n%s", baked)
	}
	lacking := fstest.MapFS{}
	var kept []journal.Record
	for name, r := range testkit.ReadJournal(t, out) {
		if strings.HasPrefix(name, "mermaid/") {
			continue
		}
		kept = append(kept, r)
		for _, v := range r.Files() {
			body, err := os.ReadFile(filepath.Join(out, v.Name))
			if err != nil {
				t.Fatal(err)
			}
			lacking[v.Name] = &fstest.MapFile{Data: body}
		}
	}
	lacking[journal.JournalName] = &fstest.MapFile{Data: journal.WriteJournal(kept)}
	if _, err := load(lacking); err == nil || !strings.Contains(err.Error(), "not baked") {
		t.Fatalf("a document whose diagram the bundle lacks must fail the load, got %v", err)
	}
}

func TestBrokenPageFailsAtBoot(t *testing.T) {
	e := New(model.Site{})
	e.AddPage("/about", templ.ComponentFunc(func(context.Context, io.Writer) error { return fmt.Errorf("boom") }))
	defer func() {
		if recover() == nil {
			t.Fatal("a page that cannot render was served")
		}
	}()
	e.Handler()
}

func TestBakeIsWhatTheMountAsked(t *testing.T) {
	dir := t.TempDir()
	post := "---\ntitle: A\ndate: 2026-02-01\nimage: hero.png\n---\n\n![inline](inline.png)\n\n```mermaid\nflowchart LR\n  a --> b\n```\n"
	for name, body := range map[string][]byte{
		"posts/a.md":       []byte(post),
		"posts/hero.png":   testkit.Photo(t, 300, 200, false),
		"posts/inline.png": testkit.Photo(t, 300, 200, false),
		"posts/unused.png": testkit.Photo(t, 300, 200, false),
		"posts/notes.txt":  []byte("not a document"),
		"drafts/b.md":      []byte("---\ntitle: B\n---\n"),
	} {
		if err := os.MkdirAll(filepath.Join(dir, "content", filepath.Dir(name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "content", name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	spec := model.BakeSpec{Content: filepath.Join(dir, "content"), Dir: filepath.Join(dir, ".quire")}
	bakeCounting := func() (decodes, encodes, converts int64) {
		d0, e0, c0 := picture.Decodes(), picture.Encodes(), markdown.Converts()
		if err := New(model.Site{}, WithBake(spec)).bake(posts); err != nil {
			t.Fatal(err)
		}
		return picture.Decodes() - d0, picture.Encodes() - e0, markdown.Converts() - c0
	}
	if dec, _, conv := bakeCounting(); dec != 2 || conv != 1 {
		t.Errorf("a cold bake decoded %d times and converted %d, want once per master and document referred to", dec, conv)
	}
	if dec, enc, conv := bakeCounting(); dec != 0 || enc != 0 || conv != 0 {
		t.Errorf("a second bake decoded %d, encoded %d and converted %d", dec, enc, conv)
	}
	recs := testkit.ReadJournal(t, spec.Bundle())
	for _, name := range []string{"posts/hero.png", "posts/inline.png"} {
		if _, ok := recs[name]; !ok {
			t.Errorf("%s was referred to and not baked", name)
		}
	}
	if _, ok := recs["posts/unused.png"]; ok {
		t.Error("a master nothing refers to was baked")
	}
	if !slices.ContainsFunc(slices.Collect(maps.Keys(recs)), func(k string) bool { return strings.HasPrefix(k, "mermaid/") }) {
		t.Error("the mount's diagram was not drawn")
	}
	if _, ok := recs["drafts/b.md"]; ok {
		t.Error("a document the mount did not ask for was baked")
	}
	doc, ok := recs["posts/a.md"].File()
	if !ok {
		t.Fatal("the mount's document is not in the journal")
	}
	bundle := os.DirFS(spec.Bundle())
	e := New(model.Site{}, WithBundle(bundle))
	if err := posts(e); err != nil {
		t.Fatalf("serving from the bake's bundle: %v", err)
	}
	rec := httptest.NewRecorder()
	e.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, model.ImageURL(doc), nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("a document's file is served at %s: %d", model.ImageURL(doc), rec.Code)
	}
}

func TestServingBinaryLinksNoBake(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "-tags", "quire_nobake", ".").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, dep := range strings.Fields(string(out)) {
		for _, pkg := range []string{"github.com/lahickey/quire/internal/bake/", "github.com/yuin/goldmark", "github.com/alecthomas/chroma"} {
			if strings.HasPrefix(dep, pkg) {
				t.Errorf("a quire_nobake build links %s", dep)
			}
		}
	}
}

func TestLayers(t *testing.T) {
	const q = "github.com/lahickey/quire"
	out, err := exec.Command("go", "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", "./...").Output()
	if err != nil {
		t.Fatal(err)
	}
	may := map[string]func(string) bool{
		q + "/model":  func(p string) bool { return p == q+"/internal/journal" || p == q+"/internal/radix" },
		q + "/ui":     func(p string) bool { return p == q+"/model" },
		q + "/plugin": func(p string) bool { return p == q+"/model" || p == q+"/internal/session" },
	}
	for line := range strings.Lines(string(out)) {
		f := strings.Fields(line)
		pkg := f[0]
		for _, imp := range f[1:] {
			if !strings.HasPrefix(imp, q) {
				continue
			}
			if imp == q && !strings.HasPrefix(pkg, q+"/examples/") && !strings.HasPrefix(pkg, q+"/cmd/") && pkg != q+"/site" {
				t.Errorf("%s imports the engine", pkg)
			}
			if rule, ok := may[pkg]; ok && !rule(imp) {
				t.Errorf("%s imports %s", pkg, imp)
			}
		}
	}
}
