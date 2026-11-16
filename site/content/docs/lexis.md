---
title: Lexis
summary: A syntax highlighter's lexers, as a Go package with no dependencies.
---

Lexis turns source text into tokens, each with a kind such as comment, keyword or string, for a caller to colour.
A lexer gives its tokens one at a time, so a caller takes only as many as it has room to show, which makes it quick enough to run as someone types.

It lives in `libs/lexis`, imports only Go's standard library, and nothing in it is specific to Quire.
Quire highlights the [code in a document](./content.md#code) with it, and [archr](https://github.com/lahickey/archr) highlights source in a terminal.

## Using it

A set of languages starts empty.
`langs.Add` puts in every language Lexis comes with, and the set then finds a lexer by a file's name or by a language's.

```go
package main

import (
  "fmt"

  "github.com/lahickey/quire/libs/lexis"
  "github.com/lahickey/quire/libs/lexis/langs"
)

func main() {
  var set lexis.Languages
  langs.Add(&set)

  next := set.File("main.go", "package main // hello\n")
  for tok, ok := next(); ok; tok, ok = next() {
    fmt.Printf("%d %q\n", tok.Kind, tok.Text)
  }
}
```

`set.Named("go", text)` finds the same lexer by name, as a Markdown fence names its language.
Either gives `nil` when the set has no such language, and the caller shows the text plain.
The tokens' text, joined, is always the text that went in.

| Package | Holds |
|-|-|
| `libs/lexis` | Tokens, kinds, a lexer, a set of languages |
| `libs/lexis/langs` | Every language Lexis comes with |
| `libs/lexis/rules` | The engine that runs a file of rules |

## Kinds

A token is one of eighteen kinds, and a caller gives each a colour.

| Kinds | For |
|-|-|
| `Text` | Anything not marked |
| `Comment`, `Keyword`, `Type`, `Builtin` | Code |
| `String`, `Number`, `Function`, `Tag`, `Preproc` | Code |
| `Operator`, `Punct` | Operators, and brackets and separators |
| `Heading`, `Subheading`, `Strong`, `Emph` | Prose |
| `Inserted`, `Deleted` | Diffs |

## HTML

`lexis.AppendHTML` writes a lexer's tokens as HTML: the text escaped, and each token that is not `Text` in a `span` with a class for its kind.
The class is `lx-` and the kind's name in lower case, which `Kind.String` gives.

```go
out := lexis.AppendHTML(nil, set.Named("go", "return nil // done"))
```

```html
<span class="lx-keyword">return</span> <span class="lx-keyword">nil</span> <span class="lx-comment">// done</span>
```

The caller wraps it in `pre` and `code`, and brings the styles; Quire's are in [the look](./look.md#code).

## Languages

| Language | How |
|-|-|
| Go | Go's own scanner |
| C, C++ | One lexer, a word table each |
| fish | Its own lexer |
| justfile | Its own lexer, a recipe's body going to bash's |
| templ | Its own lexer, handing Go, scripts and styles to theirs |
| Markdown | Its own lexer, a line at a time |
| sh, bash, zsh, ksh | Bash's rules |
| Python, Rust, TypeScript, JavaScript, Java, C#, Kotlin, Zig, Lua, Nix | Rules |
| HTML, CSS, XML, YAML, TOML, JSON, INI, SQL | Rules |
| Dockerfile, Makefile, diff | Rules |

[The languages page](./languages.md) shows the start of a file in each, coloured.

The rules are [Chroma](https://github.com/alecthomas/chroma)'s, which took them from [Pygments](https://pygments.org), and a program that imports `langs` carries both notices; they are in `THIRD_PARTY_NOTICES.md`.

## A language of your own

A language is its names, the files it is for, and a function from text to a lexer.
One added later is found before those added earlier, so a caller can put its own in place of one Lexis has.

```go
set.Add(lexis.Language{
  Names: []string{"conf"},
  Files: []string{"*.conf", "*.rc"},
  Lexer: confLexer,
})
```

`rules.Load` makes a language from a file of rules in Chroma's XML, taking its names from the `<alias>` entries in its `<config>` and its files from the `<filename>` entries.

```go
conf, err := rules.Load(&set, xml, nil)
if err != nil {
  return err
}
set.Add(conf)
```

## Adding a language to Lexis

Everything a language has is in `libs/lexis/langs`, in files named for it.

**From Chroma's rules.**

1. Copy its XML file in and put the notice at its top, as the others have.
   Its `<config>` gives the language its names and files; nothing else lists it.
2. Add a file of real size to `libs/lexis/_corpus`, in a folder named for the language, and run the tests.
   `just lexis docs` then puts it on the languages and benchmarks pages.
   A pattern Go's `regexp` cannot compile fails there, naming the state, as does a `push` that names two states, which wants a `push` for each.
3. Rewrite each such rule.
   `ahead="..."` holds what must follow a match without being taken, and `behind="..."` what the character before it may be.
   `func="..."` names a Go function for what a pattern cannot say: it goes in a Go file beside the rules, as `lua.go` is beside `lua.xml`, with a line in `matchers` in `langs.go`.
4. Add what its tokens should be to `TestKinds`.

**With a lexer written by hand**, for a language used heavily enough to want the speed.

1. Write it in a file named for the language, as `fish.go` is.
   Every name the file declares starts with the file's name, and the last is its `Language`: `fishLanguage` in `fish.go`.
2. List that function in `langs.Add`.
3. A lexer that hands text to other languages embeds `stepper`, as templ's does, and finds them by name in the set.

## Speed

The lexers written by hand run at tens to hundreds of megabytes a second.
Those run from rules manage a few, which is still several times what Chroma does with the same rules.
[The benchmarks](./benchmarks.md) have every language beside Chroma, with how far the two agree.

`just lexis bench` measures them and writes that page, and `just lexis docs` writes it and the languages page together.
It is a module of its own, in `libs/lexis/bench`, so that Quire does not depend on Chroma.
The page lists the languages run from rules slowest first, which is the order to write them by hand in.
