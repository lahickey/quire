Rules for coding agents working on Quire.

## Read first

- [DEVELOPING.md](DEVELOPING.md): the tools, how to run, test and release.
- [Design](site/content/docs/architecture.md): why Quire is shaped as it is, and the decisions it keeps to.
- [TODO.md](TODO.md): what is planned.
- The page in `site/content/docs/` for whatever you are changing.

## Where things are

| Path | Holds |
|-|-|
| `*.go` at the root | The engine a site calls |
| `model/`, `ui/`, `plugin/` | Data, components, the plugin API |
| `pack/` | Whole kinds of site, such as `pack/docs` |
| `libs/` | Packages for use outside Quire, such as `libs/lexis` |
| `internal/bake/` | The converters: markdown, picture, diagram, card |
| `examples/` | Complete sites, and what `quire new` copies |
| `site/` | The documentation site, built with `pack/docs` |

A package under `libs/` imports nothing else of Quire's.

## A change is not done until

- `just test` passes. It regenerates templ, vets, lints and runs every example through `quire new`.
- **It is documented and demonstrated.** A feature a site can use gets a section on its page in `site/content/docs/`, and that section shows the feature working on the page, not only its source: the rendered callout beside the markdown for it, the picture beside the line that places it. What this site cannot draw is shown as a screenshot of an example, retaken by `site/shots.sh`.
- A new page is listed in `docs.sections` in `site/content/index.md`, or the site will not load.
- A feature of the default look is in an example, with a check in `examples/examples_test.go`.
- A converter whose output changed has its version bumped in `internal/bake/versions.go`; `just release-check` says which.

## Code and docs

- Comments are the absolute minimum.
- No history in code or docs; git has it.
- Docs: one sentence per line where it reads well, short tables only, no tabs.
