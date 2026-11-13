//go:build !quire_nobake

package bake_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/lahickey/quire/internal/bake"
	"github.com/lahickey/quire/internal/bake/picture"
	"github.com/lahickey/quire/internal/testkit"
	"github.com/lahickey/quire/model"
)

func photoMasters(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(n)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, n), testkit.Photo(t, 600, 400, false), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func bakeAll(s model.BakeSpec) error {
	b, err := bake.Start(s)
	if err != nil {
		return err
	}
	err = fs.WalkDir(os.DirFS(s.Content), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		_, err = b.Master(name)
		return err
	})
	if err != nil {
		return err
	}
	return b.Finish()
}

func bakeCounted(t *testing.T, s model.BakeSpec) (enc int64) {
	t.Helper()
	e0 := picture.Encodes()
	if err := bakeAll(s); err != nil {
		t.Fatal(err)
	}
	return picture.Encodes() - e0
}

func TestDirectoriesMustNotOverlap(t *testing.T) {
	src := photoMasters(t, "a.png")
	for what, dir := range map[string]string{
		"dir holds content": filepath.Dir(src),
		"content holds dir": filepath.Join(src, ".quire"),
	} {
		if err := bakeAll(model.BakeSpec{Content: src, Dir: dir}); err == nil {
			t.Errorf("%s: baked without an error", what)
		}
	}
	if _, err := os.Stat(filepath.Join(src, "a.png")); err != nil {
		t.Fatalf("a refused bake touched the content: %v", err)
	}
}

func TestAMissingVariantIsEncodedAgain(t *testing.T) {
	s := model.BakeSpec{Content: photoMasters(t, "a.png"), Dir: t.TempDir()}
	bakeCounted(t, s)
	name := testkit.ReadJournal(t, s.Cache())["a.png"].Files[0].Name
	if err := os.Remove(filepath.Join(s.Cache(), name)); err != nil {
		t.Fatal(err)
	}
	if enc := bakeCounted(t, s); enc != 1 {
		t.Errorf("re-encoded %d variants, want the one that went missing", enc)
	}
	if _, err := os.Stat(filepath.Join(s.Bundle(), name)); err != nil {
		t.Error("the variant is still missing from out")
	}
}
