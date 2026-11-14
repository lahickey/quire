package main

import (
	"bytes"
	"cmp"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/lahickey/quire/examples"
)

const (
	engine   = "github.com/lahickey/quire"
	scaffold = "scaffold"
)

func starters() []string {
	entries, _ := fs.ReadDir(examples.Files, ".")
	var out []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != scaffold {
			out = append(out, e.Name())
		}
	}
	return out
}

func engineVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok || !strings.HasPrefix(bi.Main.Version, "v") || strings.Contains(bi.Main.Version, "+dirty") {
		return "latest"
	}
	return bi.Main.Version
}

func goTool(dir string, args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = dir, os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("go %s: %v", strings.Join(args, " "), err)
	}
}

func copyInto(dir, from string) {
	sub, err := fs.Sub(examples.Files, from)
	if err == nil {
		err = os.CopyFS(dir, sub)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func rename(dir, from, to string) error {
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || (filepath.Ext(p) != ".go" && filepath.Ext(p) != ".templ") {
			return err
		}
		body, err := os.ReadFile(p)
		if err != nil || !bytes.Contains(body, []byte(from)) {
			return err
		}
		return os.WriteFile(p, bytes.ReplaceAll(body, []byte(from), []byte(to)), 0o644)
	})
}

func newSite(args []string) {
	fl := flag.NewFlagSet("new", flag.ExitOnError)
	from := fl.String("from", "1-minimal", "the example to start from: "+strings.Join(starters(), ", "))
	module := fl.String("module", "", "the site's module path (default: the directory's name)")
	replace := fl.String("replace", "", "build against the Quire checkout in this directory")
	noJust := fl.Bool("no-just", false, "write no justfile")
	fl.Usage = func() {
		_, _ = fmt.Fprintln(fl.Output(), "usage: quire new [flags] <dir>")
		fl.PrintDefaults()
	}
	_ = fl.Parse(args)
	if fl.NArg() != 1 {
		fl.Usage()
		os.Exit(2)
	}
	dir, err := filepath.Abs(fl.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	if _, err := fs.Stat(examples.Files, *from+"/main.go"); err != nil {
		log.Fatalf("quire: no example %q; there is %s", *from, strings.Join(starters(), ", "))
	}
	if entries, _ := os.ReadDir(dir); len(entries) > 0 {
		log.Fatalf("quire: %s is not empty", dir)
	}

	copyInto(dir, *from)
	copyInto(dir, scaffold)
	if *noJust {
		_ = os.Remove(filepath.Join(dir, "justfile"))
	}
	mod := cmp.Or(*module, filepath.Base(dir))
	if err := rename(dir, engine+"/examples/"+*from, mod); err != nil {
		log.Fatal(err)
	}

	goTool(dir, "mod", "init", mod)
	if *replace != "" {
		checkout, err := filepath.Abs(*replace)
		if err != nil {
			log.Fatal(err)
		}
		goTool(dir, "mod", "edit", "-require", engine+"@v0.0.0", "-replace", engine+"="+checkout)
	} else {
		goTool(dir, "get", engine+"@"+engineVersion())
	}
	goTool(dir, "mod", "edit", "-tool", "github.com/a-h/templ/cmd/templ")
	goTool(dir, "mod", "tidy")
	fmt.Printf("%s is a copy of %s. In it, `go run .` serves the site on :8090.\n", fl.Arg(0), *from)
}
