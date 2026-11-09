package journal

import (
	"slices"
	"strings"
	"testing"
)

func TestReadJournalRefusesDamage(t *testing.T) {
	good := string(WriteJournal(nil)) +
		`{"name":"a.jpg","key":"k","prep":{"h":"h","w":800,"y":600},"formats":[]}` + "\n"
	if recs, err := readJournal(strings.NewReader(good)); err != nil || len(recs) != 1 {
		t.Fatalf("a whole journal read %d records: %v", len(recs), err)
	}
	for what, body := range map[string]string{
		"torn tail":       good + `{"name":"b.jpg","ke`,
		"no header":       `{"name":"a.jpg","key":"k"}` + "\n",
		"foreign version": `{"journal":"quire-variants","version":99}` + "\n" + strings.SplitN(good, "\n", 2)[1],
		"empty":           "",
	} {
		if _, err := readJournal(strings.NewReader(body)); err == nil {
			t.Errorf("%s: read without an error", what)
		}
	}
}

func TestLadderIsTheMaster(t *testing.T) {
	for _, c := range []struct {
		master int
		want   []int
	}{
		{2400, []int{64, 128, 232, 408, 680, 1096, 1704, 2400}},
		{1704, []int{64, 128, 232, 408, 680, 1096, 1704}},
		{1100, []int{64, 128, 232, 408, 680, 1100}},
		{50, []int{50}},
	} {
		got := LadderFor(c.master)
		if !slices.Equal(got, c.want) {
			t.Errorf("LadderFor(%d) = %v, want %v", c.master, got, c.want)
		}
		if slices.Max(got) != c.master {
			t.Errorf("LadderFor(%d) tops at %d, not the master", c.master, slices.Max(got))
		}
	}
}
