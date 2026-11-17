// Command bench sets Lexis beside Chroma on the same text, a language at a
// time: how fast each is, and how far they agree on what the text is. Given
// a file, it writes the documentation's page of the results there; given
// diff and a language, it prints where the two differ. It is a module of
// its own so that Quire does not depend on Chroma.
package main

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/lahickey/quire/libs/lexis"
	"github.com/lahickey/quire/libs/lexis/langs"
	lexisrules "github.com/lahickey/quire/libs/lexis/rules"
)

const (
	corpus = "../_corpus"
	rules  = "../langs"
)

// lexisEngine is how to lex text in the language called name with Lexis,
// giving the kind of each of its bytes.
func lexisEngine(set *lexis.Languages, name, text string) func() []lexis.Kind {
	return func() []lexis.Kind {
		kinds, at := make([]lexis.Kind, len(text)), 0
		next := set.Named(name, text)
		for t, ok := next(); ok; t, ok = next() {
			for i := range len(t.Text) {
				kinds[at+i] = t.Kind
			}
			at += len(t.Text)
		}
		return kinds
	}
}

// chromaEngine is the same with Chroma, its token types as the kinds Lexis
// gives them, or nil when Chroma does not have the language.
func chromaEngine(name, text string) func() []lexis.Kind {
	lexer := lexers.Get(name)
	if lexer == nil {
		return nil
	}
	return func() []lexis.Kind {
		kinds, at := make([]lexis.Kind, len(text)), 0
		next, err := lexer.Tokenise(nil, text)
		if err != nil {
			panic(err)
		}
		for t := next(); t != chroma.EOF; t = next() {
			kind := lexisrules.KindOf(t.Type.String())
			for i := at; i < min(at+len(t.Value), len(kinds)); i++ {
				kinds[i] = kind
			}
			at += len(t.Value)
		}
		return kinds
	}
}

// guarded is run with a panic in it turned into nil.
func guarded(run func() []lexis.Kind) (kinds []lexis.Kind) {
	if run == nil {
		return nil
	}
	defer func() {
		if recover() != nil {
			kinds = nil
		}
	}()
	return run()
}

// rate is how many megabytes of text a second run gets through, 0 when it
// is nil or fails.
func rate(text string, run func() []lexis.Kind) float64 {
	if guarded(run) == nil {
		return 0
	}
	r := testing.Benchmark(func(b *testing.B) {
		for b.Loop() {
			run()
		}
	})
	return float64(len(text)) * float64(r.N) / 1e6 / r.T.Seconds()
}

// agreement is how much of the text that either colours gets the same kind
// from both, as a percentage, or -1 when there is no other.
func agreement(text string, ours, theirs []lexis.Kind) float64 {
	if theirs == nil {
		return -1
	}
	same, coloured := 0, 0
	for i := range text {
		if strings.IndexByte(" \t\r\n", text[i]) >= 0 || ours[i] == lexis.Text && theirs[i] == lexis.Text {
			continue
		}
		coloured++
		if ours[i] == theirs[i] {
			same++
		}
	}
	return 100 * float64(same) / float64(max(coloured, 1))
}

type row struct {
	name                     string
	lexis, chroma, agreement float64
}

func measure(set *lexis.Languages, name, text string) row {
	ours, theirs := lexisEngine(set, name, text), chromaEngine(name, text)
	return row{name, rate(text, ours), rate(text, theirs), agreement(text, ours(), guarded(theirs))}
}

// diff prints the pieces of text Lexis and Chroma give different kinds, the
// commonest first.
func diff(text string, ours, theirs []lexis.Kind) {
	type piece struct {
		text         string
		ours, theirs lexis.Kind
	}
	counts := map[piece]int{}
	for i := 0; i < len(text); {
		j := i + 1
		for j < len(text) && ours[j] == ours[i] && theirs[j] == theirs[i] {
			j++
		}
		if ours[i] != theirs[i] && strings.TrimSpace(text[i:j]) != "" {
			counts[piece{strings.TrimSpace(text[i:j]), ours[i], theirs[i]}]++
		}
		i = j
	}
	pieces := slices.SortedFunc(maps.Keys(counts), func(a, b piece) int {
		return cmp.Or(cmp.Compare(counts[b], counts[a]), cmp.Compare(a.text, b.text))
	})
	fmt.Printf("%5s  %-10s %-10s %s\n", "times", "lexis", "Chroma", "text")
	for _, p := range pieces {
		fmt.Printf("%5d  %-10s %-10s %.60q\n", counts[p], p.ours, p.theirs, p.text)
	}
}

