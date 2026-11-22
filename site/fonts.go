package site

import (
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/lahickey/quire/model"
)

const (
	sansStack = `system-ui, -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif`
	monoStack = `ui-monospace, Menlo, Consolas, monospace`
)

func family(name, stack string) string {
	if name == "" {
		return stack
	}
	return strconv.Quote(name) + ", " + stack
}

// fontCSS is a font set's @font-face rules and the two properties sheets read, with each face served and the Latin text face preloaded.
func fontCSS(set model.FontSet, fsys fs.FS, serve func(file string, body []byte) string) (css []byte, preload []string) {
	var b strings.Builder
	for _, f := range []model.FontFamily{set.Text, set.Mono} {
		for _, face := range f.Faces {
			body, err := fs.ReadFile(fsys, face.File)
			if err != nil {
				panic("quire: font " + face.File + ": " + err.Error())
			}
			url := serve(face.File, body)
			fmt.Fprintf(&b, "@font-face {\n  font-family: %q;\n  font-style: %s;\n  font-weight: %s;\n  font-display: swap;\n  src: url(%q) format(\"woff2\");\n  unicode-range: %s;\n}\n\n",
				f.Name, face.Style, face.Weight, url, face.Range)
			if f.Name == set.Text.Name && face.Subset == "latin" {
				preload = append(preload, url)
			}
		}
	}
	fmt.Fprintf(&b, ":root {\n  --qr-font: %s;\n  --qr-font-mono: %s;\n}\n",
		family(set.Text.Name, sansStack), family(set.Mono.Name, monoStack))
	return []byte(b.String()), preload
}
