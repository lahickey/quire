package bundle

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/lahickey/quire/internal/journal"
)

func TestServe(t *testing.T) {
	const name = "0123456789abcdef0123456789abcdef.webp"
	s := New(0, 0, false, "")
	s.add(loaded{
		records: map[string]journal.Record{"a.jpg": {Name: "a.jpg", Formats: []journal.Variants{
			{Ext: "webp", Type: "image/webp", Out: []journal.Variant{{Name: name, W: 64, Size: 4}}},
		}}},
		fsys: fstest.MapFS{name: {Data: []byte("webp")}},
	})
	for file, want := range map[string]int{
		name:                                    http.StatusOK,
		"fedcba9876543210fedcba9876543210.webp": http.StatusNotFound,
		"../../etc/passwd":                      http.StatusNotFound,
	} {
		r := httptest.NewRequest("GET", "/_quire/img/x", nil)
		r.SetPathValue("file", file)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != want {
			t.Errorf("%s: %d, want %d", file, w.Code, want)
		}
	}
}
