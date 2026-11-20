# Third-party notices

Quire is MIT licensed; see [LICENSE](./LICENSE).
It carries the following, each with its licence beside it.

| Component | Where | Licence |
|-|-|-|
| Datastar 1.x | `internal/assets/datastar.js` | MIT, `LICENSE-datastar.txt` |
| Radix Colors 3.0.0 | `internal/radix/radix.go` | MIT, `LICENSE-radix.txt` |
| Inter | `internal/fonts/web`, `internal/fonts/measure/fonts` | OFL 1.1, `LICENSE-inter.txt` |
| JetBrains Mono | `internal/fonts/web`, `internal/fonts/measure/fonts` | OFL 1.1, `LICENSE-jetbrains-mono.txt` |
| Chroma's lexer rules | `libs/lexis/langs/*.xml` | MIT, `LICENSE-chroma.txt` |
| Pygments' lexer rules | The same files | BSD 2-Clause, `LICENSE-pygments.txt` |
| GitHub and LinkedIn marks, from Simple Icons | `ui/components.templ` | CC0 1.0; the marks are their owners' trademarks |

`datastar.js` carries its version and notice in its first line, so they travel with every site that serves it.
The lexer rules are from [Chroma](https://github.com/alecthomas/chroma), which took them from [Pygments](https://pygments.org); some are changed for Lexis, as the note at the top of each file says.
They are compiled into any program that imports `libs/lexis/langs`, which then carries both notices.
Go dependencies are fetched by `go mod`, each with its own licence.
