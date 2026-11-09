package compress

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
)

const minSize = 256

const PageQuality = 5

var Codings = []string{"br", "gzip"}

type Variants map[string][]byte

func compressible(ct string, n int) bool {
	if n < minSize {
		return false
	}
	ct, _, _ = strings.Cut(ct, ";")
	switch {
	case strings.HasPrefix(ct, "text/"), ct == "image/svg+xml",
		ct == "application/javascript", ct == "application/xml",
		ct == "application/json":
		return true
	}
	return false
}

func gzipBody(body []byte, ct string) []byte {
	if !compressible(ct, len(body)) {
		return nil
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	_, _ = zw.Write(body)
	_ = zw.Close()
	if buf.Len() >= len(body) {
		return nil
	}
	return buf.Bytes()
}

func brotliBytes(body []byte, quality int) []byte {
	var buf bytes.Buffer
	bw := brotli.NewWriterLevel(&buf, quality)
	_, _ = bw.Write(body)
	_ = bw.Close()
	if buf.Len() >= len(body) {
		return nil
	}
	return bytes.Clone(buf.Bytes())
}

func Encode(body []byte, ct, coding string) ([]byte, string) {
	if coding == "" || !compressible(ct, len(body)) {
		return body, ""
	}
	var out []byte
	switch coding {
	case "br":
		out = brotliBytes(body, PageQuality)
	case "gzip":
		out = gzipBody(body, ct)
	}
	if len(out) == 0 {
		return body, ""
	}
	return out, coding
}

func Prepare(body []byte, ct string, quality int) Variants {
	if !compressible(ct, len(body)) {
		return nil
	}
	v := Variants{}
	if b := brotliBytes(body, quality); b != nil {
		v["br"] = b
	}
	if g := gzipBody(body, ct); g != nil {
		v["gzip"] = g
	}
	if len(v) == 0 {
		return nil
	}
	return v
}

func Accepted(r *http.Request, offer ...string) string {
	q := map[string]float64{}
	for part := range strings.SplitSeq(r.Header.Get("Accept-Encoding"), ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if name = strings.ToLower(strings.TrimSpace(name)); name == "" {
			continue
		}
		q[name] = 1
		if _, v, ok := strings.Cut(params, "q="); ok {
			if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				q[name] = f
			}
		}
	}
	for _, c := range offer {
		v, ok := q[c]
		if !ok {
			v, ok = q["*"]
		}
		if ok && v > 0 {
			return c
		}
	}
	return ""
}

func Negotiate(w http.ResponseWriter, r *http.Request, body []byte, v Variants) []byte {
	if len(v) == 0 {
		return body
	}
	w.Header().Set("Vary", "Accept-Encoding")
	offer := make([]string, 0, len(Codings))
	for _, c := range Codings {
		if _, have := v[c]; have {
			offer = append(offer, c)
		}
	}
	if c := Accepted(r, offer...); c != "" {
		w.Header().Set("Content-Encoding", c)
		return v[c]
	}
	return body
}
