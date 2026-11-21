package base

import (
	"image"
	_ "image/png"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lahickey/quire"
	"github.com/lahickey/quire/bake"
	"github.com/lahickey/quire/bundle"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/testkit"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base/convert"
	"github.com/lahickey/quire/pack/base/ui"
)

func described(declare func(*quire.Site) error) *quire.Description {
	return quire.Describe(model.Identity{}, quire.BlueprintFunc(declare), quire.WithPacks(Pack{}))
}

func baked(t *testing.T, files fs.FS, declare func(*quire.Site) error) (string, *bundle.Bundle, error) {
	t.Helper()
	content := t.TempDir()
	if err := os.CopyFS(content, files); err != nil {
		t.Fatal(err)
	}
	spec := model.BakeSpec{Content: content, Dir: t.TempDir()}
	baker, err := bake.Start(spec, convert.Standard())
	if err != nil {
		return "", nil, err
	}
	if err := described(declare).Collect(baker); err != nil {
		return "", nil, err
	}
	b, err := baker.Finish()
	return spec.Bundle(), b, err
}

func bundleOf(t *testing.T, files fs.FS, declare func(*quire.Site) error) fs.FS {
	t.Helper()
	dir, _, err := baked(t, files, declare)
	if err != nil {
		t.Fatal(err)
	}
	return os.DirFS(dir)
}

func served(t *testing.T, files fs.FS, declare func(*quire.Site) error) (*quire.Site, *bundle.Bundle, error) {
	t.Helper()
	_, b, err := baked(t, files, declare)
	if err != nil {
		return nil, nil, err
	}
	site, err := described(declare).Build(b)
	return site, b, err
}

func posts(s *quire.Site) error {
	_, err := s.Collection(model.Basic{ID: "posts"})
	return err
}

