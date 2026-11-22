package sessions

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lahickey/quire/model"
)

func TestID(t *testing.T) {
	s := New()
	if err := s.Start(t.Context(), model.Env{BaseURL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	w, r := httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil)
	id := s.ID(w, r)
	cookies := w.Result().Cookies()
	if id == "" || len(cookies) != 1 || cookies[0].Value != id || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatalf("a first visit was given %q with %v", id, cookies)
	}
	if again := s.ID(w, r); again != id || len(w.Result().Cookies()) != 1 {
		t.Errorf("asked twice in one request, the visitor is %q then %q", id, again)
	}
	w, r = httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil)
	r.AddCookie(cookies[0])
	if got := s.ID(w, r); got != id || len(w.Result().Cookies()) != 0 {
		t.Errorf("a returning visitor is %q and was given %v", got, w.Result().Cookies())
	}
	r = httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "../../etc/passwd"})
	if got := s.Read(r); got != "" {
		t.Errorf("a forged cookie was taken as the id %q", got)
	}

	for what, s := range map[string]*Sessions{"dev": New(), "said so": New(Secure(false))} {
		env := model.Env{BaseURL: "https://example.com", Dev: what == "dev"}
		if err := s.Start(t.Context(), env); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		s.ID(w, httptest.NewRequest("GET", "/", nil))
		if w.Result().Cookies()[0].Secure {
			t.Errorf("%s: the cookie is kept to HTTPS", what)
		}
	}
}
