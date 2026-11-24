Rules for coding agents working on Quire.

## Read first

- [DEVELOPING.md](DEVELOPING.md): the tools, how to run, test and release.
- [Design](www/content/docs/architecture.md): why Quire is shaped as it is, and the decisions it keeps to.
- [TODO.md](TODO.md): what is planned.
- The page in `www/content/docs/` for whatever you are changing.

## Where things are

| Path | Holds |
|-|-|
| `*.go` at the root | The helpers: `Run` |
| `site/`, `server/` | What a feature declares on; what serves it |
| `model/` | Data |
| `feature/` | Features with no look, each handed one: content, search, reload |
| `service/` | Services Quire ships: a store, sessions |
| `bundle/` | Opens what a bake left |
| `pack/` | What has a look, shipped together: `pack/docs` |
| `pack/base/` | The default look: `ui`, stylesheets, fonts, widgets |
| `libs/` | Packages for use outside Quire: `libs/lexis`, `libs/streams` |
| `bake/` | The baker: cache, keys, journal |
| `pack/base/convert/` | The converters: markdown, mermaid, card, picture |
| `examples/` | Complete sites, and what `quire new` copies |
| `www/` | The documentation site, built with `pack/docs` |

A package under `libs/` imports nothing else of Quire's.

## A change is not done until

- `just test` passes. It regenerates templ, vets, lints and runs every example through `quire new`.
- **It is documented and demonstrated.** A feature a site can use gets a section on its page in `www/content/docs/`, and that section shows the feature working on the page, not only its source: the rendered callout beside the markdown for it, the picture beside the line that places it. What this site cannot draw is shown as a screenshot of an example, retaken by `www/shots.sh`.
- A new page is listed in `docs.sections` in `www/content/index.md`, or the site will not load.
- A feature of the default look is in an example, with a check in `examples/examples_test.go`.
- A converter whose output changed has `Version` bumped in its package under `pack/base/convert/`; `just release-check` says which.

## Code and docs

- Comments are the absolute minimum.
- No history in code or docs; git has it.
- Docs: one sentence per line where it reads well, short tables only, no tabs.
