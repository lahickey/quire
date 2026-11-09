package compress

import (
	"net/http/httptest"
	"testing"
)

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
		if got := Accepted(r, Codings...); got != want {
			t.Errorf("Accepted(%q) = %q, want %q", header, got, want)
		}
	}
}
