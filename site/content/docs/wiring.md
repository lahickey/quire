---
title: Wiring
summary: How a site is put together in Go - the site, its options, the mount and its routes.
---

A Quire site is one call:

```go
quire.Run(site, part, options...)
```

- The **site** is data: what the site is.
- The **part** says what the site holds. It is your own Go or a [pack](./packs.md), and has up to two steps: [the mount](#the-mount) and [Live](#live).
- The **options** change how the engine behaves and looks.

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
- `WithPalette` and `WithScale` set the colours and the page grid. See [The look](./look.md).
- `WithStylesheets` and `WithStylesheetsAt` add the site's CSS.
- `WithBrand(name)` is the name the header shows as the link home, in place of `Site.Name`. `WithBrand("")` shows none, and the nav starts with a Home link instead.
- `WithFooterText(text)` replaces the footer's line, which is otherwise the year and `Site.Author`, or `Site.Name` when there is no author.
- `WithFooterLinks(links…)` adds links to the footer, each drawn as its mark alone when it has one. See [Links](./look.md#links). `model.BuiltWithQuire` is one ready made.
- `WithFooter(component)` replaces the whole footer.
- `WithNavEnd(components…)` adds controls to the end of Quire's nav, before any a plugin adds.
- `WithHeader(func(Chrome, PageSpec) templ.Component)` replaces the whole header, nav included. It is handed the Chrome and the page, so it can still draw `ui.Nav` (active against `PageSpec.Path`), the controls in `NavEnd`, `ui.PalettePicker` and `ui.ThemeToggle`.
- `WithIcons(fsys, names…)` serves each file in `fsys` at `/{name}` and links it in the head as an icon, typed by its extension. A name starting `apple-touch-icon` is linked as one.
- `WithSiteCard` changes the share card the bake draws.
- `WithBake` says where the content and the bake's output live. See [Baking](./bake.md).
- `WithDev` and `WithDrafts` are what `--dev` and `--drafts` set.

Datastar is always linked.
`Ship` is not an option: the `ship.go` the bake writes calls it to hand `Run` the embedded bundle.

## The mount

A part's `Mount` is handed the engine and declares the site on it. It runs at bake and at boot, so it only declares: work belongs in [Live](#live).

A site with nothing but a mount writes it as a function, and `quire.MountFunc` makes that a part:

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

func main() {
  quire.Run(site, quire.MountFunc(mount))
}
```

What it can add:

- `AddCollection(spec)` loads documents and mounts their index and pages. See [Content](./content.md#collections).
- `AddPage(path, component)` mounts a fixed page, rendered once and held. `AddPage("/", …)` replaces the default home. A nil component panics.
- `SetNotFound(component)` replaces the default not-found page. Like any fixed page it is rendered once at boot and never handed the request, so every missing URL gets the same bytes and none of them reaches the page cache.
- `AddFile(path, fsys, name)` serves a file directly, so Range still works, and returns its URL with a content hash, which is cached for good. Link to that.
- `Markdown(name)` returns one baked markdown file as a component, and `Document[T](name)` returns it with its frontmatter. See [Content](./content.md#markdown-outside-a-collection).
- `AddStylesheets` and `AddStylesheetsAt` add CSS in the `pack` layer, beneath the site's own sheets.
- `SetFooter(component)`, `SetFooterText(text)`, `AddFooterLinks(links…)` and `AddNavEnd(components…)` do what the options of those names do.
- `Site()` is the site the engine was made with.
A `quire.Part` is anything with a `Mount` method of that signature, so a type of your own is one too, and so is a pack: `quire.Run(site, &docs.Pack{})`.

## Live

A part that also has a `Live` method adds what only a served site has. A bake never calls it, so this is where a plugin is attached and where anything slow or outside the content belongs: an index, a query, a connection.
A function cannot have a second method, so a site with a live step is a type:

```go
type blog struct{ notes *model.Collection[model.Meta] }

func (b *blog) Mount(e *quire.Engine) error {
  var err error
  b.notes, err = e.AddCollection(model.Basic{ID: "writing"})
  return err
}

func (b *blog) Live(l *quire.Live) error {
  l.AddRoute("POST", "/subscribe", subscribe)
  s := search.New()
  s.Add(b.notes)
  return l.Use(s)
}

func main() {
  quire.Run(site, &blog{})
}
```

`Live` runs after `Mount` on the same value, so what `Mount` loaded is there to use.

- `Use(plugins…)` attaches plugins. See [Plugins](./plugins.md).
- `AddRoute(method, path, handler)` is the raw mux: nothing added, never cached.

Nothing can be added once the mount is over: a call after the handler is taken panics.
A part is a value, so one can be shipped as a package. That is a [pack](./packs.md).

## Engine, Chrome and Shell

Three names come up in every template.

- The `Engine` is the mount: what the site holds (collections, pages, files), then the handler that serves it. Routes and plugins are added on `Live`. A page template does not need it.
- `Chrome` is the frame's data: the site, nav, plugin controls, assets, icons and theme. It is complete only once the mount is done. The frame reads it: the document, a custom header, the default home and not-found pages.
- `Shell` (`e.Shell`, `model.Shell`) wraps a page body in the document, reading Chrome when the page renders rather than when it is declared. It is all a page template needs of the engine; everything else it shows is handed to it as data.

## The head

A page hands the shell a `model.PageSpec`, and the shell writes the head from it.

| Field | Does |
|-|-|
| `Title` | The title, and the share title |
| `Description` | The description; `Site.Description` when empty |
| `Path` | The page's address, for the canonical URL |
| `NoIndex` | Keeps the page out of search engines |

- **Canonical.** A page with a `Path` gets `<link rel="canonical">` and `og:url`, made from `BaseURL`. A page with no `Path`, a `NoIndex` page and a draft get `<meta name="robots" content="noindex">` instead.
- **Sharing.** Every page gets Open Graph and Twitter card tags: title, description, site name and a large image.
- **The share image** is `ShareImage` when set, else the site's share card, which the bake draws. A document with an `image` shares that picture. See [Images](./images.md#share-images).
- **Articles.** `Article` marks a page as one: `og:type` is `article` and its published and modified dates are written in ISO form.
- **Author.** `Site.Author` becomes `<meta name="author">`.

`Collection.IndexPage()` and `Collection.ItemPage(d)` fill a `PageSpec` for a collection's pages, so a template of your own rarely builds one by hand.
The sitemap lists every fixed page and every published document, with a document's `updated` or `date` as its last change.

## Dev mode

`--dev`, or `WithDev(true)`, is for writing, not for serving visitors.

- **Live reload.** The page reloads when the binary restarts. Restarting on a change is a watcher's job: the `dev` recipe in a site's justfile runs `wgo`.
- **Nothing is cached.** Every response is `no-store`, pictures included, and the page cache is off, so a reload always shows the current state.
- **No `www` redirect, and no secure cookie**, so the site works on `localhost`.

Drafts are a separate flag, `--drafts`, so a draft can be checked with production's caching too.

## Reserved paths

A leading `_` segment is a namespace with one owner: `/_quire/` is the engine's, `/_<name>/` is a plugin's and `/_site/` is the site's own.
Every route is checked the same way, whoever adds it. `AddPage`, `AddRoute` and `AddFile` panic on a path in another owner's namespace or on one already mounted, and a plugin's `Use` returns the same as an error.
Everything Quire serves for itself is under `/_quire/`: `health`, `reload`, `style.css`, `quire.js`, `datastar.js`, `fonts/`, `img/`.
Datastar signals follow the same split: `$quire` is the engine's, and a plugin's stream patches only `$<name>`.
Only the files that have to be findable at a known location stay at the root: `robots.txt` and `sitemap.xml`.

## In depth

### The handler

`Handler()` finalises. It renders every fixed page, so a template error stops the binary at boot.

> [!IMPORTANT]
> Mount everything before you take the handler. Anything added after it panics.

A site with routes of its own adds them in [Live](#live) with `AddRoute`, or as a plugin. Both go through the engine's route claims, so a clash is an error at boot.

`quire.New(site, options...)`, `e.Mount(part)` and `e.Handler()` are for a test: they give a handler for a part or a plugin with no bundle behind it. Only `Run` bakes and loads one.

### Redirects

A trailing slash redirects to the path without one. `www` redirects to the host in `BaseURL`, and only to that host.
Both are sent with `Cache-Control: no-cache`, so a browser cannot pin one.

### Caching and compression

- A fixed page is rendered once, when the handler is taken.
- A collection's document page is rendered on first request and held in a 4MB page cache, keyed by its path.
- A raw route goes straight to its handler and is never cached.
- `robots.txt` and `sitemap.xml` are built once, when the handler is taken.
- An asset linked with its content hash (`?h=`) is cached by the browser for good. Anything else is revalidated.

A cached body is compressed once, as brotli or gzip, and stored that way. A body under 256 bytes is sent as it is.
`--dev` sends pages `no-store`.

### Scripts

`/_quire/quire.js` holds the stored theme, soft navigation and any plugin scripts. It is a classic script loaded in the head, so the theme is set before the first paint.
Datastar is an ES module linked beside it.
Soft navigation replaces `main` and nothing else, so the header and anything in it lasts for the life of the tab.

A page can keep more than the header. Put `data-region="name"` on the element inside `main` that holds what changes:

```html
<main>
  <aside>a sidebar</aside>
  <article data-region="docs">the page</article>
</main>
```

Between two pages that both have that region, only the region is replaced, and everything round it stays as it was: its scroll position, its open panels, anything a plugin is streaming into it.
Between any other two pages `main` is replaced, as before.
A link outside what was replaced takes its `class` and `aria-current` from the same link in the new page, so the header and a sidebar both show where the reader now is.
The sidebar on this site is kept that way: follow a link in it and only the page beside it changes.

While the next page is on its way, `html` carries `data-navigating`. Quire's own sheet draws a thin bar across the top after a moment, so a quick page shows nothing and a slow one shows the click was taken. When the page arrives the bar runs to the end and fades, with `data-navigating="done"` for that moment. A second click on the link already being fetched does nothing, and a click on another link drops the first.
Once the new page is in, `quirenavigate` fires on `window`. Anything kept across the swap can listen for it, as this site's sidebar does to close its menu on a narrow screen:

```html
<aside data-on:quirenavigate__window="$site.menu = false">
```
