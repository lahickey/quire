//go:build release

package bake

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type listed struct {
	ImportPath string
	Dir        string
	Standard   bool
	GoFiles    []string
	EmbedFiles []string
	Module     *struct{ Path, Version, Sum string }
}

func fingerprint(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-deps", "-json", "./"+dir).Output()
	if err != nil {
		t.Fatalf("go list %s: %v", dir, err)
	}
	var lines []string
	dec := jsontext.NewDecoder(strings.NewReader(string(out)))
	for {
		var p listed
		if err := json.UnmarshalDecode(dec, &p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		switch {
		case p.Standard:
		case p.Module != nil && p.Module.Sum != "":
			lines = append(lines, p.Module.Path+"@"+p.Module.Version+" "+p.Module.Sum)
		default:
			h := sha256.New()
			for _, f := range append(p.GoFiles, p.EmbedFiles...) {
				body, err := os.ReadFile(filepath.Join(p.Dir, f))
				if err != nil {
					t.Fatal(err)
				}
				fmt.Fprintf(h, "%s\x00%d\x00", f, len(body))
				h.Write(body)
			}
			lines = append(lines, p.ImportPath+" "+hex.EncodeToString(h.Sum(nil)))
		}
	}
	slices.Sort(lines)
	sum := sha256.Sum256([]byte(strings.Join(slices.Compact(lines), "\n")))
	return hex.EncodeToString(sum[:8])
}

func TestConverterFingerprints(t *testing.T) {
	var changed []string
	var table strings.Builder
	for _, c := range converters {
		got := fingerprint(t, c.dir)
		if got != c.fingerprint {
			changed = append(changed, c.dir)
		}
		fmt.Fprintf(&table, "\t{%q, %d, %q},\n", c.dir, c.version, got)
	}
	if len(changed) > 0 {
		t.Fatalf("changed since the last release: %s\nBump the version of each whose output changed, then make the table in versions.go:\n\nvar converters = []converter{\n%s}",
			strings.Join(changed, ", "), table.String())
	}
}
