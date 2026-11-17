---
title: Lexis benchmarks
summary: Lexis against Chroma, a language at a time.
---

[Lexis](./lexis.md) beside [Chroma](https://github.com/alecthomas/chroma): how fast each splits source text into tokens, and how far they agree on what the tokens are.
`just lexis bench` measures it and writes this page.

- **Lexis** and **Chroma** are megabytes a second.
- **Agree** is how much of the text that either gives a kind to gets the same kind from both, Chroma's token types being mapped to Lexis's kinds first.
- The text is the corpus in `libs/lexis/_corpus`: one file for each language, of about a hundred lines, written to use what the language has.
- A dash is a language Chroma does not have.

Measured with go1.27.4-X:nodwarf5 on Intel(R) Core(TM) i7-1065G7 CPU @ 1.30GHz, one core.

## Written by hand

These have a lexer of their own in Go.
One differs from Chroma wherever it was meant to, and wherever it is wrong.

| Language | Lexis | Chroma | Agree |
|-|-|-|-|
| [just](./languages.md#just) | 9.87 | – | – |
| [markdown](./languages.md#markdown) | 10.8 | 0.647 | 76% |
| [templ](./languages.md#templ) | 28.5 | 1.14 | 72% |
| [go](./languages.md#go) | 54.5 | 0.599 | 98% |
| [python](./languages.md#python) | 98 | 0.519 | 81% |
| [javascript](./languages.md#javascript) | 99 | 0.708 | 79% |
| [typescript](./languages.md#typescript) | 103 | 1 | 74% |
| [fish](./languages.md#fish) | 103 | 0.681 | 36% |
| [tsx](./languages.md#tsx) | 104 | 0.882 | 74% |
| [c](./languages.md#c) | 108 | 0.555 | 95% |
| [cpp](./languages.md#cpp) | 115 | 0.446 | 82% |
| [html](./languages.md#html) | 122 | 1.09 | 91% |
| [objc](./languages.md#objc) | 131 | 0.685 | 74% |
| [jsx](./languages.md#jsx) | 133 | 0.91 | 85% |
| [css](./languages.md#css) | 137 | 0.41 | 67% |
| [objcpp](./languages.md#objcpp) | 137 | – | – |
| [scss](./languages.md#scss) | 162 | 0.344 | 63% |
| [xml](./languages.md#xml) | 212 | 5.78 | 85% |
| [json](./languages.md#json) | 238 | 2 | 100% |

## From Chroma's rules

These run Chroma's rules on Lexis's engine, slowest first: the order to write them by hand in.
One should agree with Chroma, and where it does not, a rule Lexis rewrote is the first place to look.

| Language | Lexis | Chroma | Agree |
|-|-|-|-|
| [java](./languages.md#java) | 5.27 | 0.88 | 98% |
| [makefile](./languages.md#makefile) | 5.69 | 1.41 | 100% |
| [bash](./languages.md#bash) | 6.05 | 1.16 | 100% |
| [nix](./languages.md#nix) | 6.18 | 0.907 | 100% |
| [rust](./languages.md#rust) | 7.05 | 0.614 | 100% |
| [zig](./languages.md#zig) | 7.41 | 0.786 | 100% |
| [kotlin](./languages.md#kotlin) | 8.07 | 1.37 | 100% |
| [docker](./languages.md#docker) | 9.15 | 1.72 | 100% |
| [yaml](./languages.md#yaml) | 9.95 | 2.68 | 98% |
| [csharp](./languages.md#csharp) | 10.8 | 1.52 | 100% |
| [sql](./languages.md#sql) | 11 | 0.425 | 100% |
| [lua](./languages.md#lua) | 11.5 | 1.67 | 100% |
| [toml](./languages.md#toml) | 13.9 | 2.24 | 100% |
| [ini](./languages.md#ini) | 18.7 | 4.72 | 100% |
| [diff](./languages.md#diff) | 57.1 | 14.9 | 100% |

`just lexis diff go` lists the text the two disagree on in a language, the commonest first.