func table(b *strings.Builder, rows []row) {
	cell := func(v float64, format string) string {
		if v <= 0 {
			return "–"
		}
		return fmt.Sprintf(format, v)
	}
	b.WriteString("| Language | Lexis | Chroma | Agree |\n|-|-|-|-|\n")
	for _, r := range rows {
		fmt.Fprintf(b, "| [%s](./languages.md#%[1]s) | %s | %s | %s |\n", r.name, cell(r.lexis, "%.3g"), cell(r.chroma, "%.3g"), cell(r.agreement, "%.0f%%"))
	}
}

func cpu() string {
	b, _ := os.ReadFile("/proc/cpuinfo")
	if m := regexp.MustCompile(`model name\s*:\s*(.+)`).FindSubmatch(b); m != nil {
		return string(m[1])
	}
	return runtime.GOARCH
}

// read is the text of the corpus's file for the language called name.
func read(name string) string {
	paths, _ := filepath.Glob(filepath.Join(corpus, name, "*"))
	if len(paths) != 1 {
		log.Fatalf("%s: want one file in the corpus, have %d", name, len(paths))
	}
	b, err := os.ReadFile(paths[0])
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}

func main() {
	testing.Init()
	if err := flag.Set("test.benchtime", "200ms"); err != nil {
		log.Fatal(err)
	}
	var set lexis.Languages
	langs.Add(&set)
	if len(os.Args) == 3 && os.Args[1] == "diff" {
		name := os.Args[2]
		text := read(name)
		theirs := guarded(chromaEngine(name, text))
		if theirs == nil {
			log.Fatalf("Chroma does not have %s", name)
		}
		diff(text, lexisEngine(&set, name, text)(), theirs)
		return
	}
	if len(os.Args) != 2 {
		log.Fatal("usage: bench <page.md> | bench diff <language>")
	}
	files, err := os.ReadDir(corpus)
	if err != nil {
		log.Fatal(err)
	}
	var names [2][]string
	for _, f := range files {
		i := 0
		if _, err := os.Stat(filepath.Join(rules, f.Name()+".xml")); err == nil {
			i = 1
		}
		names[i] = append(names[i], f.Name())
	}
	var rows [2][]row
	for i := range names {
		for _, name := range names[i] {
			fmt.Fprintf(os.Stderr, "%s ", name)
			rows[i] = append(rows[i], measure(&set, name, read(name)))
		}
		slices.SortFunc(rows[i], func(x, y row) int { return cmp.Compare(x.lexis, y.lexis) })
	}
	fmt.Fprintf(os.Stderr, "\n\n%-12s %8s %8s %6s\n%-12s %8s %8s\n", "", "lexis", "Chroma", "agree", "", "MB/s", "MB/s")
	for i, title := range []string{"written by hand", "run from rules"} {
		fmt.Fprintf(os.Stderr, "\n%s\n", title)
		for _, r := range rows[i] {
			if r.chroma == 0 {
				fmt.Fprintf(os.Stderr, "%-12s %8.3g %8s %6s\n", r.name, r.lexis, "–", "–")
			} else {
				fmt.Fprintf(os.Stderr, "%-12s %8.3g %8.3g %5.0f%%\n", r.name, r.lexis, r.chroma, r.agreement)
			}
		}
	}
	var b strings.Builder
	b.WriteString(`---
title: Lexis benchmarks
summary: Lexis against Chroma, a language at a time.
---

[Lexis](./lexis.md) beside [Chroma](https://github.com/alecthomas/chroma): how fast each splits source text into tokens, and how far they agree on what the tokens are.
` + "`just lexis bench`" + ` measures it and writes this page.

- **Lexis** and **Chroma** are megabytes a second.
- **Agree** is how much of the text that either gives a kind to gets the same kind from both, Chroma's token types being mapped to Lexis's kinds first.
- The text is the corpus in ` + "`libs/lexis/_corpus`" + `: one file for each language, of about a hundred lines, written to use what the language has.
- A dash is a language Chroma does not have.

`)
	fmt.Fprintf(&b, "Measured with %s on %s, one core.\n", runtime.Version(), cpu())
	b.WriteString("\n## Written by hand\n\nThese have a lexer of their own in Go.\nOne differs from Chroma wherever it was meant to, and wherever it is wrong.\n\n")
	table(&b, rows[0])
	b.WriteString("\n## From Chroma's rules\n\nThese run Chroma's rules on Lexis's engine, slowest first: the order to write them by hand in.\nOne should agree with Chroma, and where it does not, a rule Lexis rewrote is the first place to look.\n\n")
	table(&b, rows[1])
	b.WriteString("\n`just lexis diff go` lists the text the two disagree on in a language, the commonest first.\n")
	if err := os.WriteFile(os.Args[1], []byte(b.String()), 0o644); err != nil {
		log.Fatal(err)
	}
}
