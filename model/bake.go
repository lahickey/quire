package model

import (
	"cmp"
	"path/filepath"
)

const QuireDir = ".quire"

type BakeSpec struct {
	Content string
	Dir     string
}

type Ink struct {
	Bg, Text, Secondary, Muted, Accent, Spot string
}

type ShareCard struct {
	Kicker string
	Title  string
	Lines  []string
	Domain string
	Ink    Ink
}

func (s BakeSpec) WithDefaults() BakeSpec {
	s.Content = cmp.Or(s.Content, "content")
	s.Dir = cmp.Or(s.Dir, QuireDir)
	return s
}

func (s BakeSpec) Bundle() string { return filepath.Join(s.Dir, "bundle") }

func (s BakeSpec) Cache() string { return filepath.Join(s.Dir, "cache") }
