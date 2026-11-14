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
    func(e *quire.Engine) error {
      _, err := e.AddCollection(model.Basic{ID: "posts"})
      return err
    })
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

To ship, `go run . bake`, then `go build -tags quire_nobake -overlay .quire/ship.json` ([how](./site/content/docs/getting-started.md#ship-it)).
The result is one binary that carries the whole site and none of the image or diagram tooling.

## Examples

Each example is a complete site, commented part by part, and each adds to the one before.
`quire new` starts a site of your own from the one nearest what you want:

```sh
go run github.com/lahickey/quire/cmd/quire@latest new --from 3-themed mysite
cd mysite && go run .
```

The copy has its own `go.mod`, a Dockerfile and a justfile (`--no-just` leaves that out). Without `--from` it is `1-minimal`.

1. [`1-minimal`](./examples/1-minimal) — the site above.
2. [`2-pictures`](./examples/2-pictures) — pictures and a diagram, from the markdown alone.
3. [`3-themed`](./examples/3-themed) — a look of its own: home page, tags, a markdown widget, palette, stylesheets, icon and footer.
4. [`4-plugin`](./examples/4-plugin) — a shop: its own frontmatter and templates, and a basket plugin with sessions and live updates.

## Documentation

- [Getting started](./site/content/docs/getting-started.md)
- [Content](./site/content/docs/content.md)
- [Images](./site/content/docs/images.md)
- [Baking](./site/content/docs/bake.md)
- [The look](./site/content/docs/look.md)
- [Wiring](./site/content/docs/wiring.md)
- [Plugins](./site/content/docs/plugins.md)
- [Design](./site/content/docs/architecture.md)

Working on Quire itself is in [DEVELOPING.md](./DEVELOPING.md).

## Status

> [!WARNING]
> Quire is very early and things may well change dramatically. No guarantees of anything before 1.0.

## Licence

MIT; see [LICENSE](./LICENSE).
What Quire bundles from others, and under which licences, is in [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md).
