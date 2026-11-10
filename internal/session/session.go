package session

import (
	"context"
	"net/http"
	"strings"
	"uuid"
)

type ctxKey int

const (
	key        ctxKey = 0
	cookieName        = "qsid"
)

func ID(r *http.Request) string {
	id, _ := r.Context().Value(key).(string)
	return id
}

func Mint(w http.ResponseWriter, r *http.Request, secure bool) *http.Request {
	if c, err := r.Cookie(cookieName); err == nil {
		if _, err := uuid.Parse(c.Value); err == nil {
			return r.WithContext(context.WithValue(r.Context(), key, c.Value))
		}
	}
	if !strings.Contains(r.Header.Get("Accept"), "text/html") {
		return r
	}
	id := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: id, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure,
	})
	return r.WithContext(context.WithValue(r.Context(), key, id))
}
