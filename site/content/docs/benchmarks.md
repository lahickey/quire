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
| [markdown](./languages.md#markdown) | 9.9 | 0.743 | 76% |
| [just](./languages.md#just) | 9.98 | – | – |
| [templ](./languages.md#templ) | 20.1 | 1.18 | 74% |
| [go](./languages.md#go) | 69.6 | 0.746 | 98% |
| [fish](./languages.md#fish) | 102 | 0.701 | 36% |
| [c](./languages.md#c) | 108 | 0.639 | 95% |
| [cpp](./languages.md#cpp) | 114 | 0.505 | 82% |

## From Chroma's rules

These run Chroma's rules on Lexis's engine, slowest first: the order to write them by hand in.
One should agree with Chroma, and where it does not, a rule Lexis rewrote is the first place to look.

| Language | Lexis | Chroma | Agree |
|-|-|-|-|
| [java](./languages.md#java) | 5.19 | 0.881 | 98% |
| [typescript](./languages.md#typescript) | 5.32 | 1.05 | 100% |
| [makefile](./languages.md#makefile) | 5.52 | 1.36 | 100% |
| [python](./languages.md#python) | 5.72 | 0.524 | 100% |
| [bash](./languages.md#bash) | 6.03 | 0.874 | 100% |
| [nix](./languages.md#nix) | 6.08 | 0.87 | 100% |
| [javascript](./languages.md#javascript) | 6.25 | 0.896 | 100% |
| [rust](./languages.md#rust) | 7.28 | 0.603 | 100% |
| [zig](./languages.md#zig) | 7.29 | 0.793 | 100% |
| [css](./languages.md#css) | 7.31 | 0.461 | 100% |
| [kotlin](./languages.md#kotlin) | 7.8 | 1.37 | 100% |
| [html](./languages.md#html) | 8.04 | 1.46 | 100% |
| [docker](./languages.md#docker) | 8.53 | 1.6 | 100% |
| [yaml](./languages.md#yaml) | 9.68 | 2.6 | 100% |
| [csharp](./languages.md#csharp) | 10.3 | 1.35 | 100% |
| [sql](./languages.md#sql) | 10.8 | 0.423 | 100% |
| [lua](./languages.md#lua) | 11.2 | 1.63 | 100% |
| [toml](./languages.md#toml) | 13.7 | 2.29 | 100% |
| [json](./languages.md#json) | 15.1 | 2.35 | 100% |
| [xml](./languages.md#xml) | 19.2 | 6.57 | 100% |
| [ini](./languages.md#ini) | 20.3 | 4.69 | 100% |
| [diff](./languages.md#diff) | 55 | 14.8 | 100% |

`just lexis diff go` lists the text the two disagree on in a language, the commonest first.
