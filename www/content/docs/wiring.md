---
title: Wiring
summary: How a site is put together in Go - its identity, its features, its options and its routes.
---

A Quire site is one call:

```go
quire.Run(identity, feature, options...)
```

- The **identity** is data: what the site is.
- The **feature** says what the site holds and how it looks. It is your own Go, a [pack](./packs.md), or both.
- The **options** are the few things that have to be known before the feature runs.

`Run` is in package `quire`, the helpers. Everything else on this page is package `site`, unless it says otherwise.

## Identity

`model.Identity` is what the site is:

- `Name` is the publication. The header shows it as the link home; see `WithBrand`.
- `Author` is who writes it.
- `Description` is the default page description.
- `BaseURL` is the production address. Setting it enables `/robots.txt` and `/sitemap.xml`.
- `Dates` is the layouts dates render in.

## The site's own feature

A [feature](./features.md) is the one way to add to a site, and the site's own is the one handed to `Run`.
Its `Declare` is handed the site and says what it holds. It runs at bake and at boot, so it only declares.
A site writes it as a function, and `site.FeatureFunc` makes it a feature:

```go
func declare(s *site.Site) error {
  s.AddStylesheets(static, "site.css")
  notes, err := s.Collection(model.Spec{ID: "writing", Title: "Writing"})
  if err != nil {
    return err
  }
  base.Pages(s, notes)
  s.AddPage(model.PageSpec{Title: "Writing", Path: "/"}, homePage(notes))
  s.SetNotFound(notFoundPage())
  return s.Include(basket.New("/basket"))
}

func main() {
  quire.Run(identity, site.FeatureFunc(declare))
}
```

### Content and pages

