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
| [just](./languages.md#just) | 9.49 | – | – |
| [markdown](./languages.md#markdown) | 10.4 | 0.631 | 76% |
| [templ](./languages.md#templ) | 27.6 | 1.07 | 72% |
| [go](./languages.md#go) | 57.2 | 0.639 | 98% |
| [python](./languages.md#python) | 96.8 | 0.5 | 81% |
| [fish](./languages.md#fish) | 103 | 0.634 | 36% |
| [tsx](./languages.md#tsx) | 104 | 0.797 | 74% |
| [typescript](./languages.md#typescript) | 107 | 0.941 | 74% |
| [c](./languages.md#c) | 108 | 0.605 | 95% |
| [cpp](./languages.md#cpp) | 109 | 0.485 | 82% |
| [html](./languages.md#html) | 113 | 1.13 | 91% |
| [javascript](./languages.md#javascript) | 113 | 0.738 | 79% |
| [objc](./languages.md#objc) | 130 | 0.663 | 74% |
| [css](./languages.md#css) | 133 | 0.422 | 67% |
| [objcpp](./languages.md#objcpp) | 136 | – | – |
| [jsx](./languages.md#jsx) | 139 | 0.896 | 85% |
| [scss](./languages.md#scss) | 154 | 0.373 | 63% |
| [xml](./languages.md#xml) | 194 | 5.74 | 85% |
| [json](./languages.md#json) | 264 | 1.8 | 100% |

## From Chroma's rules

These run Chroma's rules on Lexis's engine, slowest first: the order to write them by hand in.
One should agree with Chroma, and where it does not, a rule Lexis rewrote is the first place to look.

| Language | Lexis | Chroma | Agree |
|-|-|-|-|
| [java](./languages.md#java) | 5.2 | 0.85 | 98% |
| [makefile](./languages.md#makefile) | 5.58 | 1.32 | 100% |
| [nix](./languages.md#nix) | 6.06 | 0.822 | 100% |
| [bash](./languages.md#bash) | 6.35 | 1.06 | 100% |
| [zig](./languages.md#zig) | 7.31 | 0.76 | 100% |
| [rust](./languages.md#rust) | 7.54 | 0.616 | 100% |
| [kotlin](./languages.md#kotlin) | 8.15 | 1.33 | 100% |
| [docker](./languages.md#docker) | 8.81 | 1.6 | 100% |
| [yaml](./languages.md#yaml) | 10 | 2.52 | 98% |
| [csharp](./languages.md#csharp) | 10 | 1.43 | 100% |
| [sql](./languages.md#sql) | 10.7 | 0.415 | 100% |
| [lua](./languages.md#lua) | 11.6 | 1.56 | 100% |
| [toml](./languages.md#toml) | 14.5 | 2.07 | 100% |
| [ini](./languages.md#ini) | 20.5 | 4.26 | 100% |
| [diff](./languages.md#diff) | 57.4 | 14.6 | 100% |

`just lexis diff go` lists the text the two disagree on in a language, the commonest first.
