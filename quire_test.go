package quire

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
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
	"github.com/lahickey/quire/bake"
	"github.com/lahickey/quire/bundle"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/testkit"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base/convert"
	"github.com/lahickey/quire/pack/base/convert/markdown"
	"github.com/lahickey/quire/pack/base/convert/picture"
	"github.com/lahickey/quire/plugin"
)

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

func bakeSite(spec model.BakeSpec, id model.Identity, b Blueprint, opts ...Option) error {
	baker, err := bake.Start(spec, convert.Standard())
	if err != nil {
		return err
	}
	if err := Describe(id, b, opts...).Collect(baker); err != nil {
		return err
	}
	_, err = baker.Finish()
	return err
}

func serveOver(files fs.FS, id model.Identity, b Blueprint, opts ...Option) (*Server, error) {
	baked, err := bundle.Open(files)
	if err != nil {
		return nil, err
	}
	site, err := Describe(id, b, opts...).Build(baked)
	if err != nil {
		return nil, err
	}
	return NewServer(site)
}

func bare(id model.Identity, declare func(*Site) error, opts ...Option) (*Server, error) {
	site, err := Describe(id, BlueprintFunc(declare), opts...).Build(nil)
	if err != nil {
		return nil, err
	}
	return NewServer(site)
}

func served(t *testing.T, files fs.FS, id model.Identity, declare func(*Site) error, opts ...Option) (*Server, error) {
	t.Helper()
	content := t.TempDir()
	if err := os.CopyFS(content, files); err != nil {
		t.Fatal(err)
	}
	spec := model.BakeSpec{Content: content, Dir: t.TempDir()}
	if err := bakeSite(spec, id, BlueprintFunc(declare), opts...); err != nil {
		return nil, err
	}
	return serveOver(os.DirFS(spec.Bundle()), id, BlueprintFunc(declare), opts...)
}

