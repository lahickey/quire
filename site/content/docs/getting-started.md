---
title: Getting started
summary: The smallest site, running it, and shipping it.
---

## The smallest site

```go
func main() {
	quire.Run(model.Site{Name: "Example", BaseURL: "https://example.com"},
		func(e *quire.Engine) error {
			_, err := e.AddCollection(model.Basic{ID: "posts"})
			return err
		})
}
```

That is a working site: home, post index, post pages, navigation, colour-scheme toggle, sitemap, robots.
No extra templates, no extra CSS, no custom types: `model.Basic` is a collection of stock frontmatter.
`examples/1-minimal/` is this. `examples/2-pictures/` is the same code with pictures and a diagram in its content, `examples/3-themed/` a site with a look of its own, and `examples/4-plugin/` a shop with a basket plugin.

## Running it

`Run` is the binary: `--dev` (live reload, `no-store`), `--drafts`, `--addr` (default `:8090`), and the argument `bake` to bake and exit.
Run it from the site's directory. It bakes `content/` into `.quire`, then serves the bundle off disk; the function it is given is the mount.
Everything under `.quire` is made by the bake, so deleting it is a clean start.

## Shipping it

The shipped binary carries its bundle. Add this file beside `main.go`:

```go
//go:build quire_embed

package main

import (
	"embed"

	"github.com/lahickey/quire"
)

//go:embed all:.quire/bundle
var bundle embed.FS

func init() { quire.Ship(bundle) }
```

Then `go run . bake` and `go build -tags quire_nobake,quire_embed`.
That binary serves what it embeds from any directory, links no converter, and refuses to bake.
Only a `quire_embed` build embeds the bundle, so nothing needs `.quire` to exist before the first bake.
See [Baking](./bake.md).
