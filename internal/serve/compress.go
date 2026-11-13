package serve

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
)

const minSize = 256

const (
	pageQuality = 5
	bestQuality = brotli.BestCompression
)

var codings = []string{"br", "gzip"}

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

func compressed(body []byte, ct, coding string, quality int) ([]byte, string) {
	if !compressible(ct, len(body)) {
		return body, ""
	}
	var buf bytes.Buffer
	var zw io.WriteCloser
	switch coding {
	case "br":
		zw = brotli.NewWriterLevel(&buf, quality)
	case "gzip":
		zw, _ = gzip.NewWriterLevel(&buf, gzip.BestCompression)
	default:
		return body, ""
	}
	_, _ = zw.Write(body)
	_ = zw.Close()
	if buf.Len() >= len(body) {
		return body, ""
	}
	return buf.Bytes(), coding
}

func accepted(r *http.Request, offer ...string) string {
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
