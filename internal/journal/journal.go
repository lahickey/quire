package journal

import (
	"bufio"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
)

const (
	JournalName    = "journal.jsonl"
	journalMagic   = "quire-variants"
	journalVersion = 3
)

const SiteCard = "card"

type Kind string

const (
	KindPicture  Kind = "picture"
	KindDocument Kind = "document"
	KindDiagram  Kind = "diagram"
)

type Format struct {
	Ext      string
	Type     string
	Quality  int
	Alpha    bool
	Lossless bool
}

type MasterMeta struct {
	Hash  string `json:"h"`
	W     int    `json:"w"`
	H     int    `json:"y"`
	SrcW  int    `json:"sw,omitzero"`
	SrcH  int    `json:"sh,omitzero"`
	Alpha bool   `json:"a,omitzero"`
}

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

type Record struct {
	Name    string     `json:"name"`
	Kind    Kind       `json:"kind"`
	Key     string     `json:"key"`
	Prep    MasterMeta `json:"prep,omitzero"`
	Formats []Variants `json:"formats"`
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

func Hash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:16])
}

func DiagramName(src, fontID string) string {
	return "mermaid/" + Hash([]byte(fontID+"\x00"+strings.TrimSpace(src)))
}

func (vs Variants) Format() Format {
	return Format{Ext: vs.Ext, Type: vs.Type, Quality: vs.Q, Lossless: vs.Less}
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
