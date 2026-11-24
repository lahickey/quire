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
A site made wholly from its content folder writes none: it hands `Run` [`base.Content()`](./content.md#a-site-from-the-folder).
A site with pages of its own writes it as a function, and `site.FeatureFunc` makes it a feature:

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
  return s.Include(search.New("/search", searchui.Look()))
}

func main() {
  quire.Run(identity, site.FeatureFunc(declare))
}
```

### Content and pages

- `Collection(spec)` loads documents. A collection is its documents and nothing else. See [Content](./content.md#collections).
- `Names(pattern)` is the documents under the content root that match a pattern, for a feature that finds out what is there.
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
- `AddServices(services…)` says the site has these [services](./services.md), for whatever serves it to start and stop.

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

A page hands the site a `model.PageSpec`, and the site writes the head from it.

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

## Beyond Run

`Run` is a helper over parts that are all public.
[The parts](./parts.md) is how to compose them yourself: a `main` of your own, a site at a path, several sites in one server, and a bake of your own.
