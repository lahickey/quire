package serve

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/lahickey/quire/internal/journal"
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
		p.Cached(h.ServeHTTP)(w, r)
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

func TestAcceptEncoding(t *testing.T) {
	for header, want := range map[string]string{
		"":                  "",
		"gzip":              "gzip",
		"br":                "br",
		"gzip, deflate, br": "br",
		"gzip;q=0, br":      "br",
		"br;q=0, gzip":      "gzip",
		"br;q=0, gzip;q=0":  "",
		"*":                 "br",
		"identity":          "",
		"GZIP":              "gzip",
	} {
		r := httptest.NewRequest("GET", "/", nil)
		if header != "" {
			r.Header.Set("Accept-Encoding", header)
		}
		if got := accepted(r, codings...); got != want {
			t.Errorf("accepted(%q) = %q, want %q", header, got, want)
		}
	}
}

func TestEvictsByBytes(t *testing.T) {
	entry := func(n int) []byte { return make([]byte, n) }
	c := newCache(300, func(b []byte) int { return len(b) })
	c.put("a", entry(100))
	c.put("b", entry(100))
	c.put("c", entry(100))
	if c.used != 300 {
		t.Fatalf("used %d, want 300", c.used)
	}
	c.get("a")
	c.put("d", entry(100))
	if c.used > 300 {
		t.Errorf("used %d, over budget", c.used)
	}
	for key, want := range map[string]bool{"a": true, "b": false, "c": true, "d": true} {
		if _, got := c.get(key); got != want {
			t.Errorf("%q cached = %v, want %v", key, got, want)
		}
	}
	c.put("huge", entry(400))
	if _, ok := c.get("huge"); ok || c.used > 300 {
		t.Errorf("oversized entry admitted (used %d)", c.used)
	}
}

func TestStore(t *testing.T) {
	const name = "0123456789abcdef0123456789abcdef.webp"
	s := NewStore(0, 0, false, "")
	if err := s.Load(fstest.MapFS{
		name: {Data: []byte("webp")},
		journal.JournalName: {Data: journal.WriteJournal([]journal.Record{{
			Name: "a.jpg", Kind: journal.KindPicture,
			Files:   []journal.File{{Name: name, W: 64, Size: 4}},
			Picture: journal.Picture{Format: journal.Format{Ext: "webp", Type: "image/webp"}},
		}})},
	}); err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string]int{
		name:                                    http.StatusOK,
		"fedcba9876543210fedcba9876543210.webp": http.StatusNotFound,
		"../../etc/passwd":                      http.StatusNotFound,
	} {
		r := httptest.NewRequest("GET", "/_quire/img/x", nil)
		r.SetPathValue("file", file)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != want {
			t.Errorf("%s: %d, want %d", file, w.Code, want)
		}
	}
}
