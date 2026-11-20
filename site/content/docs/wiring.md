---
title: Wiring
summary: How a site is put together in Go - its identity, its options, the blueprint and its routes.
---

A Quire site is one call:

```go
app.Run(identity, blueprint, options...)
```

- The **identity** is data: what the site is.
- The **blueprint** says what the site holds. It is your own Go or a [pack](./packs.md), and has up to two steps: [Declare](#the-blueprint) and [Live](#live).
- The **options** change how Quire behaves and looks.

`Run` is in package `app`, the helpers. Everything else on this page is package `quire`.

## Identity

`model.Identity` is what the site is:

- `Name` is the publication. The header shows it as the link home; see `WithBrand`.
- `Author` is who writes it.
- `Description` is the default page description.
- `BaseURL` is the production address. Setting it enables `/robots.txt` and `/sitemap.xml`.
- `Dates` is the layouts dates render in.

How the site looks is options, not `Identity`.

## Options

An `Option` changes how Quire behaves.

- `WithTheme` is the colour scheme when the visitor has chosen none and the OS states none. The default is `ThemeLight`, and the visitor's own choice always wins.
- `WithPalette` and `WithScale` set the colours and the page grid. See [The look](./look.md).
- `WithStylesheets` and `WithStylesheetsAt` add the site's CSS.
- `WithBrand(name)` is the name the header shows as the link home, in place of `Identity.Name`. `WithBrand("")` shows none, and the nav starts with a Home link instead.
- `WithFooterText(text)` replaces the footer's line, which is otherwise the year and `Identity.Author`, or `Identity.Name` when there is no author.
- `WithFooterLinks(links…)` adds links to the footer, each drawn as its mark alone when it has one. See [Links](./look.md#links). `model.BuiltWithQuire` is one ready made.
- `WithFooter(component)` replaces the whole footer.
- `WithNavEnd(components…)` adds controls to the end of Quire's nav, before any a plugin adds.
- `WithHeader(func(Chrome, PageSpec) templ.Component)` replaces the whole header, nav included. It is handed the Chrome and the page, so it can still draw `ui.Nav` (active against `PageSpec.Path`), the controls in `NavEnd`, `ui.PalettePicker` and `ui.ThemeToggle`.
- `WithIcons(fsys, names…)` serves each file in `fsys` at `/{name}` and links it in the head as an icon, typed by its extension. A name starting `apple-touch-icon` is linked as one.
- `WithWidgets` gives the site its [widgets](./content.md#widgets).
- `WithSiteCard` changes the share card the bake draws.
- `WithDev` and `WithDrafts` are what `--dev` and `--drafts` set.

Datastar is always linked.
`Ship` is not an option: the `ship.go` the bake writes calls it to hand `Run` the embedded bundle.

## The blueprint

A blueprint's `Declare` is handed the site and says what it holds. It runs at bake and at boot, so it only declares: work belongs in [Live](#live).

A site with nothing but that writes it as a function, and `quire.BlueprintFunc` makes it a blueprint:

```go
func declare(site *quire.Site) error {
  notes, err := site.AddCollection(model.Basic{ID: "writing", Title: "Writing"})
  if err != nil {
    return err
  }
  site.AddPage("/", homePage(site.Shell, notes))
  site.SetNotFound(notFoundPage(site.Shell))
  return nil
}

func main() {
  app.Run(identity, quire.BlueprintFunc(declare))
}
```

What it can add:

- `AddCollection(spec)` loads documents and adds their index and pages. See [Content](./content.md#collections).
- `AddPage(path, component)` adds a fixed page, rendered once and held. `AddPage("/", …)` replaces the default home. A nil component fails the build.
- `SetNotFound(component)` replaces the default not-found page. Like any fixed page it is rendered once at boot and never handed the request, so every missing URL gets the same bytes and none of them reaches the page cache.
- `AddFile(path, fsys, name)` serves a file directly, so Range still works, and returns its URL with a content hash, which is cached for good. Link to that.
- `Markdown(name)` returns one baked markdown file as a component, and `Document[T](name)` returns it with its frontmatter. See [Content](./content.md#markdown-outside-a-collection).
- `AddStylesheets` and `AddStylesheetsAt` add CSS in the `pack` layer, beneath the site's own sheets.
- `AddWidget(name, widget)` adds a [widget](./content.md#widgets), and has to come before any content is read.
- `SetFooter(component)`, `SetFooterText(text)`, `AddFooterLinks(links…)` and `AddNavEnd(components…)` do what the options of those names do.
- `Identity()` is the identity the site was made with.
- `Include(blueprint)` runs another blueprint on the same site, its `Live` too, so a site can add a pack to pages of its own.

A `quire.Blueprint` is anything with a `Declare` method of that signature, so a type of your own is one too, and so is a pack: `app.Run(identity, &docs.Pack{})`.

## Live

A blueprint that also has a `Live` method adds what only a served site has. A bake never calls it, so this is where a plugin is attached and where anything slow or outside the content belongs: an index, a query, a connection.
A function cannot have a second method, so a site with a live step is a type:

```go
type blog struct{ notes *model.Collection[model.Meta] }

func (b *blog) Declare(site *quire.Site) error {
  var err error
  b.notes, err = site.AddCollection(model.Basic{ID: "writing"})
  return err
}

func (b *blog) Live(l *quire.Live) error {
  l.AddRoute("POST", "/subscribe", subscribe)
  s := search.New()
  s.Add(b.notes)
  return l.Use(s)
}

func main() {
  app.Run(identity, &blog{})
}
```

`Live` runs after `Declare` on the same value, so what `Declare` loaded is there to use.

- `Use(plugins…)` attaches plugins. See [Plugins](./plugins.md).
- `AddRoute(method, path, handler)` is the raw mux: nothing added, never cached.

Nothing can be added once the site is built: a call after that panics.
A blueprint is a value, so one can be shipped as a package. That is a [pack](./packs.md).

## Site, Chrome and Shell

Three names come up in every template.

- The `Site` is what a blueprint declares on: what the site holds (collections, pages, files). Routes and plugins are added on `Live`. A page template does not need it.
- `Chrome` is the frame's data: the identity, nav, plugin controls, assets, icons and theme. It is complete only once the site is built. The frame reads it: the document, a custom header, the default home and not-found pages.
- `Shell` (`site.Shell`, `model.Shell`) wraps a page body in the document, reading Chrome when the page renders rather than when it is declared. It is all a page template needs of the site; everything else it shows is handed to it as data.

## The head

A page hands the shell a `model.PageSpec`, and the shell writes the head from it.

| Field | Does |
|-|-|
| `Title` | The title, and the share title |
| `Description` | The description; `Identity.Description` when empty |
| `Path` | The page's address, for the canonical URL |
| `NoIndex` | Keeps the page out of search engines |

- **Canonical.** A page with a `Path` gets `<link rel="canonical">` and `og:url`, made from `BaseURL`. A page with no `Path`, a `NoIndex` page and a draft get `<meta name="robots" content="noindex">` instead.
- **Sharing.** Every page gets Open Graph and Twitter card tags: title, description, site name and a large image.
- **The share image** is `ShareImage` when set, else the site's share card, which the bake draws. A document with a `picture` shares that picture. See [Pictures](./pictures.md#share-images).
- **Articles.** `Article` marks a page as one: `og:type` is `article` and its published and modified dates are written in ISO form.
- **Author.** `Identity.Author` becomes `<meta name="author">`.

`Collection.IndexPage()` and `Collection.ItemPage(d)` fill a `PageSpec` for a collection's pages, so a template of your own rarely builds one by hand.
The sitemap lists every fixed page and every published document, with a document's `updated` or `date` as its last change.

## Dev mode

`--dev`, or `WithDev(true)`, is for writing, not for serving visitors.

- **Live reload.** The page reloads when the binary restarts. Restarting on a change is a watcher's job: the `dev` recipe in a site's justfile runs `wgo`.
- **Nothing is cached.** Every response is `no-store`, pictures included, and the page cache is off, so a reload always shows the current state.
- **No `www` redirect, and no secure cookie**, so the site works on `localhost`.

Drafts are a separate flag, `--drafts`, so a draft can be checked with production's caching too.

## Reserved paths

A leading `_` segment is a namespace with one owner: `/_quire/` is Quire's, `/_<name>/` is a plugin's and `/_site/` is the site's own.
Every route is checked the same way, whoever adds it. `AddPage`, `AddRoute` and `AddFile` fail the build on a path in another owner's namespace or on one already taken, and a plugin's `Use` returns the same error.
Everything Quire serves for itself is under `/_quire/`: `health`, `reload`, `style.css`, `quire.js`, `datastar.js`, `fonts/`, `img/`.
Datastar signals follow the same split: `$quire` is Quire's, and a plugin's stream patches only `$<name>`.
Only the files that have to be findable at a known location stay at the root: `robots.txt` and `sitemap.xml`.

## In depth

### The server

A site has no server in it: its routes are a table.
The server makes a mux from that table and adds what claims the whole address space: the not-found page for every other path, `robots.txt`, the sitemap, the health check, the `www` redirect and the session cookie.

Building the site renders every fixed page, so a template error stops the binary at boot.
A site with routes of its own adds them in [Live](#live) with `AddRoute`, or as a plugin. Both go through the site's route claims, so a clash is an error at boot.

A test of a blueprint or a plugin needs no bundle: `quire.Build(identity, nil, blueprint)` and `quire.NewServer` give its handler.

### Without Run

Serving is three calls, and a program that wants a `main` of its own makes them itself:

```go
//go:embed all:.quire/bundle
var embedded embed.FS

func main() {
  files, err := fs.Sub(embedded, ".quire/bundle")
  if err != nil {
    log.Fatal(err)
  }
  baked, err := bundle.Open(files)
  if err != nil {
    log.Fatal(err)
  }
  site, err := quire.Build(identity, baked, blueprint)
  if err != nil {
    log.Fatal(err)
  }
  server, err := quire.NewServer(site)
  if err != nil {
    log.Fatal(err)
  }
  log.Fatal(http.ListenAndServe(":8090", server))
}
```

- `bundle.Open` reads a bundle from any `fs.FS`: one embedded by hand, as here, or a folder through `os.DirFS`.
- `quire.Build` runs the blueprint over it, `Live` included, and returns the site. It takes options as `Run` does.
- `quire.NewServer` returns the server, which is an `http.Handler`.

`quire.Build(identity, nil, blueprint)` builds a site with nothing baked. Its pages are all Go, it needs no bundle, and its stylesheet, scripts and fonts are served as for any other site.
What a site asks of a bundle is `quire.Source`, and anything with those six methods can stand in for one.

### A bake of your own

Baking is three calls too:

```go
func main() {
  baker, err := bake.Start(model.BakeSpec{Content: "content", Dir: ".quire"}, convert.Standard())
  if err != nil {
    log.Fatal(err)
  }
  if err := quire.Collect(identity, baker, blueprint); err != nil {
    log.Fatal(err)
  }
  if _, err := baker.Finish(); err != nil {
    log.Fatal(err)
  }
}
```

- `bake.Start` is told where the content and the output live, and handed what it can convert. `convert.Standard()`, from `pack/base/convert`, is the default set: markdown, mermaid, the share card and WebP pictures. Several sets are combined, and two converters of one name are refused.
- `quire.Collect` runs the blueprint over the baker, so the site asks for everything it needs. It calls no `Live`.
- `Finish` encodes the pictures, writes the bundle to `.quire/bundle`, and returns it.

A baker asked for a picture has not made it yet, because pictures are encoded together in `Finish`. Until then its answer is the master's name and nothing more, and the record is in the bundle `Finish` returns.

A program that serves with `quire` and bakes in a program of its own, as these two do, links no codec, no markdown and no highlighter into what it ships, with no build tag.

### Redirects

A trailing slash redirects to the path without one. `www` redirects to the host in `BaseURL`, and only to that host.
Both are sent with `Cache-Control: no-cache`, so a browser cannot pin one.

### Caching and compression

- A fixed page is rendered once, when the site is built.
- A collection's document page is rendered on first request and held in a 4MB page cache, keyed by its path.
- A raw route goes straight to its handler and is never cached.
- `robots.txt` and `sitemap.xml` are built once, at boot.
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
