package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

func output(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		log.Fatalf("%s %s: %v", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}

func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("%s %s: %v", name, strings.Join(args, " "), err)
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("release: ")
	if len(os.Args) != 2 || semver.Canonical(os.Args[1]) != os.Args[1] {
		log.Fatal("usage: just release vMAJOR.MINOR.PATCH")
	}
	v := os.Args[1]
	for t := range strings.FieldsSeq(output("git", "tag", "-l", "v*")) {
		if t == v {
			log.Fatalf("%s is already a tag", v)
		}
		if semver.Compare(t, v) > 0 {
			log.Fatalf("%s does not come after %s", v, t)
		}
	}
	if output("git", "branch", "--show-current") != "master" {
		log.Fatal("not on master")
	}
	if output("git", "status", "--porcelain") != "" {
		log.Fatal("the tree has uncommitted changes")
	}
	data, err := os.ReadFile("go.mod")
	if err != nil {
		log.Fatal(err)
	}
	mod, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		log.Fatal(err)
	}
	if len(mod.Replace) > 0 {
		log.Fatal("go.mod has a replace")
	}
	if exec.Command("go", "mod", "tidy", "-diff").Run() != nil {
		log.Fatal("go.mod or go.sum is not tidy: run go mod tidy")
	}
	if _, err := exec.LookPath("govulncheck"); err != nil {
		log.Fatal("govulncheck is not installed: go install golang.org/x/vuln/cmd/govulncheck@latest")
	}
	run("just", "release-check")
	run("govulncheck", "./...")
	run("just", "test")
	if output("git", "status", "--porcelain") != "" {
		log.Fatal("the tests changed the tree: commit what they generated")
	}
	run("git", "tag", "-a", v, "-m", v)
	fmt.Printf("tagged %s; push it, then fetch it through the proxy so pkg.go.dev has it:\n", v)
	fmt.Printf("  git push public master %s\n", v)
	fmt.Printf("  GOPROXY=https://proxy.golang.org go list -m %s@%s\n", mod.Module.Mod.Path, v)
}
