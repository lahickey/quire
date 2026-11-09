package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func request(path, cookie string, navigation bool) *http.Request {
	r := httptest.NewRequest("GET", path, nil)
	if navigation {
		r.Header.Set("Accept", "text/html")
	}
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	}
	return r
}

func TestMint(t *testing.T) {
	w := httptest.NewRecorder()
	id := Session(Mint(w, request("/", "", true), true))
	cookies := w.Result().Cookies()
	if id == "" || len(cookies) != 1 || cookies[0].Value != id || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatalf("a navigation minted %q with %v", id, cookies)
	}
	w = httptest.NewRecorder()
	if got := Session(Mint(w, request("/style.css", id, false), true)); got != id || len(w.Result().Cookies()) != 0 {
		t.Errorf("an asset request with the cookie got %q and minted %v", got, w.Result().Cookies())
	}

	if got := Session(Mint(httptest.NewRecorder(), request("/style.css", "", false), true)); got != "" {
		t.Errorf("an asset request minted %q", got)
	}

	if got := Session(Mint(httptest.NewRecorder(), request("/style.css", "../../etc/passwd", false), true)); got != "" {
		t.Errorf("a forged cookie was taken as the id %q", got)
	}
}
