package journal

import (
	"bufio"
	"cmp"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
)

const (
	JournalName    = "journal.jsonl"
	journalMagic   = "quire-variants"
	journalVersion = 2
)

type Variant struct {
	Name string `json:"k"`
	W    int    `json:"w"`
	Size int64  `json:"n"`
}

type Variants struct {
	Ext  string    `json:"f"`
	Type string    `json:"t"`
	Q    int       `json:"q,omitzero"`
	Less bool      `json:"l,omitzero"`
	Out  []Variant `json:"out"`
}

func (vs Variants) Format() Format {
	return Format{Ext: vs.Ext, Type: vs.Type, Quality: vs.Q, Lossless: vs.Less}
}

type Record struct {
	Name    string     `json:"name"`
	Key     string     `json:"key"`
	Prep    MasterMeta `json:"prep"`
	Formats []Variants `json:"formats"`
}

func (r Record) File() (string, bool) {
	if len(r.Formats) != 1 || len(r.Formats[0].Out) != 1 {
		return "", false
	}
	return r.Formats[0].Out[0].Name, true
}

func (r Record) Files() []Variant {
	var out []Variant
	for _, vs := range r.Formats {
		out = append(out, vs.Out...)
	}
	return out
}

type journalHeader struct {
	Journal string `json:"journal"`
	Version int    `json:"version"`
	Note    string `json:"note"`
}

func readJournal(src io.Reader) (map[string]Record, error) {
	sc := bufio.NewScanner(src)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	if !sc.Scan() {
		return nil, cmp.Or(sc.Err(), errors.New("quire: journal is empty"))
	}
	var h journalHeader
	if err := json.Unmarshal(sc.Bytes(), &h); err != nil {
		return nil, fmt.Errorf("quire: journal header: %w", err)
	}
	if h.Journal != journalMagic || h.Version != journalVersion {
		return nil, fmt.Errorf("quire: journal is %q v%d, want %q v%d", h.Journal, h.Version, journalMagic, journalVersion)
	}
	out := map[string]Record{}
	for line := 2; sc.Scan(); line++ {
		var r Record
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("quire: journal line %d: %w", line, err)
		}
		if r.Name == "" {
			return nil, fmt.Errorf("quire: journal line %d: a record with no name", line)
		}
		out[r.Name] = r
	}
	return out, sc.Err()
}

func LoadJournal(fsys fs.FS) (map[string]Record, error) {
	f, err := fsys.Open(JournalName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return readJournal(f)
}

func WriteJournal(records []Record) []byte {
	b, err := json.Marshal(journalHeader{
		Journal: journalMagic,
		Version: journalVersion,
		Note:    "generated variants — everything here is rebuildable and safe to delete",
	})
	if err != nil {
		panic("quire: journal header: " + err.Error())
	}
	out := append(b, '\n')
	slices.SortFunc(records, func(a, b Record) int { return cmp.Compare(a.Name, b.Name) })
	for _, r := range records {
		line, err := json.Marshal(r)
		if err != nil {
			panic("quire: journal record: " + err.Error())
		}
		out = append(append(out, line...), '\n')
	}
	return out
}

var ladder = []int{64, 128, 232, 408, 680, 1096, 1704, MasterCap}

func Key(version int, srcDigest string) string {
	return Hash(fmt.Appendf(nil, "v=%d\nsrc=%s\nladder=%v\n", version, srcDigest, ladder))
}

const ladderSame = 1.15

func Ladder() []int { return slices.Clone(ladder) }

func LadderFor(masterW int) []int {
	if masterW < 1 {
		return nil
	}
	out := make([]int, 0, len(ladder)+1)
	for _, w := range ladder {
		if w >= masterW {
			break
		}
		out = append(out, w)
	}
	if n := len(out); n > 0 && float64(masterW) < ladderSame*float64(out[n-1]) {
		out = out[:n-1]
	}
	return append(out, masterW)
}

func Snap(ladder []int, want int) int {
	for _, w := range ladder {
		if w >= want {
			return w
		}
	}
	if len(ladder) == 0 {
		return want
	}
	return ladder[len(ladder)-1]
}

const (
	ShareWidth = 1200
	CardHeight = 630
)

const SiteCard = "card"