- `Collection(spec)` loads documents. A collection is its documents and nothing else. See [Content](./content.md#collections).
- `AddPage(head, body)` adds a fixed page at `head.Path`, drawn once and kept, in the navigation and the sitemap. The page at `/` is the home page.
- `AddCollectionPages(collection, listing, document)` gives a collection its listing and a page for each document, drawn on its first request and kept. One with nothing in it gets neither.
- `AddDocumentPages(collection, page)` is the second half alone, for a page that sets its own head data.
- `AddLivePage(path, page)` adds a page drawn for each request and never kept. It is handed the request and answers with the head data and the body.
- `AddRoute(method, path, handler)` is the raw mux: nothing added, never kept.
- `AddSharedRoute(path, handler)` is a `GET` whose answer is the same for every visitor, so it is drawn on the first request for a path and kept. See [Features](./features.md#a-route-that-is-kept).
- `AddFile(path, fsys, name)` serves a file directly, so Range still works, and returns its URL with a content hash, which is cached for good. Link to that.
- `Markdown(name)` returns one baked markdown file as a component, and `Document[T](name)` returns its frontmatter read into `T`, and its body. See [Content](./content.md#markdown-outside-a-collection).
- `Picture(name)` is a picture of the site's content, for a page of your own to place.
- `Include(feature)` runs another [feature](./features.md) on the site: a pack, a basket, a search box.
- `In(group)` is a [group](./features.md#groups) to add to, and `SetGroup(group, gate)` gives one its gate, which is what makes it served.
- `AddServices(services…)` says the site has these [services](./features.md#services), for whatever serves it to start and stop.

### A page is head data and a body

A page does not draw the document it sits in. It is two things, handed over together:

- `model.PageSpec`, its head data: the title, the address and what it shares. See [The head](#the-head).
- a body, which is any templ component.

The site puts the body in the frame and the frame in the document. So a body can be drawn alone, which is what a Datastar patch needs, and no template is handed the site.
`s.Page(head, body)` is that wrapping as a component, for a raw route that wants a whole page.

`model.DocumentPage(collection, document)` is the head data for a document's own page, and `collection.Page()` for its listing.

### The look

- `AddStylesheets` and `AddStylesheetsAt` add CSS, and `AddScripts` adds JavaScript to the site's one script. See [The look](./look.md#add-your-own-css).
- `SetFrame(frame)` draws what goes round every page's content: a header and a footer. The base pack's is `ui.Frame`, and `ui.FrameWith(header, footer)` is that frame with either of your own. Without a frame a page is its content and nothing else.
- `SetNotFound(body)` is the body of the page for an address nothing answers. Like any fixed page it is drawn once at boot and never handed the request.
- `SetDefaultHome(body)` is the body of the page at `/` when nothing else takes it.
- `SetFooterText(text)`, `AddFooterLinks(links…)` and `AddNavEnd(components…)` fill in the frame's footer line, footer links and the end of its nav. See [Links](./look.md#links).
- `SetWidget(name, widget)` adds a [widget](./content.md#widgets), and `SetPicturePlacer` and `SetVideoPlacer` draw the pictures and videos in a document's body. Each has to come before any content is read.

### Packs, and who wins

A pack is a feature with no name, included like any other: `s.Include(&docs.Pack{})`.

A site and its packs call the same methods, and the site's word always wins, whichever spoke first. A pack's frame, not-found page, footer text or widget gives way to the site's own of the same kind, and the site's CSS is laid over a pack's.
So a pack can be included before or after the site says its piece. Only one thing depends on order: a widget or a placer has to be set before the content it applies to is read.

### What a pack asks of the site

- `Identity()` is the identity the site was made with, `Chrome()` the site as a page sees it, with its scale and the palettes on offer, and `Palette()` its own palette.
- `AddBaseCSS(name, css)` adds to the bottom layer of the stylesheet, beneath every feature, pack and the site: the look everything else is laid over. It takes the CSS itself, since some of it is made and not read, such as what a palette becomes.
- `SetFonts(fsys)` is the font set a look draws in, unless the site chose one with `WithFonts`: a `fonts.json` and the faces it names. A site's own feature does not call it. `UseDatastar()` links Datastar on every page.
- `Converted(converter, body, args…)` is a converter's output for that input, as the bundle file to link to, and `Body(file)` reads one to inline it. `FontSet()` is the font set, for a converter that lays text out by its widths.
- `ShareCard()` is what the site's share card says, and `SetShareImage(file)` is the bundle file pages share. `Baked()` says there is a source to ask: a site built over nothing has no card.

The methods take three forms. A plain noun asks for something: `Collection`, `Picture`, `Document`. `Add` is one more of many. `Set` is the one of its kind, where the site's word beats a pack's.

A `site.Feature` is anything with `Name` and `Declare` methods, so a type of your own is one too, and so is a pack.
One with a name is held to its own namespace, and may not set the look; see [A name is a namespace](./features.md#a-name-is-a-namespace).
Nothing can be added once the site is built: a call after that panics.

## Options

An `Option` is something that has to be known before a feature runs.

- `WithTheme` is the colour scheme when the visitor has chosen none and the OS states none. The default is `ThemeLight`, and the visitor's own choice always wins.
- `WithPalette`, `WithPalettePicker` and `WithScale` set the colours and the page grid. See [The look](./look.md).
- `WithFonts(fsys)` is the font set the site draws in, in place of its look's. See [Fonts](./look.md#fonts).
- `WithBrand(name)` is the name the header shows as the link home, in place of `Identity.Name`. `WithBrand("")` shows none, and the nav starts with a Home link instead.
- `WithIcons(fsys, names…)` serves each file in `fsys` at `/{name}` and links it in the head as an icon, typed by its extension. A name starting `apple-touch-icon` is linked as one.
- `WithSiteCard` changes the share card the bake draws.
- `WithFeatures(features…)` declares features beneath the site's own, before it runs. `Run` puts the base pack there, and live reload in dev.
- `WithLog(logger)` is the `*slog.Logger` the site, its features and its services write to. The default is `slog.Default()`.
- `WithDev` and `WithDrafts` are what `--dev` and `--drafts` set.

Datastar is linked when a feature has streams, or one asks for it, as the base pack does for its theme toggle and copy buttons.
`Ship` is not an option: the `ship.go` the bake writes calls it to hand the binary its embedded bundles.

### Without the base pack

`quire.RunBare` is `Run` with no pack beneath the site: it bakes and serves, and adds nothing.
The site has a correct document and whatever its own feature says, with no stylesheet, script, frame or default page.
`examples/6-naked` is one, and `quire new --from naked mysite` copies it.

## Site and Chrome

- The `Site` is what a feature declares on. A page's body does not need it.
- `Chrome` is the site as a page sees it: the identity, the navigation, features' controls, assets, icons, theme and scale. It is data only. The frame is handed it, and anything drawn inside a page can read it with `model.ChromeOf(ctx)`, which is how the default home page lists the site's sections and how a picture knows the scale it is shown at.

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

`Collection.Page()` and `model.DocumentPage(collection, document)` fill a `PageSpec` for a collection's pages, so a page of your own rarely builds one by hand.
The sitemap lists every fixed page and every published document, with a document's `updated` or `date` as its last change.

## Dev mode

`--dev`, or `WithDev(true)`, is for writing, not for serving visitors.

- **Live reload.** `Run` includes the `feature/reload` feature, and the page reloads when the binary restarts. Restarting on a change is a watcher's job: the `dev` recipe in a site's justfile runs `wgo`.
- **Nothing is cached.** Every response is `no-store`, pictures included, and the page cache is off, so a reload always shows the current state.
- **No `www` redirect, and no secure cookie**, so the site works on `localhost`.

Drafts are a separate flag, `--drafts`, so a draft can be checked with production's caching too.

## Reserved paths

A leading `_` segment is a namespace with one owner: `/_quire/` is Quire's and `/_<name>/` is the feature's of that name.
A [group](./features.md#groups) the site has given a gate owns everything under its name, `/admin/`, and only what was added to the group is there.
A feature with no name, the site's own among them, has nothing under `/_`.
Every route is checked the same way, whoever adds it. `AddPage`, `AddRoute` and `AddFile` fail the build on a path in another owner's namespace or on one already taken.
Everything Quire serves for itself is under `/_quire/`: `health`, `style.css`, `quire.js`, `datastar.js`, `fonts/`, `img/`.
Datastar signals follow the same split: a feature's stream patches only `$<name>`.
Only the files that have to be findable at a known location stay at the root: `robots.txt` and `sitemap.xml`.

## In depth

### The server

A site has no server in it: its routes are a table, which `Routes()` reads.
The server, in package `server`, makes a mux from that table and adds what claims the whole address space: the not-found page for every other path, `robots.txt`, the sitemap, the health check and the `www` redirect.
It also runs the site's services: `Start` starts them in the order they were added, and `Stop` stops them in reverse.

Building the site renders every fixed page, so a template error stops the binary at boot.
A site with routes of its own adds them with `AddRoute`, in its own feature or one it includes. Both go through the site's route claims, so a clash is an error at boot.

A test of a feature needs no bundle: `site.Describe(identity, feature).Build(nil)` and `server.New` give its handler.

### Without Run

A site is described once, and the same description is baked and served. Serving is three calls on it, and a program that wants a `main` of its own makes them itself:

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
  built, err := site.Describe(identity, feature, site.WithFeatures(base.Pack{})).Build(baked)
  if err != nil {
    log.Fatal(err)
  }
  handler, err := server.New(built)
  if err != nil {
    log.Fatal(err)
  }
  if err := handler.Start(context.Background()); err != nil {
    log.Fatal(err)
  }
  log.Fatal(http.ListenAndServe(":8090", handler))
}
```

- `bundle.Open` reads a bundle from any `fs.FS`: one embedded by hand, as here, or a folder through `os.DirFS`.
- `site.Describe` gathers what makes a site but its source: the identity, the site's own feature and the options, as `Run` takes them.
- `Build` runs the features over the bundle and returns the site.
- `base.Pack{}`, from `pack/base`, is what `Run` puts beneath every site: what a document's fences and pictures become, the share card, and the home and not-found pages. Without it a `mermaid` fence is plain code, a picture is a bare `<img>`, no page has a share image but a document's own, and there is no page at `/` until the site adds one.
- `server.New` returns the server, which is an `http.Handler`. `Start` starts the site's services, and `Stop` stops them; `Run` calls `Stop` when the process is interrupted, after closing its connections.

A program with a mux of its own needs no server: it adds each of `built.Routes()` to its mux, and `built.NotFound()` is the page for anything else.

`Build(nil)` builds a site with nothing baked. Its pages are all Go, it needs no bundle, and its stylesheet, scripts and fonts are served as for any other site.
What a site asks of a bundle is `site.Source`, and anything with those six methods can stand in for one.

### Placing a site

A site writes its own addresses as if it were at `/`: its pages, its links, its features' endpoints and its groups.
Where it is served is a separate thing, a `model.Place`, and nothing in the site changes when that does.

```go
at := model.Place{Root: "/docs", Engine: "/docs/_quire"}
built, err := site.Describe(identity, feature, site.WithPlace(at)).Build(baked)
```

- `Root` goes in front of the site's pages, its features' `/_<name>/` and its groups: `/docs/install`, `/docs/_search/live`, `/docs/admin/`.
- `Engine` is where what the site has under `/_quire/` goes: its stylesheet, scripts, fonts and pictures. Left empty that is `/_quire` itself, which is right for a site alone. Under a host that owns the rest of the address space it is beneath the root, as here.

The place is applied at the two edges, and nowhere between:

- **Coming in.** `Routes()` gives the placed patterns, and a handler is handed the request with the site's own address in it.
- **Going out.** A page, any markup a handler answers with, and a stream's element patch have their addresses placed, in `href`, `src`, `srcset`, `action`, `poster` and what a Datastar action fetches. So are a redirect, the stylesheet's `url()`s, the canonical and share addresses, the sitemap and the session cookie's path.

An address built in a script, or written into an attribute not listed, is not placed; a link in a page always is.
The saved theme and palette are kept under the root, so two sites on one host do not share them.

`--root /docs` gives `Run` a place with everything under it, for a site behind a proxy that hands it one path.
`examples/mounted` is a program with a mux of its own that mounts a site under `/docs`, beside a page of its own drawn with the site's document.

### Several sites

`quire.RunSites` is `Run` for several sites in one server:

```go
quire.RunSites(
  quire.Site{Identity: model.Identity{Name: "Front"}, Feature: base.Content(), Content: "front"},
  quire.Site{Identity: model.Identity{Name: "Notes"}, Feature: base.Content(), Content: "notes", Root: "/notes"},
)
```

- `Root` is the path a site is served under. The first has none.
- `Content` is the site's content folder, `content` if it names none, so each of several names its own.
- `Options` are the site's own, as the last arguments of `Run` are.
- Each site is baked into a bundle of its own: the first into `.quire`, and one at `/notes` into `.quire/sites/notes`. The one `ship.go` embeds them all, so shipping is the same two commands.
- The flags are `Run`'s, and apply to every site. `--root` puts all of them under one path.
- Every site has base beneath it. Sites with no pack beneath them are composed by hand.

`examples/several` is two sites made this way.

Beneath it, a server takes any number of placed sites:

```go
v, err := server.New(
  front,                                                             // model.Place{}
  notes,                                                             // model.Place{Root: "/notes", Engine: "/_quire/notes"}
)
```

- **Each site is whole.** It has its own identity, features, look, groups and bundle, and is written as if it were alone at `/`. One of several can be given a server of its own later and nothing in it changes.
- **Everything Quire serves for itself is under `/_quire/`**, each site in a part of its own, so one rule at a proxy covers every site on the box.
- **The first site is the box's own.** The health check, `robots.txt`, the `www` redirect and the page for an address no site has are its. Every other site has its own not-found page under its root, and each has its own sitemap, which the one `robots.txt` lists.
- **Two sites with a route at one address** fail when the server is made.
- **Every site's services are started**, in the order the sites were given. One handed to several sites is started once, and told the environment of the first.


### A bake of your own

Baking is three calls too:

```go
func main() {
  baker, err := bake.Start(model.BakeSpec{Content: "content", Dir: ".quire"}, convert.Standard())
  if err != nil {
    log.Fatal(err)
  }
  if err := site.Describe(identity, feature, site.WithFeatures(base.Pack{})).Collect(baker); err != nil {
    log.Fatal(err)
  }
  if _, err := baker.Finish(); err != nil {
    log.Fatal(err)
  }
}
```

`quire.Bake(description, spec)` is those three calls with base's converters and any added with `quire.Converters`, and it also writes the `ship.go` that embeds what was baked.
A program with several sites bakes each into a folder of its own beneath `.quire`, and the one `ship.go` embeds them all.
`quire.Bundle(spec)` then opens a site's bundle: from the binary when it was built with that `ship.go`, and from disk otherwise.

- `bake.Start` is told where the content and the output live, and handed what it can convert. `convert.Standard()`, from `pack/base/convert`, is the default set: markdown, mermaid, the share card and WebP pictures. Several sets are combined, and two converters of one name are refused.
- `Collect` runs the features over the baker, so the site asks for everything it needs. It is the same description `Build` is called on, so the bake and the site cannot ask for different things.
- `Finish` encodes the pictures, writes the bundle to `.quire/bundle`, and returns it.

A baker asked for a picture has not made it yet, because pictures are encoded together in `Finish`. Until then its answer is the master's name and nothing more, and the record is in the bundle `Finish` returns.

A program that serves with `site` and `server` and bakes in a program of its own, as these two do, links no codec, no markdown and no highlighter into what it ships, with no build tag.

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

`/_quire/quire.js` holds the stored theme, soft navigation and any scripts a feature adds. It is a classic script loaded in the head, so the theme is set before the first paint.
Datastar is an ES module linked beside it.
Soft navigation replaces `main` and nothing else, so the header and anything in it lasts for the life of the tab.

A page can keep more than the header. Put `data-region="name"` on the element inside `main` that holds what changes:

```html
<main>
  <aside>a sidebar</aside>
  <article data-region="docs">the page</article>
</main>
```

Between two pages that both have that region, only the region is replaced, and everything round it stays as it was: its scroll position, its open panels, anything a feature is streaming into it.
Between any other two pages `main` is replaced, as before.
A link outside what was replaced takes its `class` and `aria-current` from the same link in the new page, so the header and a sidebar both show where the reader now is.
The sidebar on this site is kept that way: follow a link in it and only the page beside it changes.

While the next page is on its way, `html` carries `data-navigating`. Quire's own sheet draws a thin bar across the top after a moment, so a quick page shows nothing and a slow one shows the click was taken. When the page arrives the bar runs to the end and fades, with `data-navigating="done"` for that moment. A second click on the link already being fetched does nothing, and a click on another link drops the first.
Once the new page is in, `quirenavigate` fires on `window`. Anything kept across the swap can listen for it, as this site's sidebar does to close its menu on a narrow screen:

```html
<aside data-on:quirenavigate__window="$site.menu = false">
```
