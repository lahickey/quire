package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lahickey/quire/internal/bake/picture"
	"github.com/lahickey/quire/internal/journal"
)

const usage = `usage: quire <command>
commands:
  ingest [-lossless] <dir> <name> <file>
                                  write a web master; -lossless for one whose
                                  variants are to be lossless

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
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	lossless := fs.Bool("lossless", false, "a lossless master, for an image whose variants are to be lossless")
	if err := fs.Parse(args); err != nil || fs.NArg() != 3 {
		log.Fatal("usage: quire ingest [-lossless] <dir> <name> <file>")
	}
	src, err := os.ReadFile(fs.Arg(2))
	if err != nil {
		log.Fatal(err)
	}
	m, written, err := picture.Ingest(fs.Arg(0), fs.Arg(1), src, *lossless)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s %d×%d → %s\n", fs.Arg(1), m.W, m.H, filepath.Join(fs.Arg(0), written))

	if m.SrcW > m.W || m.SrcH > m.H {
		fmt.Printf("  scaled down from %d×%d (%s source). The master caps at %d.\n",
			m.SrcW, m.SrcH, humanBytes(len(src)), journal.MasterCap)
	}
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		log.Fatal(usage)
	}
	switch os.Args[1] {
	case "ingest":
		ingest(os.Args[2:])
	default:
		log.Fatalf("quire: unknown command %q\n%s", os.Args[1], usage)
	}
}
