package plugin

import (
	"context"
	"net/http"
	"strings"
	"uuid"
)

type ctxKey int

const sessionKey ctxKey = 0

const sessionCookie = "qsid"

func Session(r *http.Request) string {
	id, _ := r.Context().Value(sessionKey).(string)
	return id
}

func Mint(w http.ResponseWriter, r *http.Request, secure bool) *http.Request {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if _, err := uuid.Parse(c.Value); err == nil {
			return r.WithContext(context.WithValue(r.Context(), sessionKey, c.Value))
		}
	}
	if !strings.Contains(r.Header.Get("Accept"), "text/html") {
		return r
	}
	id := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: id, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure,
	})
	return r.WithContext(context.WithValue(r.Context(), sessionKey, id))
}
