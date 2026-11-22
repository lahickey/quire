// Package sessions is who a visitor is, by a cookie: nothing is known of them but that they came before.
package sessions

import (
	"cmp"
	"context"
	"net/http"
	"strings"
	"uuid"

	"github.com/lahickey/quire/model"
)

const cookieName = "qsid"

type Sessions struct {
	secure bool
	chosen bool
	path   string
}

type Option func(*Sessions)

// Secure says whether the cookie is kept to HTTPS. Left unsaid, it is on a site whose address is https that is not in dev.
func Secure(on bool) Option {
	return func(s *Sessions) { s.secure, s.chosen = on, true }
}

func New(opts ...Option) *Sessions {
	s := &Sessions{}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Sessions) Start(_ context.Context, env model.Env) error {
	s.path = env.Root
	if !s.chosen {
		s.secure = !env.Dev && strings.HasPrefix(env.BaseURL, "https://")
	}
	return nil
}

func (s *Sessions) Stop(context.Context) error { return nil }

// Read is the visitor's id, or nothing if they have none yet.
func (s *Sessions) Read(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	if _, err := uuid.Parse(c.Value); err != nil {
		return ""
	}
	return c.Value
}

// ID is the visitor's id, given to them now if they had none. Ask before the response is begun, since it may set a cookie.
func (s *Sessions) ID(w http.ResponseWriter, r *http.Request) string {
	if id := s.Read(r); id != "" {
		return id
	}
	c := &http.Cookie{
		Name: cookieName, Value: uuid.New().String(), Path: cmp.Or(s.path, "/"),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secure,
	}
	http.SetCookie(w, c)
	r.Header.Del("Cookie")
	r.AddCookie(c)
	return c.Value
}
