package serve

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirect(t *testing.T) {
	for _, c := range []struct{ method, url, apex, want string }{
		{"GET", "https://www.example.com/a?b=1", "example.com", "https://example.com/a?b=1"},
		{"GET", "https://www.elsewhere.com/a", "example.com", ""},
		{"GET", "/docs/?x=1", "", "/docs?x=1"},
		{"POST", "/docs/", "", ""},
		{"GET", "/", "", ""},
	} {
		got, _ := Redirect(httptest.NewRequest(c.method, c.url, nil), c.apex)
		if got != c.want {
			t.Errorf("%s %s: redirected to %q, want %q", c.method, c.url, got, c.want)
		}
	}
}

func TestCacheControl(t *testing.T) {
	hashes := map[string]string{"/a.css": "abc"}
	for url, want := range map[string]string{
		"/a.css?h=abc": "public, max-age=31536000, immutable",
		"/a.css?h=old": "no-cache",
		"/a.css":       "no-cache",
	} {
		if got := CacheControl(httptest.NewRequest("GET", url, nil), hashes); got != want {
			t.Errorf("%s: %q, want %q", url, got, want)
		}
	}
}

func TestPages(t *testing.T) {
	renders := 0
	page := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		renders++
		http.SetCookie(w, &http.Cookie{Name: "id", Value: "first-visitor"})
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(bytes.Repeat([]byte("page "), 200))
	})
	p := NewPages(1<<20, false)
	get := func(h http.Handler, coding, inm string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/a", nil)
		r.Header.Set("Accept-Encoding", coding)
		r.Header.Set("If-None-Match", inm)
		w := httptest.NewRecorder()
		p.Serve(w, r, h, nil)
		return w
	}
	first := get(page, "br", "")
	again := get(page, "br", "")
	if renders != 1 {
		t.Errorf("rendered %d times for one path and coding", renders)
	}
	if again.Header().Get("Set-Cookie") != "" {
		t.Error("a cached page replayed one visitor's cookie to the next")
	}
	if gz := get(page, "gzip", ""); renders != 2 || gz.Header().Get("ETag") == first.Header().Get("ETag") {
		t.Error("the coding is not part of the cache key and the ETag")
	}
	if w := get(page, "br", first.Header().Get("ETag")); w.Code != http.StatusNotModified {
		t.Errorf("a matching ETag got %d, want 304", w.Code)
	}
	missing := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	n := p.Len()
	get(missing, "br", "")
	if p.Len() != n {
		t.Error("a 404 was stored")
	}
}
