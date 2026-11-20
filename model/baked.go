package model

import (
	"encoding/json/v2"
	"fmt"
	"path"
	"strconv"
	"strings"
)

const slotMark = "\x00"

// Slot is a place in a baked document that something goes: a picture or video beside the text, or a named fence.
// The bake fills a fence it can, with HTML, and leaves the rest with their Body for the site to fill.
type Slot struct {
	Picture string `json:"picture,omitempty"`
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
	Figure  bool   `json:"figure,omitzero"`
	Fence   string `json:"fence,omitempty"`
	Info    string `json:"info,omitempty"`
	Body    string `json:"body,omitempty"`
	HTML    string `json:"html,omitempty"`
}

// BakedDocument is a document as a converter leaves it: its raw frontmatter, and HTML with a mark where each slot goes.
type BakedDocument struct {
	Path     string    `json:"path"`
	Front    string    `json:"front"`
	HTML     string    `json:"html"`
	Words    int       `json:"words,omitzero"`
	Headings []Heading `json:"headings,omitempty"`
	Slots    []Slot    `json:"slots,omitempty"`
}

func Resolve(doc, ref string) string {
	if rest, ok := strings.CutPrefix(ref, "/"); ok {
		return path.Clean(rest)
	}
	return path.Join(path.Dir(doc), ref)
}

// SlotMark is what a converter writes in the HTML where slot i goes.
func SlotMark(i int) string { return slotMark + strconv.Itoa(i) + slotMark }

func (d BakedDocument) Fill(fill func(Slot) (string, error)) (string, error) {
	var b strings.Builder
	rest := d.HTML
	for {
		before, after, ok := strings.Cut(rest, slotMark)
		b.WriteString(before)
		if !ok {
			return b.String(), nil
		}
		n, tail, _ := strings.Cut(after, slotMark)
		i, err := strconv.Atoi(n)
		if err != nil || i < 0 || i >= len(d.Slots) {
			return "", fmt.Errorf("%s: the bundle names slot %q, which it does not hold; rebake", d.Path, n)
		}
		html, err := fill(d.Slots[i])
		if err != nil {
			return "", fmt.Errorf("%s: %w", d.Path, err)
		}
		b.WriteString(html)
		rest = tail
	}
}

func (d BakedDocument) Encode() ([]byte, error) { return json.Marshal(d) }

func DecodeBaked(body []byte) (BakedDocument, error) {
	var d BakedDocument
	err := json.Unmarshal(body, &d)
	return d, err
}
