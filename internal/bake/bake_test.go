//go:build !quire_nobake

package bake_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lahickey/quire/internal/bake/baketest"
	"github.com/lahickey/quire/internal/bake/picture"
	"github.com/lahickey/quire/internal/testkit"
	"github.com/lahickey/quire/model"
)

func TestJournalOrderPutsImgLast(t *testing.T) {
	s := model.BakeSpec{Content: photoMasters(t, "a.png"), Dir: t.TempDir(), Codecs: []string{"avif", "webp", "jpg"}, AVIFSpeed: 9}
	for range 2 {
		bakeCounted(t, s)
		var exts []string
		size := map[string]int64{}
		for _, vs := range testkit.ReadJournal(t, s.Bundle())["a.png"].Formats {
			exts = append(exts, vs.Ext)
			for _, v := range vs.Out {
				size[vs.Ext] += v.Size
			}
		}
		if n := len(exts); n != 3 || exts[n-1] != "jpg" || size[exts[0]] > size[exts[1]] {
			t.Fatalf("formats %v sized %v, want the smaller <source> first and jpg last", exts, size)
		}
	}
}

func TestACodecThatCannotCarryTheImageFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flat.png"), testkit.Flat(t, 200, 150, true), 0o644); err != nil {
		t.Fatal(err)
	}
	err := baketest.BakeMasters(model.BakeSpec{Content: dir, Dir: t.TempDir(), Codecs: []string{"jpg"}})
	if err == nil || !strings.Contains(err.Error(), "alpha") {
		t.Fatalf("transparent art as JPEG alone: %v", err)
	}
}

func TestDirectoriesMustNotOverlap(t *testing.T) {
	src := photoMasters(t, "a.png")
	for what, dir := range map[string]string{
		"dir holds content": filepath.Dir(src),
		"content holds dir": filepath.Join(src, ".quire"),
	} {
		if err := baketest.BakeMasters(model.BakeSpec{Content: src, Dir: dir}); err == nil {
			t.Errorf("%s: baked without an error", what)
		}
	}
	if _, err := os.Stat(filepath.Join(src, "a.png")); err != nil {
		t.Fatalf("a refused bake touched the content: %v", err)
	}
}

func TestOutOmitsWhatThisBakeDidNotAskFor(t *testing.T) {
	s := model.BakeSpec{Content: photoMasters(t, "a.png"), Dir: t.TempDir(), AVIFSpeed: 9}
	wide, narrow := []string{"avif", "webp", "jpg"}, []string{"webp", "jpg"}
	s.Codecs = wide
	if _, enc := bakeCounted(t, s); enc == 0 || filesIn(t, s.Bundle(), ".avif") == 0 {
		t.Fatal("the first bake shipped no avif")
	}
	avifs := filesIn(t, s.Cache(), ".avif")
	s.Codecs = narrow
	if dec, enc := bakeCounted(t, s); dec != 0 || enc != 0 {
		t.Errorf("a narrower bake decoded %d and encoded %d", dec, enc)
	}
	if filesIn(t, s.Cache(), ".avif") != avifs || filesIn(t, s.Bundle(), ".avif") != 0 {
		t.Fatal("a narrower bake must keep avif in the cache and leave it out of out")
	}
	s.Codecs = wide
	if dec, enc := bakeCounted(t, s); dec != 0 || enc != 0 {
		t.Errorf("returning to the wider set decoded %d and encoded %d", dec, enc)
	}
}

func TestAMissingVariantIsEncodedAgain(t *testing.T) {
	s := model.BakeSpec{Content: photoMasters(t, "a.png"), Dir: t.TempDir()}
	bakeCounted(t, s)
	name := testkit.ReadJournal(t, s.Cache())["a.png"].Formats[0].Out[0].Name
	if err := os.Remove(filepath.Join(s.Cache(), name)); err != nil {
		t.Fatal(err)
	}
	if _, enc := bakeCounted(t, s); enc != 1 {
		t.Errorf("re-encoded %d variants, want the one that went missing", enc)
	}
	if _, err := os.Stat(filepath.Join(s.Bundle(), name)); err != nil {
		t.Error("the variant is still missing from out")
	}
}

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

func bakeCounted(t *testing.T, s model.BakeSpec) (dec, enc int64) {
	t.Helper()
	d0, e0 := picture.Decodes(), picture.Encodes()
	if err := baketest.BakeMasters(s); err != nil {
		t.Fatal(err)
	}
	return picture.Decodes() - d0, picture.Encodes() - e0
}

func filesIn(t *testing.T, dir, ext string) int {
	t.Helper()
	got, err := filepath.Glob(filepath.Join(dir, "*"+ext))
	if err != nil {
		t.Fatal(err)
	}
	return len(got)
}