func placed(t *testing.T, files fs.FS, name string) model.Picture {
	t.Helper()
	var im model.Picture
	if _, err := served(t, files, model.Identity{}, func(s *Site) (err error) {
		im, err = s.Picture(name)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return im
}

func rungFile(r journal.Record, w int) string {
	if i := slices.Index(r.Picture().Widths, w); i >= 0 {
		return r.Files[i].Name
	}
	return ""
}

func posts(s *Site) error {
	c, err := s.Collection(model.Basic{ID: "posts"})
	if err != nil {
		return err
	}
	s.AddPage(c.Page(), body("index"))
	s.AddDocumentPages(c, func(d model.Document[model.Meta]) (model.PageSpec, templ.Component) {
		return model.DocumentPage(c, d), body("item")
	})
	return nil
}

func nothing(*Site) error { return nil }

func TestShareImages(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string][]byte{
		"content/posts/pic.md":    []byte("---\ntitle: Pic\npicture: ridge.png\n---\nA ridge.\n"),
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
	site := model.Identity{Name: "Example", BaseURL: "https://example.com", Description: "A site."}
	spec := model.BakeSpec{Content: filepath.Join(dir, "content"), Dir: filepath.Join(dir, ".quire")}
	if err := bakeSite(spec, site, BlueprintFunc(posts)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".quire/.gitignore")); err != nil {
		t.Errorf("the bake left .quire without its .gitignore: %v", err)
	}
	files := os.DirFS(spec.Bundle())
	var col *model.Collection[model.Meta]
	if _, err := serveOver(files, site, BlueprintFunc(func(s *Site) (err error) {
		col, err = s.Collection(model.Basic{ID: "posts"})
		return err
	})); err != nil {
		t.Fatal(err)
	}
	pic, _ := col.Get("pic")
	plain, _ := col.Get("plain")
	if u, want := model.DocumentPage(col, pic).ShareImage, model.BundleURL(rungFile(testkit.ReadJournal(t, spec.Bundle())["posts/ridge.png"], 1096)); u != want {
		t.Errorf("a document with a picture shares %q, want its own 1096 rung %q", u, want)
	}
	if u := model.DocumentPage(col, plain).ShareImage; u != "" {
		t.Errorf("a document without a picture shares %q of its own, want the site's", u)
	}
}

func TestDraftsLoadOnlyWithTheFlag(t *testing.T) {
	fsys := fstest.MapFS{"posts/d.md": {Data: []byte("---\ntitle: D\ndate: 2026-01-01\ndraft: true\n---\n")}}
	for _, drafts := range []bool{false, true} {
		var col *model.Collection[model.Meta]
		if _, err := served(t, fsys, model.Identity{}, func(s *Site) (err error) {
			col, err = s.Collection(model.Basic{ID: "posts"})
			return err
		}, WithDrafts(drafts)); err != nil {
			t.Fatal(err)
		}
		if (col.Len() == 1) != drafts {
			t.Errorf("drafts %v: the collection holds %d documents", drafts, col.Len())
		}
	}
}

func TestArticleHead(t *testing.T) {
	v, err := bare(model.Identity{BaseURL: "https://example.com"}, nothing)
	if err != nil {
		t.Fatal(err)
	}
	head := func(a model.ArticleSEO) string {
		var b strings.Builder
		if err := v.site.Page(model.PageSpec{Title: "T", Path: "/posts/t", Article: &a}, body("")).Render(context.Background(), &b); err != nil {
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
	var renders atomic.Int32
	v, err := bare(model.Identity{}, func(s *Site) error {
		s.SetNotFound(templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			renders.Add(1)
			_, err := io.WriteString(w, strings.Repeat("gone ", 400))
			return err
		}))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 200 {
		r := httptest.NewRequest("GET", fmt.Sprintf("/junk-%d", i), nil)
		r.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()
		v.ServeHTTP(rec, r)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("junk-%d: status %d", i, rec.Code)
		}
		if rec.Header().Get("Content-Encoding") != "gzip" {
			t.Fatalf("junk-%d: served uncompressed", i)
		}
	}
	if n := v.pages.Len(); n != 0 {
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
	v, err := served(t, fsys, model.Identity{BaseURL: "https://example.com"}, func(s *Site) error {
		c, err := s.Collection(model.Spec[note]{ID: "writing", Markdown: "*.md"})
		if err != nil {
			return err
		}
		s.AddPage(c.Page(), body("index"))
		s.AddDocumentPages(c, func(d model.Document[note]) (model.PageSpec, templ.Component) {
			return model.DocumentPage(c, d), body("item")
		})
		s.AddPage(model.PageSpec{Path: "/contact"}, body("contact"))
		s.AddPage(model.PageSpec{Path: "/"}, body("home"))
		return nil
	}, WithDev(true))
	if err != nil {
		t.Fatal(err)
	}

	get := func(path string) string {
		rec := httptest.NewRecorder()
		v.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Body.String()
	}
	if g := get("/robots.txt"); !strings.Contains(g, "Sitemap: https://example.com/sitemap.xml") {
		t.Fatalf("robots: %s", g)
	}
	sm := get("/sitemap.xml")
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
}

// A frame of the site's own takes the place of any other, so it must still be handed
// what a header draws: the page's path and the controls plugins add.
func TestFrameSeesThePathAndControls(t *testing.T) {
	v, err := bare(model.Identity{}, func(s *Site) error {
		s.SetFrame(func(c model.Chrome, p model.PageSpec) templ.Component {
			return body(fmt.Sprintf("<header>%s %d</header>", p.Path, len(c.NavEnd)))
		})
		s.AddPage(model.PageSpec{Path: "/about"}, body("about"))
		s.AddPlugins(testPlugin{"basket", func(m *plugin.Mount) error {
			m.AddNavEnd(body("basket"))
			return nil
		}})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	v.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/about", nil))
	if !strings.Contains(rec.Body.String(), "<header>/about 1</header>") {
		t.Fatalf("the frame was not handed the path and the plugin's control: %s", rec.Body.String())
	}
}

func TestOnlyPathDependentRoutesAreCached(t *testing.T) {
	var live, posts atomic.Int32
	h, err := bare(model.Identity{BaseURL: "https://example.com"}, func(l *Site) error {
		l.AddRoute("GET", "/live", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, "<html>%d</html>", live.Add(1))
		})
		l.AddRoute("POST", "/act", func(w http.ResponseWriter, r *http.Request) {
			posts.Add(1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<html>posted</html>"))
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

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
	if h.pages.Len() != 0 {
		t.Errorf("%d entries in the page cache; neither route may be stored", h.pages.Len())
	}
}

func TestAliases(t *testing.T) {
	doc := func(front string) *fstest.MapFile {
		return &fstest.MapFile{Data: []byte("---\ntitle: T\ndate: 2026-01-01\n" + front + "---\n")}
	}
	v, err := served(t, fstest.MapFS{"posts/new.md": doc("aliases: [/old/name]\n")}, model.Identity{}, posts)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	v.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/old/name", nil))
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/posts/new" {
		t.Errorf("alias answered %d to %q", rec.Code, rec.Header().Get("Location"))
	}
	shadow := fstest.MapFS{"posts/a.md": doc(""), "posts/b.md": doc("aliases: [/posts/a]\n")}
	if _, err := served(t, shadow, model.Identity{}, posts); err == nil {
		t.Error("an alias shadowing a live document loaded")
	}
}

func TestFrontmatterFailsLoad(t *testing.T) {
	if _, err := load[note]([]model.BakedDocument{{Path: "x.md", Front: "title: A\nnope: 1"}}); err == nil {
		t.Error("an unknown frontmatter field loaded")
	}
	docs := []model.BakedDocument{{Path: "a/intro.md", Front: "title: A"}, {Path: "b/intro.md", Front: "title: B"}}
	if _, err := load[note](docs); err == nil || !strings.Contains(err.Error(), "intro") {
		t.Errorf("two documents with one slug loaded: %v", err)
	}
}

func (p testPlugin) Name() string { return p.name }

func (p testPlugin) Attach(m *plugin.Mount) error { return p.attach(m) }

func TestPluginScoping(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("x")) }
	use := func(ps ...plugin.Plugin) (*Server, error) {
		return bare(model.Identity{}, func(s *Site) error {
			s.AddPlugins(ps...)
			return nil
		})
	}
	var prefix string
	basket := testPlugin{"basket", func(m *plugin.Mount) error {
		prefix = m.URL("")
		m.Route("GET", "events", ok)
		return nil
	}}
	h, err := use(basket)
	if err != nil || prefix != "/_basket/" {
		t.Fatalf("prefix %q, %v", prefix, err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_basket/events", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("a plugin's endpoint answered %d", rec.Code)
	}
	if _, err := use(basket, basket); err == nil || !strings.Contains(err.Error(), "basket") {
		t.Errorf("two plugins of one name must fail the build, got %v", err)
	}
	if _, err := use(testPlugin{"has/slash", func(*plugin.Mount) error { return nil }}); err == nil {
		t.Error("a name that would break the prefix was accepted")
	}
	if _, err := use(testPlugin{"quire", func(*plugin.Mount) error { return nil }}); err == nil {
		t.Error("a site plugin took the name reserved for Quire's own")
	}

	css := fstest.MapFS{"ok.css": {Data: []byte("/* .line */ .basket-line { background: url(a.svg) } a.basket-x:hover {}")}, "bad.css": {Data: []byte(".basket-line, .line {}")}}
	for name, want := range map[string]bool{"ok.css": true, "bad.css": false} {
		_, err := use(testPlugin{"basket", func(m *plugin.Mount) error {
			m.AddStylesheets(css, name)
			return nil
		}})
		if (err == nil) != want {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestSrcSetIsPerSize(t *testing.T) {
	body := testkit.Photo(t, 2400, 1600, false)
	im := placed(t, fstest.MapFS{"r.png": {Data: body}}, "r.png")
	scale := model.DefaultScale()
	thumb, page := scale.SrcSet(im, model.SizeThumb), scale.SrcSet(im, model.SizePage)
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
}

func TestMissingBodyImageFailsLoad(t *testing.T) {
	fsys := fstest.MapFS{
		"posts/hello.md": {Data: []byte("---\ntitle: Hello\ndate: 2026-02-01\n---\n\n![](nope.png)\n")},
	}
	if _, err := served(t, fsys, model.Identity{}, posts); err == nil {
		t.Fatal("a missing picture in the body baked silently")
	}
}

func TestNarrowMasterCollapsesSizes(t *testing.T) {
	im := placed(t, fstest.MapFS{"small.png": {Data: testkit.Flat(t, 600, 400, false)}}, "small.png")
	scale := model.DefaultScale()
	if scale.URL(im, model.SizePage) != scale.URL(im, model.SizeColumn) {
		t.Fatalf("600px master serves two files: %s", scale.SrcSet(im, model.SizePage))
	}
	if n := strings.Count(scale.SrcSet(im, model.SizePage), " 600w"); n != 1 {
		t.Fatalf("width 600 listed %d times: %s", n, scale.SrcSet(im, model.SizePage))
	}
	if scale.Attrs(im, model.SizePage, false)["sizes"] == scale.Attrs(im, model.SizeColumn, false)["sizes"] {
		t.Error("a collapsed file must not collapse the sizes")
	}
	if !strings.Contains(scale.SrcSet(im, model.SizePage), scale.URL(im, model.SizePage)+" 600w") {
		t.Fatalf("src is not in srcset: %s vs %s", scale.URL(im, model.SizePage), scale.SrcSet(im, model.SizePage))
	}
}

func TestMarkdownOutsideACollection(t *testing.T) {
	files := fstest.MapFS{"home/sample.md": {Data: []byte("---\n---\n\n```go\nx := 1\n```\n")}}
	var got, missing error
	var html strings.Builder
	_, err := served(t, files, model.Identity{}, func(s *Site) error {
		c, err := s.Markdown("home/sample.md")
		if err != nil {
			return err
		}
		html.Reset()
		got = c.Render(t.Context(), &html)
		_, missing = s.Markdown("home/absent.md")
		return nil
	})
	if err != nil || got != nil || !strings.Contains(html.String(), `<span class="lx-number">1</span>`) {
		t.Fatalf("markdown was not baked and served: %v %v %s", err, got, html.String())
	}
	if missing == nil {
		t.Fatal("a missing file is not an error")
	}
}

func TestWidgets(t *testing.T) {
	files := fstest.MapFS{"home/a.md": {Data: []byte("---\n---\n\n```box Title\ninside\n```\n\n```go\nx := 1\n```\n\n```nothing\nplain\n```\n")}}
	box := func(info, body string) (string, error) {
		return "<aside>" + info + ":" + strings.TrimSpace(body) + "</aside>", nil
	}
	var html strings.Builder
	read := func(s *Site) error {
		c, err := s.Markdown("home/a.md")
		if err != nil {
			return err
		}
		html.Reset()
		return c.Render(t.Context(), &html)
	}
	with := func(name string) func(*Site) error {
		return func(s *Site) error {
			s.SetWidget(name, box)
			return read(s)
		}
	}
	if _, err := served(t, files, model.Identity{}, with("box")); err != nil {
		t.Fatal(err)
	}
	for what, want := range map[string]string{
		"a widget's fence, outside any collection": "<aside>Title:inside</aside>",
		"a fence in a language":                    "lx-number",
		"a fence nothing claims":                   "<pre><code>plain\n</code></pre>",
	} {
		if !strings.Contains(html.String(), want) {
			t.Errorf("%s: no %s in %s", what, want, html.String())
		}
	}
	if _, err := served(t, files, model.Identity{}, with("go")); err == nil || !strings.Contains(err.Error(), "language") {
		t.Errorf("a widget named for a language the bake highlights must fail the build, got %v", err)
	}
	late := func(s *Site) error {
		s.SetWidget("box", box)
		return nil
	}
	if _, err := served(t, files, model.Identity{}, func(s *Site) error { return cmp.Or(read(s), late(s)) }); err == nil {
		t.Error("a widget added after content was read was accepted")
	}
}

// Without a pack a site is a correct document and nothing else: no stylesheet, no script, no frame.
func TestASiteWithNoPackIsNaked(t *testing.T) {
	v, err := bare(model.Identity{Name: "Bare"}, func(s *Site) error {
		s.AddPage(model.PageSpec{Title: "About", Path: "/about"}, body("<p>about</p>"))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	v.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/about", nil))
	page := rec.Body.String()
	if !strings.Contains(page, "<title>About</title>") || !strings.Contains(page, "<body><p>about</p></body>") {
		t.Errorf("not a document round the body alone:\n%s", page)
	}
	for _, look := range []string{"stylesheet", "<script", "<header", "<main"} {
		if strings.Contains(page, look) {
			t.Errorf("a site with no pack has %q:\n%s", look, page)
		}
	}
}

func TestBrokenPageFailsAtBoot(t *testing.T) {
	_, err := bare(model.Identity{}, BlueprintFunc(func(s *Site) error {
		s.AddPage(model.PageSpec{Path: "/about"}, templ.ComponentFunc(func(context.Context, io.Writer) error { return fmt.Errorf("boom") }))
		return nil
	}))
	if err == nil {
		t.Fatal("a page that cannot render was served")
	}
}

func TestBakeIsWhatTheBlueprintAsked(t *testing.T) {
	dir := t.TempDir()
	post := "---\ntitle: A\ndate: 2026-02-01\npicture: hero.png\n---\n\n![inline](inline.png)\n\n```mermaid\nflowchart LR\n  a --> b\n```\n"
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
		if err := bakeSite(spec, model.Identity{}, BlueprintFunc(posts)); err != nil {
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
	if _, ok := recs["drafts/b.md"]; ok {
		t.Error("a document the blueprint did not ask for was baked")
	}
	doc, ok := recs["posts/a.md"].File()
	if !ok {
		t.Fatal("the document the blueprint asked for is not in the journal")
	}
	files := os.DirFS(spec.Bundle())
	v, err := serveOver(files, model.Identity{}, BlueprintFunc(posts))
	if err != nil {
		t.Fatalf("serving from the bake's bundle: %v", err)
	}
	rec := httptest.NewRecorder()
	v.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, model.BundleURL(doc), nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("a document's file is served at %s: %d", model.BundleURL(doc), rec.Code)
	}
}

func TestServingLinksNoBake(t *testing.T) {
	const q = "github.com/lahickey/quire/"
	heavy := []string{q + "bake", q + "pack/base/convert", q + "internal/fonts/measure", "github.com/yuin/goldmark", q + "libs/lexis"}
	for what, args := range map[string][]string{
		"quire":                       {"."},
		"app built with quire_nobake": {"-tags", "quire_nobake", "./app"},
	} {
		out, err := exec.Command("go", append([]string{"list", "-deps"}, args...)...).Output()
		if err != nil {
			t.Fatal(err)
		}
		for _, dep := range strings.Fields(string(out)) {
			if slices.ContainsFunc(heavy, func(pkg string) bool { return strings.HasPrefix(dep, pkg) }) {
				t.Errorf("%s links %s", what, dep)
			}
			if what == "quire" && dep == q+"app" {
				t.Error("quire imports its helpers")
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
		q + "/model":        func(p string) bool { return p == q+"/model/radix" },
		q + "/bundle":       func(p string) bool { return p == q+"/model" || p == q+"/internal/journal" },
		q + "/pack/base/ui": func(p string) bool { return p == q+"/model" },
		q + "/plugin":       func(p string) bool { return p == q+"/model" || p == q+"/internal/session" },
	}
	for line := range strings.Lines(string(out)) {
		f := strings.Fields(line)
		pkg := f[0]
		for _, imp := range f[1:] {
			if !strings.HasPrefix(imp, q) {
				continue
			}
			if imp == q && !strings.HasPrefix(pkg, q+"/examples/") && !strings.HasPrefix(pkg, q+"/cmd/") && !strings.HasPrefix(pkg, q+"/pack/") && pkg != q+"/site" && pkg != q+"/app" {
				t.Errorf("%s imports quire", pkg)
			}
			if strings.HasPrefix(pkg, q+"/pack/") && strings.HasPrefix(imp, q+"/internal/") {
				t.Errorf("%s imports %s, which a pack from outside could not", pkg, imp)
			}
			if strings.HasPrefix(pkg, q+"/libs/") && !strings.HasPrefix(imp, q+"/libs/") {
				t.Errorf("%s imports %s", pkg, imp)
			}
			if rule, ok := may[pkg]; ok && !rule(imp) {
				t.Errorf("%s imports %s", pkg, imp)
			}
		}
	}
}
