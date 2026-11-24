---
kicker: Composable parts for static and live sites in Go
title: A folder of markdown in. A finished website out.
lede: Quire bakes your words, pictures and diagrams into one binary and serves them. It looks good by default, and every layer of it can be replaced.
install: go get github.com/lahickey/quire
repo: https://github.com/lahickey/quire
footer: Quire is open source, under the MIT licence.
docs:
  summary: How to build a site with Quire, and why it is shaped the way it is.
  sections:
    - title: Start here
      pages: [getting-started, examples, blog]
    - title: Authoring
      pages: [content, pictures, look]
    - title: Building
      pages: [wiring, features, services, packs, parts, bake, deploy]
    - title: Libraries
      pages: [lexis, languages, benchmarks]
    - title: Background
      pages: [architecture]
---

```cards
- href: /docs/bake
  kicker: Shipping
  title: One binary
  description: Words, pictures and diagrams are baked into the site's own binary. It serves from any directory and carries none of the tooling that made it.
- href: /docs/pictures
  kicker: Pictures
  title: Pictures, end to end
  description: Refer to a picture beside a document and get a responsive ladder in WebP, sized for where it is placed.
- href: /docs/content#diagrams
  kicker: Diagrams
  title: Mermaid, drawn once
  description: A mermaid fence is drawn by the bake and inlined as SVG, so it takes the site's palette and fonts.
- href: /docs/look
  kicker: The look
  title: Good by default
  description: Palette, fonts, a light and dark toggle, navigation and a sitemap, with no templates or CSS of your own.
- href: /docs/packs
  kicker: Packs
  title: A docs site, ready made
  description: A landing page, a sidebar and search from a folder of markdown. This site is one, and its Go is one short file.
- href: /docs/features
  kicker: Features
  title: Live when it needs to be
  description: Pages are server-rendered HTML from templ. Live features stream through Datastar.
```

## This is a whole site

Home page, post index, post pages, navigation, a colour-scheme toggle, sitemap and robots.txt. Add markdown to `content/posts` and run it. [The smallest site](./docs/getting-started.md) builds it from an empty directory.

```go
package main

import (
  "github.com/lahickey/quire"
  "github.com/lahickey/quire/model"
  "github.com/lahickey/quire/pack/base"
)

func main() {
  quire.Run(model.Identity{Name: "Example", BaseURL: "https://example.com"}, base.Content())
}
```

## You are looking at one

This site is built with Quire, from the [docs pack](./docs/packs.md) and a folder of markdown.
Its [source](https://github.com/lahickey/quire/tree/master/www) is one short Go file, a stylesheet and `content/`.
Each of the [examples](./docs/examples.md) runs beneath it, from the same code and content as in the repository.
