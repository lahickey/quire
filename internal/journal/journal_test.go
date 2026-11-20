package journal

import (
	"strings"
	"testing"
)

func TestReadJournalRefusesDamage(t *testing.T) {
	good := string(Write(nil)) +
		`{"name":"a.jpg","kind":"picture","key":"k","prep":{"h":"h","w":800,"y":600},"formats":[]}` + "\n"
	if recs, err := read(strings.NewReader(good)); err != nil || len(recs) != 1 {
		t.Fatalf("a whole journal read %d records: %v", len(recs), err)
	}
	for what, body := range map[string]string{
		"torn tail":       good + `{"name":"b.jpg","ke`,
		"no header":       `{"name":"a.jpg","key":"k"}` + "\n",
		"foreign version": `{"journal":"quire-journal","version":99}` + "\n" + strings.SplitN(good, "\n", 2)[1],
		"empty":           "",
	} {
		if _, err := read(strings.NewReader(body)); err == nil {
			t.Errorf("%s: read without an error", what)
		}
	}
}
