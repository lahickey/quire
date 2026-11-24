package main

import (
	"encoding/json/v2"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lahickey/quire/internal/measure"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base/convert/picture"
)

const usage = `usage: quire <command>
commands:
  new [flags] <dir>             start a site from an example
  ingest <dir> <name> <file>    write a web master
  fonts <dir> <face>...         write character widths into a font set

A site bakes itself when it runs; go run . bake bakes and exits.`

func humanBytes(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%dKB", n/(1<<10))
	}
	return fmt.Sprintf("%dB", n)
}

func ingest(args []string) {
	if len(args) != 3 {
		log.Fatal("usage: quire ingest <dir> <name> <file>")
	}
	src, err := os.ReadFile(args[2])
	if err != nil {
		log.Fatal(err)
	}
	m, written, err := picture.Ingest(args[0], args[1], src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s %d×%d → %s\n", args[1], m.W, m.H, filepath.Join(args[0], written))

	if m.SrcW > m.W || m.SrcH > m.H {
		fmt.Printf("  scaled down from %d×%d (%s source). The master caps at %d.\n",
			m.SrcW, m.SrcH, humanBytes(len(src)), model.MasterCap)
	}
}

// fonts writes into dir's fonts.json how wide each family's characters are, from a TTF for each weight: text:400=inter-400.ttf.
func fonts(args []string) {
	const use = "usage: quire fonts <dir> text:400=<ttf> text:700=<ttf> mono:400=<ttf> ..."
	if len(args) < 2 {
		log.Fatal(use)
	}
	file := filepath.Join(args[0], model.FontsFile)
	set, err := model.ReadFontSet(os.DirFS(args[0]))
	if err != nil {
		log.Fatal(err)
	}
	set.Text.Widths, set.Mono.Widths = map[int]model.FontWidths{}, map[int]model.FontWidths{}
	for _, arg := range args[1:] {
		face, ttf, _ := strings.Cut(arg, "=")
		family, weight, _ := strings.Cut(face, ":")
		w, err := strconv.Atoi(weight)
		fam := map[string]*model.FontFamily{"text": &set.Text, "mono": &set.Mono}[family]
		if err != nil || fam == nil || ttf == "" {
			log.Fatalf("%s: not a face\n%s", arg, use)
		}
		body, err := os.ReadFile(ttf)
		if err != nil {
			log.Fatal(err)
		}
		var ranges []string
		for _, f := range fam.Faces {
			ranges = append(ranges, f.Range)
		}
		if fam.Widths[w], err = measure.Widths(body, ranges...); err != nil {
			log.Fatalf("%s: %v", ttf, err)
		}
		fmt.Printf("%s %d: %d characters that are not %d wide\n", fam.Name, w, len(fam.Widths[w].Of), fam.Widths[w].Default)
	}
	b, err := json.Marshal(set, json.Deterministic(true))
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(file, append(b, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		log.Fatal(usage)
	}
	switch os.Args[1] {
	case "new":
		newSite(os.Args[2:])
	case "ingest":
		ingest(os.Args[2:])
	case "fonts":
		fonts(os.Args[2:])
	default:
		log.Fatalf("quire: unknown command %q\n%s", os.Args[1], usage)
	}
}
