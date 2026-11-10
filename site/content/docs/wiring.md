---
title: Wiring
summary: Site, options, routes and reserved paths.
---

## Engine, Chrome and Shell

- The `Engine` is the mount: what the site holds (collections, pages, routes, files, plugins), then the handler that serves it. A page template does not need it.
- `Chrome` is the frame's data — the site, nav, plugin controls, assets, icons, theme — complete only once the mount is done. The frame reads it: the document, a custom header, the default home and not-found pages.
- `Shell` (`e.Shell`, `model.Shell`) wraps a page body in the document, reading Chrome when the page renders rather than when it is declared. It is all a page template needs of the engine; everything else it shows is handed to it as data.

## Site

`model.Site` is what the site is: `Name` (the publication), `Author` (who writes it), `Description`, `BaseURL`, `Dates`.
`Site.BaseURL` enables `/robots.txt` and `/sitemap.xml`.
How it looks is options.

## Options

An `Option` changes engine behaviour: `WithMode` `WithDrafts` `WithTheme` `WithPalette` `WithStylesheets` `WithStylesheetsAt` `WithHeader` `WithFooter` `WithNavEnd` `WithBake` `WithSiteCard` `WithScale` `WithWidest` `WithIcons`.
`WithTheme` is the colour scheme when the visitor has chosen none and the OS states none; default `ThemeLight`, and the visitor's own choice always wins.
`WithFooter(component)` replaces the footer. `WithNavEnd(components…)` adds controls to the end of Quire's nav, before any a plugin adds.
`WithHeader(func(Chrome, PageSpec) templ.Component)` replaces the whole header, nav included; it is handed the Chrome and the page, so it can still draw `Nav` (active against `PageSpec.Path`) and the plugins' `Controls`.
`WithIcons(fsys, names…)` serves each file in `fsys` at `/{name}` and links it in the head as an icon, typed by its extension and, for a PNG, sized from the file; a name starting `apple-touch-icon` is linked as one.
`Ship` is not an option: the `ship.go` the bake writes calls it to hand `Run` the embedded bundle.

Datastar is always linked.

## Routes

`AddPage(path, component)` mounts a fixed page, rendered once and held; `AddPage("/", …)` replaces the default home.
`SetNotFound(component)` replaces the default not-found page. Like any fixed page it is rendered once at boot and never handed the request, so every missing URL gets the same bytes and none of them reaches the page cache.
`AddRoute(method, path, handler)` is the raw mux: nothing added, never cached.
`AddFile(path, fsys, name)` serves a file the same way (direct, so Range still works) and returns its URL with a content hash, which is cached for good; link to that.

`Handler()` finalises — mount everything before you take it. It renders every fixed page, so a template error stops the binary at boot, and anything added after it panics.
A trailing slash redirects to the path without one. `www` redirects to the host in `BaseURL`.
An app that needs routes outside Quire entirely can own `main` and mount the engine as a fallback: `mux.Handle("/", e.Handler())`, its own patterns beside it.
Prefer a plugin.

## Reserved paths

A leading `_` segment belongs to the engine or to a plugin; `AddPage` refuses one.
Everything Quire serves for itself is under `/_quire/` — `health`, `reload`, `style.css`, `quire.js`, `datastar.js`, `fonts/`, `img/` — and each plugin gets `/_<name>/`.
Datastar signals follow the same split: `$quire` is the engine's, and a plugin's stream patches only `$<name>`.
Only the files that have to be findable at a known location stay at the root: `robots.txt` and `sitemap.xml`.
