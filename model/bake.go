package model

import (
	"cmp"
	"log/slog"
	"path/filepath"
)

const QuireDir = ".quire"

// BakeSpec is where a bake reads, where it writes, and where it reports what it made.
type BakeSpec struct {
	Content string
	Dir     string
	Log     *slog.Logger
}

func (s BakeSpec) WithDefaults() BakeSpec {
	s.Content = cmp.Or(s.Content, "content")
	s.Dir = cmp.Or(s.Dir, QuireDir)
	s.Log = cmp.Or(s.Log, slog.Default())
	return s
}

func (s BakeSpec) Bundle() string { return filepath.Join(s.Dir, "bundle") }

func (s BakeSpec) Cache() string { return filepath.Join(s.Dir, "cache") }
