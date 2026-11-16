# Quire

A Go engine that turns a folder of markdown into a finished website, served from a single binary.
Content in, complete site out — looks good by default yet every layer of it can be replaced.

- **One binary.** Words, pictures and diagrams are baked into the site's own binary.
- **Pictures handled end to end.** Refer to a picture beside a document and get a responsive ladder in WebP.
- **Server-rendered.** HTML from [templ](https://templ.guide); live features are plugins streaming through [Datastar](https://data-star.dev).

## Getting started

Quire needs Go 1.27 or later.

```sh
go mod init example.com/site
go get github.com/lahickey/quire
```

`main.go`:

```go
package main

import (
  "github.com/lahickey/quire"
  "github.com/lahickey/quire/model"
)

func main() {
  quire.Run(model.Site{Name: "Example", BaseURL: "https://example.com"},
    quire.MountFunc(func(e *quire.Engine) error {
      _, err := e.AddCollection(model.Basic{ID: "posts"})
      return err
    }))
}
```

`content/posts/hello.md`:

```markdown
---
title: Hello
date: 2026-02-01
---

Some words.
```

`go run .` serves it on http://localhost:8090: home page, post index, post pages, navigation, a light and dark toggle, sitemap and robots.txt, with no templates or CSS of your own.

To ship, `go run . bake`, then `go build -tags quire_nobake -overlay .quire/ship.json` ([how](https://quire.hiddenbridge.io/docs/getting-started#ship-it)).
The result is one binary that carries the whole site and none of the image or diagram tooling.

## Examples

Each example is a complete site, commented part by part, and each adds to the one before.
`quire new` starts a site of your own from the one nearest what you want:

```sh
go run github.com/lahickey/quire/cmd/quire@latest new --from themed mysite
cd mysite && go run .
```

The copy has its own `go.mod`, a Dockerfile and a justfile (`--no-just` leaves that out). `--from` takes the name with or without its number; without it the copy is `1-minimal`.

1. [`1-minimal`](./examples/1-minimal) — the site above.
2. [`2-pictures`](./examples/2-pictures) — pictures and a diagram, from the markdown alone.
3. [`3-themed`](./examples/3-themed) — a look of its own: home page, tags, a markdown widget, palette, stylesheets, icon and footer.
4. [`4-plugin`](./examples/4-plugin) — a shop: its own frontmatter and templates, and a basket plugin with sessions and live updates.
5. [`5-docs`](./examples/5-docs) — a documentation site from the docs pack: a landing page, a sidebar and search, with no templates or CSS.

## Documentation

The documentation is at [quire.hiddenbridge.io](https://quire.hiddenbridge.io).
That site is itself built with Quire; its source is in [`site/`](./site).

- [Getting started](https://quire.hiddenbridge.io/docs/getting-started)
- [A blog](https://quire.hiddenbridge.io/docs/blog)
- [Content](https://quire.hiddenbridge.io/docs/content)
- [Images](https://quire.hiddenbridge.io/docs/images)
- [The look](https://quire.hiddenbridge.io/docs/look)
- [Wiring](https://quire.hiddenbridge.io/docs/wiring)
- [Plugins](https://quire.hiddenbridge.io/docs/plugins)
- [Packs](https://quire.hiddenbridge.io/docs/packs)
- [Baking](https://quire.hiddenbridge.io/docs/bake)
- [Deploying](https://quire.hiddenbridge.io/docs/deploy)
- [Lexis](https://quire.hiddenbridge.io/docs/lexis), the syntax highlighter, and [its benchmarks](./site/content/docs/benchmarks.md)
- [Design](https://quire.hiddenbridge.io/docs/architecture)

Working on Quire itself is in [DEVELOPING.md](./DEVELOPING.md).

## Status

> [!WARNING]
> Quire is very early and things may well change dramatically. No guarantees of anything before 1.0.

## Licence

MIT; see [LICENSE](./LICENSE).
What Quire bundles from others, and under which licences, is in [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md).