func TestBodyMarkdownImages(t *testing.T) {
	body := testkit.Photo(t, 1200, 800, false)
	fsys := fstest.MapFS{
		"posts/hello.md":   {Data: []byte("---\ntitle: Hello\ndate: 2026-02-01\n---\n\nA ridge.\n\n![the ridge](../images/ridge.png \"Dawn\")\n\nSee ![it](../images/ridge.png) here.\n\n![The ridge at dusk](/images/ridge.png)\n\n![away](https://example.com/x.jpg)\n")},
		"images/ridge.png": {Data: body},
	}
	var col *model.Collection[model.Meta]
	site, _, err := served(t, fsys, func(s *quire.Site) (err error) {
		col, err = s.Collection(model.Basic{ID: "posts"})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	d, _ := col.Get("hello")
	if !strings.Contains(d.HTML, `<figure class="figure">`) {
		t.Fatalf("a picture alone in a paragraph is not a figure: %s", d.HTML)
	}
	if !strings.Contains(d.HTML, "<figcaption>Dawn</figcaption>") {
		t.Fatal("title did not become a caption")
	}
	if !strings.Contains(d.HTML, "<figcaption>The ridge at dusk</figcaption>") {
		t.Fatalf("an untitled figure did not caption itself with its text: %s", d.HTML)
	}
	if !strings.Contains(d.HTML, `loading="lazy"`) {
		t.Fatal("a picture in the body is not lazy")
	}
	if strings.Contains(d.HTML, `fetchpriority="high"`) {
		t.Fatal("a picture in the body took the hero's priority")
	}
	im, err := site.Picture("images/ridge.png")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d.HTML, model.DefaultScale().URL(im, model.SizePage)) {
		t.Fatal("figure src is not the hero variant")
	}
	if strings.Count(d.HTML, `<figure class="figure">`) != 2 {
		t.Fatalf("a picture inside a sentence became a figure: %s", d.HTML)
	}
	if strings.Contains(d.HTML, `src="images/ridge.png"`) {
		t.Fatalf("a picture inside a sentence still points at the master: %s", d.HTML)
	}
	if !strings.Contains(d.HTML, `src="https://example.com/x.jpg"`) {
		t.Fatalf("a remote image was rewritten: %s", d.HTML)
	}
}

func TestBodyVideo(t *testing.T) {
	clip := []byte("not decoded")
	page := func(body string) fstest.MapFS {
		return fstest.MapFS{
			"posts/hello.md":  {Data: []byte("---\ntitle: Hello\ndate: 2026-02-01\n---\n\n" + body + "\n")},
			"posts/clip.webm": {Data: clip},
			"posts/clip.png":  {Data: testkit.Photo(t, 600, 400, false)},
		}
	}
	var col *model.Collection[model.Meta]
	site, baked, err := served(t, page(`![a clip](clip.webm "Caption")`), func(s *quire.Site) (err error) {
		col, err = s.Collection(model.Basic{ID: "posts"})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	d, _ := col.Get("hello")
	name, _ := baked.File("posts/clip.webm")
	poster, _ := site.Picture("posts/clip.png")
	for _, want := range []string{`src="` + model.BundleURL(name) + `"`, `poster="` + model.DefaultScale().URL(poster, model.SizePage) + `"`, `width="600"`, "<figcaption>Caption</figcaption>"} {
		if !strings.Contains(d.HTML, want) {
			t.Errorf("no %s in %s", want, d.HTML)
		}
	}
	if got, err := baked.Body(name); err != nil || string(got) != string(clip) {
		t.Errorf("the bundle holds %q, %v", got, err)
	}

	if _, _, err := served(t, page("See ![a clip](clip.webm) here."), posts); err == nil {
		t.Error("a video inside a sentence baked")
	}
	noPoster := page("![a clip](clip.webm)")
	delete(noPoster, "posts/clip.png")
	if _, _, err := served(t, noPoster, posts); err == nil {
		t.Error("a video with no poster baked")
	}
}

func TestMermaid(t *testing.T) {
	post := "---\ntitle: A\ndate: 2026-02-01\n---\n\n```mermaid\nflowchart LR\n  a[client] --> b[(store)]\n```\n"
	var col *model.Collection[model.Meta]
	read := func(s *quire.Site) (err error) {
		col, err = s.Collection(model.Basic{ID: "posts"})
		return err
	}
	if _, _, err := served(t, fstest.MapFS{"posts/a.md": {Data: []byte(post)}}, read); err != nil {
		t.Fatal(err)
	}
	doc, _ := col.Get("a")
	if !strings.Contains(doc.HTML, `<figure class="diagram"><svg`) || !strings.Contains(doc.HTML, `class="pele`) {
		t.Fatalf("a mermaid fence was not inlined as its drawing:\n%s", doc.HTML)
	}

	out := filepath.Join(t.TempDir(), "bundle")
	if err := os.CopyFS(out, bundleOf(t, fstest.MapFS{"posts/a.md": {Data: []byte(post)}}, read)); err != nil {
		t.Fatal(err)
	}
	var kept []journal.Record
	for name, r := range testkit.ReadJournal(t, out) {
		if !strings.HasPrefix(name, "mermaid/") {
			kept = append(kept, r)
		}
	}
	if err := os.WriteFile(filepath.Join(out, journal.Name), journal.Write(kept), 0o644); err != nil {
		t.Fatal(err)
	}
	lacking, err := bundle.Open(os.DirFS(out))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := described(read).Build(lacking); err == nil || !strings.Contains(err.Error(), "not baked") {
		t.Fatalf("a document whose drawing the bundle lacks must fail the build, got %v", err)
	}
}

func TestPaletteDrawsRangesAndTheRest(t *testing.T) {
	html, err := palette("moss", "all\n")
	if err != nil || strings.Count(html, `data-palette="moss"`) != 2 || !strings.Contains(html, "var(--qr-accent-12)") || !strings.Contains(html, "var(--qr-error-text)") || strings.Contains(html, "--qr-error-1") {
		t.Errorf("a named palette, whole: %v\n%s", err, html)
	}
	if _, err := palette("", "everything\n"); err == nil {
		t.Error("an unknown role drew")
	}
}

func TestShareCard(t *testing.T) {
	site, _, err := served(t, fstest.MapFS{"posts/a.md": {Data: []byte("---\ntitle: A\n---\nText.\n")}}, posts)
	if err != nil {
		t.Fatal(err)
	}
	var head strings.Builder
	if err := site.Page(model.PageSpec{Title: "T"}, ui.NotFound()).Render(t.Context(), &head); err != nil {
		t.Fatal(err)
	}
	_, after, _ := strings.Cut(head.String(), `property="og:image" content="`)
	card, _, _ := strings.Cut(after, `"`)
	server, err := quire.NewServer(site)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, card, nil))
	cfg, format, err := image.DecodeConfig(rec.Body)
	if err != nil || format != "png" || cfg.Width != 1200 || cfg.Height != 630 {
		t.Errorf("the share card at %q: %s %d×%d %v, want a 1200×630 png", card, format, cfg.Width, cfg.Height, err)
	}

	bare, err := described(func(*quire.Site) error { return nil }).Build(nil)
	if err != nil || bare == nil {
		t.Errorf("a site with nothing baked has no card to ask for, and must still build: %v", err)
	}
}

func TestPagesSkipACollectionWithNothingIn(t *testing.T) {
	draft := fstest.MapFS{"posts/d.md": {Data: []byte("---\ntitle: D\ndate: 2026-01-01\ndraft: true\n---\n")}}
	site, _, err := served(t, draft, func(s *quire.Site) error {
		c, err := s.Collection(model.Basic{ID: "posts"})
		if err != nil {
			return err
		}
		Pages(s, c)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := quire.NewServer(site)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("a collection whose only document is a draft has a listing: %d", rec.Code)
	}
}
