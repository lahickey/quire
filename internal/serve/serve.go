package serve

import (
	"bytes"
	"cmp"
	"context"
	"encoding/xml"
	"maps"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/journal"
)

const htmlType = "text/html; charset=utf-8"

var managed = map[string]bool{
	"Content-Type": true, "Content-Length": true, "Content-Encoding": true,
	"Etag": true, "Vary": true, "Cache-Control": true, "Set-Cookie": true,
}

type page struct {
	body []byte
	hdr  http.Header
	enc  string
	etag string
	ct   string
	code int
}

type coded map[string]*page

type Prepared struct {
	path string
	c    templ.Component
	code int
	by   coded
}

type buffer struct {
	header http.Header
	code   int
	body   bytes.Buffer
	wrote  bool
}

type Pages struct {
	cache *cache[*page]
	dev   bool
}

type Loc struct {
	Path, LastMod string
}

func ifNoneMatch(inm, etag string) bool {
	if inm == "" {
		return false
	}
	for part := range strings.SplitSeq(inm, ",") {
		part = strings.TrimSpace(part)
		if part == "*" || part == etag {
			return true
		}
	}
	return false
}

func replayable(h http.Header) http.Header {
	var out http.Header
	for k, v := range h {
		if managed[k] {
			continue
		}
		if out == nil {
			out = http.Header{}
		}
		out[k] = slices.Clone(v)
	}
	return out
}

func (page *page) write(w http.ResponseWriter, r *http.Request) {
	maps.Copy(w.Header(), page.hdr)
	if page.ct != "" {
		w.Header().Set("Content-Type", page.ct)
	}
	w.Header().Set("Vary", "Accept-Encoding")
	if page.enc != "" {
		w.Header().Set("Content-Encoding", page.enc)
	}
	w.Header().Set("ETag", page.etag)
	if page.code/100 == 2 && ifNoneMatch(r.Header.Get("If-None-Match"), page.etag) &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead) {
		w.Header().Del("Content-Length")
		w.Header().Del("Content-Encoding")
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(page.body)))
	w.WriteHeader(page.code)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(page.body)
}

func newPage(code int, hdr http.Header, body []byte, coding string, quality int) *page {
	ct := hdr.Get("Content-Type")
	out, enc := compressed(body, ct, coding, quality)
	etag := `"` + journal.Hash(body)
	if enc != "" {
		etag += "-" + enc
	}
	return &page{body: out, hdr: replayable(hdr), enc: enc, etag: etag + `"`, ct: ct, code: code}
}

func encode(code int, ct string, body []byte, quality int) coded {
	hdr := http.Header{"Content-Type": {ct}}
	out := coded{"": newPage(code, hdr, body, "", quality)}
	for _, coding := range codings {
		out[coding] = newPage(code, hdr, body, coding, quality)
	}
	return out
}

func (c coded) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c[accepted(r, codings...)].write(w, r)
}

func Redirect(r *http.Request, apex string) (string, bool) {
	if apex != "" {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if bare, ok := strings.CutPrefix(strings.ToLower(host), "www."); ok && bare == strings.ToLower(apex) {
			return "https://" + bare + r.URL.RequestURI(), true
		}
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return "", false
	}
	p := r.URL.Path
	if p == "/" || !strings.HasSuffix(p, "/") {
		return "", false
	}
	p = cmp.Or(strings.TrimRight(p, "/"), "/")
	if q := r.URL.RawQuery; q != "" {
		return p + "?" + q, true
	}
	return p, true
}

func CacheControl(r *http.Request, hashes map[string]string) string {
	if h := r.URL.Query().Get("h"); h != "" && hashes[r.URL.Path] == h {
		return "public, max-age=31536000, immutable"
	}
	return "no-cache"
}

func Prepare(path string, c templ.Component, code int) *Prepared {
	return &Prepared{path: path, c: c, code: code}
}

func (p *Prepared) Build() {
	var buf bytes.Buffer
	if err := p.c.Render(context.Background(), &buf); err != nil {
		panic("quire: page " + p.path + ": " + err.Error())
	}
	p.by = encode(p.code, htmlType, buf.Bytes(), pageQuality)
}

func (p *Prepared) ServeHTTP(w http.ResponseWriter, r *http.Request) { p.by.ServeHTTP(w, r) }

func Static(contentType string, body []byte) http.Handler {
	return encode(http.StatusOK, contentType, body, bestQuality)
}

func (b *buffer) Header() http.Header { return b.header }

func (b *buffer) Write(p []byte) (int, error) {
	b.wrote = true
	return b.body.Write(p)
}

func (b *buffer) WriteHeader(code int) {
	if b.wrote {
		return
	}
	b.wrote = true
	b.code = code
}

func NewPages(max int, dev bool) *Pages {
	return &Pages{cache: newCache(max, func(p *page) int { return len(p.body) }), dev: dev}
}

func (p *Pages) Len() int { return p.cache.len() }

func (p *Pages) Cached(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coding := accepted(r, codings...)
		k := r.URL.Path + "\x00" + coding
		if page, ok := p.cache.get(k); ok {
			page.write(w, r)
			return
		}
		buf := &buffer{header: w.Header(), code: http.StatusOK}
		h(buf, r)
		if buf.code != http.StatusOK {
			w.WriteHeader(buf.code)
			_, _ = w.Write(buf.body.Bytes())
			return
		}
		page := newPage(buf.code, buf.header, buf.body.Bytes(), coding, pageQuality)
		if !p.dev {
			p.cache.put(k, page)
		}
		page.write(w, r)
	}
}

func Sitemap(abs func(string) string, locs []Loc) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	seen := map[string]bool{}
	for _, loc := range locs {
		if seen[loc.Path] {
			continue
		}
		seen[loc.Path] = true
		b.WriteString("  <url><loc>")
		_ = xml.EscapeText(&b, []byte(abs(loc.Path)))
		b.WriteString("</loc>")
		if loc.LastMod != "" {
			b.WriteString("<lastmod>")
			_ = xml.EscapeText(&b, []byte(loc.LastMod))
			b.WriteString("</lastmod>")
		}
		b.WriteString("</url>\n")
	}
	b.WriteString("</urlset>\n")
	return b.String()
}

func Robots(sitemap string) string {
	return "User-agent: *\nAllow: /\n\nSitemap: " + sitemap + "\n"
}

func Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}
