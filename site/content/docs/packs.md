---
title: Packs
summary: A whole kind of site, ready made. The docs pack is this one.
---

A pack is a site's Go somebody else wrote: collections, pages, styles and plugins for one kind of site.
It is a blueprint, as a site's own is, so a site hands it to `Run` or includes it in its own, and writes content.

## The docs pack

`pack/docs` makes a documentation site: a landing page, the documents beside a sidebar, and search.
This site is the whole of its Go:

```go
package main

import (
  "github.com/lahickey/quire/app"
  "github.com/lahickey/quire/model"
  "github.com/lahickey/quire/pack/docs"
)

func main() {
  app.Run(model.Identity{Name: "Quire", BaseURL: "https://quire.hiddenbridge.io"}, &docs.Pack{})
}
```

Everything else is in `content/`.

`quire new --from docs mysite` starts one. See [Getting started](./getting-started.md#start-from-an-example).

### The landing page

`content/index.md` is the landing page and the site's settings. Every field is optional.

```yaml
---
kicker: Composable parts for static and live sites in Go
title: A folder of markdown in. A finished website out.
lede: One or two sentences under the title.
install: go get github.com/lahickey/quire
repo: https://github.com/lahickey/quire
footer: Quire is MIT licensed.
docs:
  title: Documentation
  summary: The line under the heading of /docs.
  sections:
    - title: Start here
      pages: [getting-started]
    - title: Authoring
      pages: [content, pictures, look]
---
```

- `title` is the headline. Without it the headline is the site's name.
- `install` is drawn as a shell command.
- `repo` puts a Source link in the nav, beside the two buttons and in the footer. Each is a [`ui.Link`](./look.md#links), so a repository on GitHub gets its mark.
- `footer` replaces the footer's text. Without it the footer is Quire's.
- `docs.sections` is the sidebar. See below.

The body of the file is drawn below the hero, as any document is.
The cards on a landing page are a [`cards` fence](./content.md#cards) in that body, so a picture or a recording goes above them by being written first.
[This site's home page](/) is one.

The two buttons go to the first document and to `/docs`.

### The sidebar

A document in `content/docs` has a `title` and a `summary`, and says nothing about where it sits.
The sidebar is `docs.sections` in `index.md`: each section's `title`, and its `pages` by file name.
The order written there is the order of the sidebar, the index and the pager, so moving a page is an edit to one line.

A document that no section lists fails the load, and so does a name with no document. A section with no `title` has no heading.

### Replacing a page

`Pack` has a field for each page, and nil is the default of the same name:

```go
&docs.Pack{Item: myItem}
```

`Home`, `Index` and `Item` are the pages, and `docs.Sidebar` is there for one that still wants it.
Each draws a body, and is handed a `*docs.Book`: the collection of documents, with `Section(d)` for the section `index.md` puts a document in.

### Widgets and CSS

The documents take the site's [widgets](./content.md#widgets), added with `site.SetWidget` as for any site, before the pack is included.

The pack's CSS is in the `pack` layer, so the site's own sheets win over it.
The page beside the sidebar is a [region](./wiring.md#scripts), so the sidebar is not redrawn when a link in it is followed. A page of your own keeps that by putting `data-region="docs"` on its content, as the defaults do.
