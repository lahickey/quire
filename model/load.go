package model

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/lahickey/quire/internal/journal"
	"go.yaml.in/yaml/v3"
)

func decodeYAML(data []byte, dst any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	return dec.Decode(dst)
}

func load[T any](baked []journal.Document) ([]Document[T], error) {
	out := make([]Document[T], 0, len(baked))
	bySlug := map[string]string{}
	for _, b := range baked {
		var meta T
		if err := decodeYAML([]byte(b.Front), &meta); err != nil {
			return nil, fmt.Errorf("%s: frontmatter: %w", b.Path, err)
		}
		slug := strings.TrimSuffix(path.Base(b.Path), path.Ext(b.Path))
		if other, taken := bySlug[slug]; taken {
			return nil, fmt.Errorf("%s and %s share the slug %q", other, b.Path, slug)
		}
		bySlug[slug] = b.Path
		out = append(out, Document[T]{Slug: slug, Path: b.Path, Meta: meta, HTML: b.HTML, Words: b.Words, Headings: b.Headings})
	}
	return out, nil
}
