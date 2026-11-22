package site

import (
	"bytes"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/lahickey/quire/model"
)

var (
	tags       = regexp.MustCompile(`<[a-zA-Z][^<>"]*(?:"[^"]*"[^<>"]*)*>`)
	attributes = regexp.MustCompile(`([\w:.-]+)="([^"]*)"`)
	quoted     = regexp.MustCompile(`(&#39;|')(/[^/&'\s][^&'\s]*|/)`)
	cssURLs    = regexp.MustCompile(`url\(\s*["']?(/[^/"')\s][^"')\s]*)`)
)

// placedWriter places what a handler sends: the address it redirects to, and the addresses in markup it writes.
type placedWriter struct {
	http.ResponseWriter
	place  model.Place
	markup *bytes.Buffer
}

func placedURL(p model.Place, u string) string {
	if !strings.HasPrefix(u, "/") || strings.HasPrefix(u, "//") {
		return u
	}
	at := strings.IndexAny(u, "?#")
	if at < 0 {
		at = len(u)
	}
	return p.Out(u[:at]) + u[at:]
}

// placedHTML places every address of the site's own in markup: a link, a source, a form's action, and what a Datastar action fetches.
func placedHTML(p model.Place, markup []byte) []byte {
	return tags.ReplaceAllFunc(markup, func(tag []byte) []byte {
		return attributes.ReplaceAllFunc(tag, func(attr []byte) []byte {
			m := attributes.FindSubmatch(attr)
			name, value := string(m[1]), string(m[2])
			switch {
			case name == "href" || name == "src" || name == "action" || name == "formaction" || name == "poster":
				value = placedURL(p, value)
			case name == "srcset" || name == "imagesrcset":
				each := strings.Split(value, ", ")
				for i, u := range each {
					each[i] = placedURL(p, u)
				}
				value = strings.Join(each, ", ")
			case strings.HasPrefix(name, "data-"):
				value = quoted.ReplaceAllStringFunc(value, func(q string) string {
					m := quoted.FindStringSubmatch(q)
					return m[1] + placedURL(p, m[2])
				})
			}
			return []byte(name + `="` + value + `"`)
		})
	})
}

func placedCSS(p model.Place, sheet []byte) []byte {
	return cssURLs.ReplaceAllFunc(sheet, func(u []byte) []byte {
		m := cssURLs.FindSubmatch(u)
		return []byte(strings.TrimSuffix(string(u), string(m[1])) + placedURL(p, string(m[1])))
	})
}

func placedPattern(p model.Place, pattern string) string {
	method, path, _ := strings.Cut(pattern, " ")
	if path == "/{$}" && p.Root != "" {
		return method + " " + p.Root
	}
	return method + " " + p.Out(path)
}

// placedHandler hands a handler the request as the site wrote its addresses, and places what it answers.
// Markup that was placed when it was made, as a fixed page is, is sent as it is.
func placedHandler(p model.Place, h http.HandlerFunc, asIs bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := r.Clone(r.Context())
		in.URL.Path, in.URL.RawPath = p.In(r.URL.Path), ""
		if from, err := url.Parse(r.Referer()); err == nil && from.Path != "" {
			from.Path = p.In(from.Path)
			in.Header.Set("Referer", from.String())
		}
		out := &placedWriter{ResponseWriter: w, place: p}
		if !asIs {
			out.markup = &bytes.Buffer{}
		}
		h(out, in)
		if out.markup != nil && out.markup.Len() > 0 {
			_, _ = w.Write(placedHTML(p, out.markup.Bytes()))
		}
	}
}

func (w *placedWriter) WriteHeader(code int) {
	if to := w.Header().Get("Location"); to != "" {
		w.Header().Set("Location", placedURL(w.place, to))
	}
	w.ResponseWriter.WriteHeader(code)
}

// Write holds markup back until the handler is done, so its addresses can be placed. A stream, or anything else, goes straight out.
func (w *placedWriter) Write(b []byte) (int, error) {
	h := w.Header()
	if w.markup == nil || !strings.HasPrefix(h.Get("Content-Type"), "text/html") || h.Get("Content-Encoding") != "" {
		return w.ResponseWriter.Write(b)
	}
	h.Del("Content-Length")
	return w.markup.Write(b)
}

func (w *placedWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
