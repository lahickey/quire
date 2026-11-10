package picture

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/lahickey/quire/internal/bake/cache"
	"github.com/lahickey/quire/internal/journal"
)

func outStem(name string) (string, error) {
	s := filepath.ToSlash(name)
	s = strings.TrimSuffix(s, path.Ext(s))
	if s == "" || s == "." || s == ".." || path.Base(s) != s {
		return "", fmt.Errorf("quire: image %q: a single path element", name)
	}
	return s, nil
}

func Ingest(dir, name string, src []byte, lossless bool) (journal.MasterMeta, string, error) {
	stem, err := outStem(name)
	if err != nil {
		return journal.MasterMeta{}, "", err
	}
	m, ext, body, err := master(name, src, lossless)
	if err != nil {
		return journal.MasterMeta{}, "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return journal.MasterMeta{}, "", err
	}
	master := stem + "." + ext
	if err := cache.Dir(dir).Write(master, body); err != nil {
		return journal.MasterMeta{}, "", err
	}
	return m, master, nil
}
