package picture

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/lahickey/quire/bake"
	"github.com/lahickey/quire/internal/cache"
)

func outStem(name string) (string, error) {
	s := filepath.ToSlash(name)
	s = strings.TrimSuffix(s, path.Ext(s))
	if s == "" || s == "." || s == ".." || path.Base(s) != s {
		return "", fmt.Errorf("quire: picture %q: a single path element", name)
	}
	return s, nil
}

func Ingest(dir, name string, src []byte) (bake.MasterInfo, string, error) {
	stem, err := outStem(name)
	if err != nil {
		return bake.MasterInfo{}, "", err
	}
	m, ext, body, err := master(name, src)
	if err != nil {
		return bake.MasterInfo{}, "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return bake.MasterInfo{}, "", err
	}
	master := stem + "." + ext
	if err := cache.Dir(dir).Write(master, body); err != nil {
		return bake.MasterInfo{}, "", err
	}
	return m, master, nil
}
