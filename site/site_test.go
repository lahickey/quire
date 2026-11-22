package site

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
)

type note struct {
	model.Meta `yaml:",inline"`
}

type testFeature struct {
	name    string
	declare func(*Site) error
}

func body(s string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

func bakeSite(spec model.BakeSpec, id model.Identity, f Feature, opts ...Option) error {
	baker, err := bake.Start(spec, convert.Standard())
	if err != nil {
		return err
	}
	if err := Describe(id, f, opts...).Collect(baker); err != nil {
		return err
	}
	_, err = baker.Finish()
	return err
}

func serveOver(files fs.FS, id model.Identity, f Feature, opts ...Option) (*Site, error) {
	baked, err := bundle.Open(files)
	if err != nil {
		return nil, err
	}
	return Describe(id, f, opts...).Build(baked)
}

func bare(id model.Identity, declare func(*Site) error, opts ...Option) (*Site, error) {
	return Describe(id, FeatureFunc(declare), opts...).Build(nil)
}

func get(s *Site, path string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	for _, r := range s.Routes() {
		mux.Handle(r.Pattern, r.Handler)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func served(t *testing.T, files fs.FS, id model.Identity, declare func(*Site) error, opts ...Option) (*Site, error) {
	t.Helper()
	content := t.TempDir()
	if err := os.CopyFS(content, files); err != nil {
		t.Fatal(err)
	}
	spec := model.BakeSpec{Content: content, Dir: t.TempDir()}
	if err := bakeSite(spec, id, FeatureFunc(declare), opts...); err != nil {
		return nil, err
	}
	return serveOver(os.DirFS(spec.Bundle()), id, FeatureFunc(declare), opts...)
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
	if err := bakeSite(spec, site, FeatureFunc(posts)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".quire/.gitignore")); err != nil {
		t.Errorf("the bake left .quire without its .gitignore: %v", err)
	}
	files := os.DirFS(spec.Bundle())
	var col *model.Collection[model.Meta]
	if _, err := serveOver(files, site, FeatureFunc(func(s *Site) (err error) {
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
		if err := v.Page(model.PageSpec{Title: "T", Path: "/posts/t", Article: &a}, body("")).Render(context.Background(), &b); err != nil {
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

func TestSitemapLeavesOutDrafts(t *testing.T) {
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
	})
	if err != nil {
		t.Fatal(err)
	}

	var locs []string
	for _, l := range v.Locs() {
		locs = append(locs, l.Path)
	}
	slices.Sort(locs)
	if want := []string{"/", "/contact", "/writing", "/writing/a"}; !slices.Equal(locs, want) {
		t.Fatalf("the sitemap lists %v, want %v", locs, want)
	}
}

// A frame of the site's own takes the place of any other, so it must still be handed
// what a header draws: the page's path and the controls features add.
func TestFrameSeesThePathAndControls(t *testing.T) {
	v, err := bare(model.Identity{}, func(s *Site) error {
		s.SetFrame(func(c model.Chrome, p model.PageSpec) templ.Component {
			return body(fmt.Sprintf("<header>%s %d</header>", p.Path, len(c.NavEnd)))
		})
		s.AddPage(model.PageSpec{Path: "/about"}, body("about"))
		return s.Include(testFeature{"basket", func(s *Site) error {
			s.AddNavEnd(body("basket"))
			return nil
		}})
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := get(v, "/about")
	if !strings.Contains(rec.Body.String(), "<header>/about 1</header>") {
		t.Fatalf("the frame was not handed the path and the feature's control: %s", rec.Body.String())
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
	rec := get(v, "/old/name")
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

func (f testFeature) Name() string { return f.name }

func (f testFeature) Declare(s *Site) error { return f.declare(s) }

func TestFeatureScoping(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("x")) }
	include := func(fs ...Feature) (*Site, error) {
		return bare(model.Identity{}, func(s *Site) error {
			for _, f := range fs {
				if err := s.Include(f); err != nil {
					return err
				}
			}
			return nil
		})
	}
	named := func(declare func(*Site) error) Feature { return testFeature{"basket", declare} }
	css := fstest.MapFS{"ok.css": {Data: []byte("/* .line */ .basket-line { background: url(a.svg) } a.basket-x:hover {}")}, "bad.css": {Data: []byte(".basket-line, .line {}")}}

	var prefix string
	basket := named(func(s *Site) error {
		prefix = s.URL("")
		s.Streams()
		s.AddStylesheets(css, "ok.css")
		s.AddRoute("GET", s.URL("events"), ok)
		s.AddPage(model.PageSpec{Path: "/basket"}, body("basket"))
		s.SetWidget("basket", func(info, body string) (string, error) { return "", nil })
		return nil
	})
	h, err := include(basket)
	if err != nil || prefix != "/_basket/" {
		t.Fatalf("prefix %q, %v", prefix, err)
	}
	for _, path := range []string{"/_basket/events", "/basket"} {
		if rec := get(h, path); rec.Code != http.StatusOK {
			t.Errorf("a named feature's %s answered %d", path, rec.Code)
		}
	}
	for what, f := range map[string]Feature{
		"a route in another's namespace": named(func(s *Site) error { s.AddRoute("GET", "/_other/x", ok); return nil }),
		"a class outside its namespace":  named(func(s *Site) error { s.AddStylesheets(css, "bad.css"); return nil }),
		"a widget outside its namespace": named(func(s *Site) error { s.SetWidget("box", nil); return nil }),
		"the site's look":                named(func(s *Site) error { s.SetFrame(nil); return nil }),
		"a route through a nameless feature it includes": named(func(s *Site) error {
			return s.Include(FeatureFunc(func(s *Site) error { s.AddRoute("GET", "/_other/x", ok); return nil }))
		}),
		"a name with a slash":           testFeature{"has/slash", nothing},
		"the name kept for Quire's own": testFeature{"quire", nothing},
	} {
		if _, err := include(f); err == nil {
			t.Errorf("a named feature was allowed %s", what)
		}
	}
	if _, err := include(basket, basket); err == nil || !strings.Contains(err.Error(), "twice") {
		t.Errorf("two features of one name must fail the build, got %v", err)
	}
	for what, declare := range map[string]func(*Site) error{
		"streams":                func(s *Site) error { s.Streams(); return nil },
		"a route under a prefix": func(s *Site) error { s.AddRoute("GET", "/_mine/x", ok); return nil },
	} {
		if _, err := bare(model.Identity{}, declare); err == nil {
			t.Errorf("a feature with no name was allowed %s", what)
		}
	}
	if _, err := Describe(model.Identity{}, basket).Build(nil); err == nil {
		t.Error("the site's own feature had a name")
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
	page := get(v, "/about").Body.String()
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
	_, err := bare(model.Identity{}, FeatureFunc(func(s *Site) error {
		s.AddPage(model.PageSpec{Path: "/about"}, templ.ComponentFunc(func(context.Context, io.Writer) error { return fmt.Errorf("boom") }))
		return nil
	}))
	if err == nil {
		t.Fatal("a page that cannot render was served")
	}
}

func TestBakeIsWhatTheSiteAsked(t *testing.T) {
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
		if err := bakeSite(spec, model.Identity{}, FeatureFunc(posts)); err != nil {
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
		t.Error("a document the site did not ask for was baked")
	}
	doc, ok := recs["posts/a.md"].File()
	if !ok {
		t.Fatal("the document the site asked for is not in the journal")
	}
	files := os.DirFS(spec.Bundle())
	v, err := serveOver(files, model.Identity{}, FeatureFunc(posts))
	if err != nil {
		t.Fatalf("serving from the bake's bundle: %v", err)
	}
	if rec := get(v, model.BundleURL(doc)); rec.Code != http.StatusNotFound {
		t.Errorf("a document's file is served at %s: %d", model.BundleURL(doc), rec.Code)
	}
}

func TestGroups(t *testing.T) {
	ok := func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("in")) }
	staff := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("key") != "let-me-in" {
				http.Error(w, "no", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	var at string
	basket := testFeature{"basket", func(s *Site) error {
		at = s.In(Admin).URL("orders")
		s.In(Admin).AddRoute("GET", "orders", ok)
		return nil
	}}
	built := func(declare func(*Site) error) (*Site, error) {
		return bare(model.Identity{}, func(s *Site) error { return cmp.Or(s.Include(basket), declare(s)) })
	}

	unset, err := built(nothing)
	if err != nil || at != "/admin/basket/orders" {
		t.Fatalf("a feature's address in a group is %q, %v", at, err)
	}
	if rec := get(unset, at+"?key=let-me-in"); rec.Code != http.StatusNotFound {
		t.Errorf("a group with no gate answered %d", rec.Code)
	}

	set, err := built(func(s *Site) error {
		s.SetGroup(Admin, staff)
		s.In(Admin).AddRoute("GET", "", ok)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]int{at: http.StatusForbidden, at + "?key=let-me-in": http.StatusOK, "/admin?key=let-me-in": http.StatusOK} {
		if rec := get(set, path); rec.Code != want {
			t.Errorf("%s answered %d, want %d", path, rec.Code, want)
		}
	}

	for what, declare := range map[string]func(*Site) error{
		"a route inside a group and outside its gate": func(s *Site) error {
			s.SetGroup(Admin, staff)
			s.AddRoute("GET", "/admin/open", ok)
			return nil
		},
		"a group with no gate at all": func(s *Site) error { s.SetGroup(Admin, nil); return nil },
		"a named feature setting a gate": func(s *Site) error {
			return s.Include(testFeature{"other", func(s *Site) error { s.SetGroup(Admin, staff); return nil }})
		},
	} {
		if _, err := built(declare); err == nil {
			t.Errorf("the site built with %s", what)
		}
	}
}
