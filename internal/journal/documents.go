package journal

import (
	"encoding/json/v2"
	"fmt"
	"path"
	"strconv"
	"strings"
)

const holeMark = "\x00"

var (
	JSON = Format{Ext: "json", Type: "application/json"}
	YAML = Format{Ext: "yaml", Type: "application/yaml"}
)

type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
	ID    string `json:"id"`
}

type Embed struct {
	Picture string `json:"picture,omitempty"`
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
	Figure  bool   `json:"figure,omitzero"`
	Fence   string `json:"fence,omitempty"`
	Info    string `json:"info,omitempty"`
	Body    string `json:"body,omitempty"`
}

type Document struct {
	Path     string    `json:"path"`
	Front    string    `json:"front"`
	HTML     string    `json:"html"`
	Words    int       `json:"words,omitzero"`
	Headings []Heading `json:"headings,omitempty"`
	Embeds   []Embed   `json:"embeds,omitempty"`
}

func Resolve(doc, ref string) string {
	if rest, ok := strings.CutPrefix(ref, "/"); ok {
		return path.Clean(rest)
	}
	return path.Join(path.Dir(doc), ref)
}

func Hole(i int) string { return holeMark + strconv.Itoa(i) + holeMark }

func (d Document) Fill(embed func(Embed) (string, error)) (string, error) {
	var b strings.Builder
	rest := d.HTML
	for {
		before, after, ok := strings.Cut(rest, holeMark)
		b.WriteString(before)
		if !ok {
			return b.String(), nil
		}
		n, tail, _ := strings.Cut(after, holeMark)
		i, err := strconv.Atoi(n)
		if err != nil || i < 0 || i >= len(d.Embeds) {
			return "", fmt.Errorf("%s: the bundle names embed %q, which it does not hold; rebake", d.Path, n)
		}
		html, err := embed(d.Embeds[i])
		if err != nil {
			return "", fmt.Errorf("%s: %w", d.Path, err)
		}
		b.WriteString(html)
		rest = tail
	}
}

func EncodeDocument(d Document) ([]byte, error) { return json.Marshal(d) }

func DecodeDocument(body []byte) (Document, error) {
	var d Document
	err := json.Unmarshal(body, &d)
	return d, err
}
