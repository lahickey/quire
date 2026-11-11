---
title: Wiring
summary: How a site is put together in Go - the site, its options, the mount and its routes.
---

A Quire site is one call:

```go
quire.Run(site, mount, options...)
```

- The **site** is data: what the site is.
- The **options** change how the engine behaves and looks.
- The **mount** is a function that says what the site holds.

## Site

`model.Site` is what the site is:

- `Name` is the publication. The header shows it as the link home; see `WithBrand`.
- `Author` is who writes it.
- `Description` is the default page description.
- `BaseURL` is the production address. Setting it enables `/robots.txt` and `/sitemap.xml`.
- `Dates` is the layouts dates render in.

How the site looks is options, not `Site`.

## Options

An `Option` changes engine behaviour.

- `WithTheme` is the colour scheme when the visitor has chosen none and the OS states none. The default is `ThemeLight`, and the visitor's own choice always wins.
- `WithPalette`, `WithScale` and `WithWidest` set the colours and the page grid. See [The look](./look.md).
- `WithStylesheets` and `WithStylesheetsAt` add the site's CSS.
- `WithBrand(name)` is the name the header shows as the link home, in place of `Site.Name`. `WithBrand("")` shows none, and the nav starts with a Home link instead.
- `WithFooter(component)` replaces the footer.
- `WithNavEnd(components…)` adds controls to the end of Quire's nav, before any a plugin adds.
- `WithHeader(func(Chrome, PageSpec) templ.Component)` replaces the whole header, nav included. It is handed the Chrome and the page, so it can still draw `ui.Nav` (active against `PageSpec.Path`), the controls in `NavEnd` and `ui.ThemeToggle`.
- `WithIcons(fsys, names…)` serves each file in `fsys` at `/{name}` and links it in the head as an icon, typed by its extension and, for a PNG, sized from the file. A name starting `apple-touch-icon` is linked as one.
- `WithSiteCard` changes the share card the bake draws.
- `WithBake` says where the content and the bake's output live. See [Baking](./bake.md).
- `WithMode` and `WithDrafts` are what `--dev` and `--drafts` set.

Datastar is always linked.
`Ship` is not an option: the `ship.go` the bake writes calls it to hand `Run` the embedded bundle.

## The mount

The mount is handed the engine and declares the site on it. It runs at bake and at boot.

```go
func mount(e *quire.Engine) error {
  notes, err := e.AddCollection(model.Basic{ID: "writing", Title: "Writing"})
  if err != nil {
    return err
  }
  e.AddPage("/", homePage(e.Shell, notes))
  e.SetNotFound(notFoundPage(e.Shell))
  return nil
}
```

What it can add:

- `AddCollection(spec)` loads documents and mounts their index and pages. See [Content](./content.md#collections).
- `AddPage(path, component)` mounts a fixed page, rendered once and held. `AddPage("/", …)` replaces the default home.
- `SetNotFound(component)` replaces the default not-found page. Like any fixed page it is rendered once at boot and never handed the request, so every missing URL gets the same bytes and none of them reaches the page cache.
- `AddRoute(method, path, handler)` is the raw mux: nothing added, never cached.
- `AddFile(path, fsys, name)` serves a file directly, so Range still works, and returns its URL with a content hash, which is cached for good. Link to that.
- `Markdown(name)` returns one baked markdown file as a component. See [Content](./content.md#markdown-outside-a-collection).
- `Use(plugins…)` attaches plugins. See [Plugins](./plugins.md).

## Engine, Chrome and Shell

Three names come up in every template.

- The `Engine` is the mount: what the site holds (collections, pages, routes, files, plugins), then the handler that serves it. A page template does not need it.
- `Chrome` is the frame's data: the site, nav, plugin controls, assets, icons and theme. It is complete only once the mount is done. The frame reads it: the document, a custom header, the default home and not-found pages.
- `Shell` (`e.Shell`, `model.Shell`) wraps a page body in the document, reading Chrome when the page renders rather than when it is declared. It is all a page template needs of the engine; everything else it shows is handed to it as data.

## Reserved paths

A leading `_` segment belongs to the engine or to a plugin; `AddPage` refuses one.
Everything Quire serves for itself is under `/_quire/`: `health`, `reload`, `style.css`, `quire.js`, `datastar.js`, `fonts/`, `img/`. Each plugin gets `/_<name>/`.
Datastar signals follow the same split: `$quire` is the engine's, and a plugin's stream patches only `$<name>`.
Only the files that have to be findable at a known location stay at the root: `robots.txt` and `sitemap.xml`.

## In depth

### The handler

`Handler()` finalises: mount everything before you take it. It renders every fixed page, so a template error stops the binary at boot, and anything added after it panics.

An app that needs routes outside Quire entirely can own `main` and mount the engine as a fallback: `mux.Handle("/", e.Handler())`, with its own patterns beside it.
Prefer a plugin.

### Redirects

A trailing slash redirects to the path without one. `www` redirects to the host in `BaseURL`, and only to that host.
Both are sent with `Cache-Control: no-cache`, so a browser cannot pin one.

### Caching and compression

- A fixed page is rendered once, when the handler is taken.
- A collection's document page is rendered on first request and held in a 4MB page cache, keyed by its path.
- A raw route goes straight to its handler and is never cached.
- An asset linked with its content hash (`?h=`) is cached by the browser for good. Anything else is revalidated.

A cached body is compressed once, as brotli or gzip, and stored that way. A body under 256 bytes is sent as it is.
`--dev` sends pages `no-store`.

### Scripts

`/_quire/quire.js` holds the stored theme, soft navigation and any plugin scripts. It is a classic script loaded in the head, so the theme is set before the first paint.
Datastar is an ES module linked beside it.
Soft navigation replaces `main` and nothing else, so the header and anything in it lasts for the life of the tab.
