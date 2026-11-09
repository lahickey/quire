package model

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"
)

func decodeYAML(data []byte, dst any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	return dec.Decode(dst)
}

func load[T any](r Resolver, pattern string, widgets map[string]Widget) ([]Document[T], error) {
	baked, err := r.Documents(pattern, widgets)
	if err != nil {
		return nil, err
	}
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

func loadSeq[T any](r Resolver, name string) ([]Document[T], error) {
	raw, err := r.Data(name)
	if err != nil {
		return nil, err
	}
	var items []T
	if err := decodeYAML(raw, &items); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	stem := strings.TrimSuffix(path.Base(name), path.Ext(name))
	out := make([]Document[T], len(items))
	for i, meta := range items {
		out[i] = Document[T]{
			Slug: fmt.Sprintf("%s-%d", stem, i+1),
			Path: name,
			Meta: meta,
		}
	}
	return out, nil
}
