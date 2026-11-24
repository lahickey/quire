package server

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/site"
)

func body(s string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

func bare(id model.Identity, declare func(*site.Site) error) (*Server, error) {
	s, err := site.Describe(id, site.FeatureFunc(declare)).Build(nil)
	if err != nil {
		return nil, err
	}
	return New(s)
}

// A custom 404 is a fixed page like the default: rendered once at boot and never
// handed the request, so no URL can make it vary or fill the page cache.
func TestNotFoundIsPreparedOnce(t *testing.T) {
	var renders atomic.Int32
	v, err := bare(model.Identity{}, func(s *site.Site) error {
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
	v, err := bare(model.Identity{BaseURL: "https://example.com"}, func(s *site.Site) error {
		s.AddPage(model.PageSpec{Path: "/contact"}, body("contact"))
		s.AddPage(model.PageSpec{Path: "/"}, body("home"))
		return nil
	})
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
	for _, loc := range []string{"https://example.com/", "https://example.com/contact"} {
		if !strings.Contains(sm, "<loc>"+loc+"</loc>") {
			t.Fatalf("missing %s\n%s", loc, sm)
		}
	}
}

func TestOnlyPathDependentRoutesAreCached(t *testing.T) {
	var live, shared, posts atomic.Int32
	h, err := bare(model.Identity{BaseURL: "https://example.com"}, func(l *site.Site) error {
		l.AddRoute("GET", "/live", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, "<html>%d</html>", live.Add(1))
		})
		l.AddSharedRoute("/shared/{name}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, "<html>%s %d</html>", r.PathValue("name"), shared.Add(1))
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

	if a, b := get("/shared/a"), get("/shared/a?q=1"); a != b {
		t.Errorf("AddSharedRoute drew one path twice: %q then %q", a, b)
	}
	if a, b := get("/shared/a"), get("/shared/b"); a == b {
		t.Errorf("AddSharedRoute served %q for two paths", a)
	}
}

type recorded struct {
	name string
	log  *[]string
	fail bool
}

func (r recorded) Start(_ context.Context, env model.Env) error {
	if r.fail {
		return errors.New(r.name + " would not start")
	}
	*r.log = append(*r.log, "start "+r.name+" "+env.Name)
	return nil
}

func (r recorded) Stop(context.Context) error {
	*r.log = append(*r.log, "stop "+r.name)
	return nil
}

func TestServicesStartInOrderAndStopInReverse(t *testing.T) {
	var log []string
	run := func(broken bool) (*Server, error) {
		log = nil
		return bare(model.Identity{Name: "S"}, func(s *site.Site) error {
			s.AddServices(recorded{"store", &log, false}, recorded{"bus", &log, broken}, recorded{"mail", &log, false})
			return nil
		})
	}
	v, err := run(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(log) != 0 {
		t.Fatalf("building the site started something: %v", log)
	}
	if err := cmp.Or(v.Start(t.Context()), v.Stop(t.Context())); err != nil {
		t.Fatal(err)
	}
	if want := []string{"start store S", "start bus S", "start mail S", "stop mail", "stop bus", "stop store"}; !slices.Equal(log, want) {
		t.Errorf("services ran as %v, want %v", log, want)
	}
	if v, err = run(true); err != nil {
		t.Fatal(err)
	}
	if err := v.Start(t.Context()); err == nil {
		t.Error("a service that would not start let the server start")
	}
	if want := []string{"start store S", "stop store"}; !slices.Equal(log, want) {
		t.Errorf("after a failed start the services ran as %v, want %v", log, want)
	}
}

func TestSeveralSites(t *testing.T) {
	var log []string
	shared := recorded{"store", &log, false}
	make := func(name string, at model.Place) *site.Site {
		s, err := site.Describe(model.Identity{Name: name, BaseURL: "https://example.com"}, site.FeatureFunc(func(s *site.Site) error {
			s.AddServices(shared, recorded{name, &log, false})
			s.AddPage(model.PageSpec{Path: "/"}, body("home of "+name))
			s.SetNotFound(body("nothing here in " + name))
			return nil
		}), site.WithPlace(at)).Build(nil)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	front, shop := make("front", model.Place{}), make("shop", model.Place{Root: "/shop", Engine: "/_quire/shop"})
	v, err := New(front, shop)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		"/":                 "home of front",
		"/shop":             "home of shop",
		"/missing":          "nothing here in front",
		"/shop/missing":     "nothing here in shop",
		"/robots.txt":       "Sitemap: https://example.com/shop/sitemap.xml",
		"/shop/sitemap.xml": "<loc>https://example.com/shop</loc>",
		"/_quire/health":    "ok",
	} {
		rec := httptest.NewRecorder()
		v.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("%s lacks %q: %s", path, want, rec.Body.String())
		}
	}
	if err := cmp.Or(v.Start(t.Context()), v.Stop(t.Context())); err != nil {
		t.Fatal(err)
	}
	if want := []string{"start store front", "start front front", "start shop shop", "stop shop", "stop front", "stop store"}; !slices.Equal(log, want) {
		t.Errorf("the services of two sites ran as %v, want %v", log, want)
	}
	if _, err := New(front, make("other", model.Place{})); err == nil {
		t.Error("two sites in one place were served together")
	}
}
