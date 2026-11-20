// Package journal is the index a bake writes and a bundle is read from: one record for each thing baked.
package journal

import (
	"bufio"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
)

const (
	Name    = "journal.jsonl"
	magic   = "quire-journal"
	version = 7
)

// Kind is the shape of a record: a picture, a document, a file kept as it was, or the name of the converter that made it.
type Kind string

const (
	KindPicture  Kind = "picture"
	KindDocument Kind = "document"
	KindFile     Kind = "file"
)

type File struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// Record is what every baked thing has in common. Type is set when a browser fetches its one file,
// and Meta is whatever its kind needs beyond that, which only a reader of that kind understands.
type Record struct {
	Name  string         `json:"name"`
	Kind  Kind           `json:"kind"`
	Key   string         `json:"key"`
	Type  string         `json:"type,omitempty"`
	Files []File         `json:"files"`
	Meta  jsontext.Value `json:"meta,omitzero"`
}

// Picture is a picture record's Meta: the master as it was read, the media type of its files, and the width of each.
type Picture struct {
	Hash   string `json:"hash"`
	W      int    `json:"width"`
	H      int    `json:"height"`
	SrcW   int    `json:"srcWidth,omitzero"`
	SrcH   int    `json:"srcHeight,omitzero"`
	Alpha  bool   `json:"alpha,omitzero"`
	Type   string `json:"type"`
	Widths []int  `json:"widths"`
}

type header struct {
	Journal string `json:"journal"`
	Version int    `json:"version"`
	Note    string `json:"note"`
}

func read(src io.Reader) (map[string]Record, error) {
	sc := bufio.NewScanner(src)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	if !sc.Scan() {
		return nil, cmp.Or(sc.Err(), errors.New("quire: journal is empty"))
	}
	var h header
	if err := json.Unmarshal(sc.Bytes(), &h); err != nil {
		return nil, fmt.Errorf("quire: journal header: %w", err)
	}
	if h.Journal != magic || h.Version != version {
		return nil, fmt.Errorf("quire: journal is %q v%d, want %q v%d", h.Journal, h.Version, magic, version)
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

func VideoType(name string) string {
	switch path.Ext(name) {
	case ".webm":
		return "video/webm"
	case ".mp4":
		return "video/mp4"
	}
	return ""
}

func ConvertedName(converter string, body []byte, args []string) string {
	return converter + "/" + Hash(fmt.Appendf(nil, "%s\x00%s", strings.Join(args, "\x00"), body))
}

func (r Record) File() (string, bool) {
	if len(r.Files) != 1 {
		return "", false
	}
	return r.Files[0].Name, true
}

func (r Record) Picture() Picture {
	var p Picture
	_ = json.Unmarshal(r.Meta, &p)
	return p
}

func (r Record) WithPicture(p Picture) Record {
	meta, err := json.Marshal(p)
	if err != nil {
		panic("quire: journal picture: " + err.Error())
	}
	r.Meta = meta
	return r
}

func PictureKey(version int, srcDigest string) string {
	return Hash(fmt.Appendf(nil, "v=%d\nsrc=%s\n", version, srcDigest))
}

func Load(fsys fs.FS) (map[string]Record, error) {
	f, err := fsys.Open(Name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return read(f)
}

func Write(records []Record) []byte {
	b, err := json.Marshal(header{
		Journal: magic,
		Version: version,
		Note:    "written by the bake — everything here is rebuildable and safe to delete",
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
