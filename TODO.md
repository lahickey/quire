# Quire — plan

Where Quire is going, the steps that get it there, and what else is waiting.
The model as it is today is [the design page](./www/content/docs/architecture.md).

## Where it is going

Quire's site, server, baker and bundle are parts with public interfaces, a feature is the one way to add to a site, and what it ships by default is the base pack.
Services are plain values the server starts and stops, a group is a part of a site behind one gate, and a site can be placed at any path.
A server holds several of them, and a route says whether what it answers is kept.
The plan finishes separating the parts so that each has one job and a public interface, and so that nothing Quire ships is a special case.
Quire is then the parts a static or live site is composed from, and a site engine in one binary is a helper over them.

### The words

Each word means one thing, in the code and in the docs.

The nouns:

- **Identity.** A site's name, base URL and description. Data only.
- **Feature.** Code that declares on a site: the documents it reads, its pages, routes, stylesheets and widgets. It only declares, so it runs in a bake as well as when serving, and it has no lifecycle. Several go into one site: the site's own, and those it includes.
- **Service.** Something that runs while a site is served and that others call: a store, sessions, a bus. It is started and stopped, and never sees the site.
- **Place.** Where a site is served: the root in front of its own addresses, and where the engine's are. The site does not know it.
- **Group.** A part of a site behind one gate, at the address of its name: `Admin` is `/admin`. Features add to it and the site gives it its gate. One with no gate is not served.
- **Core.** What cannot be removed: the site, the server, the baker's machinery and the bundle's format. It converts nothing and draws nothing by itself.
- **Source.** The requests a site makes for content: a document, a picture, the output of a converter. Everything below either answers them or asks them.
- **Baker.** A source that converts. It keeps the cache, the keys and the journal, runs the work and writes the bundle. It knows no formats: every conversion is a converter or a codec registered with it. It knows nothing of sites, routes or servers.
- **Bundle.** A journal and the files it names. It is a source that only reads.
- **Assets.** Stylesheets, scripts and fonts. They are embedded with the code, assembled and hashed at boot, and never baked.
- **Font set.** The fonts a site uses, chosen once: the faces a browser draws with, and how wide their characters are, for whatever lays text out in a bake. The share card draws with faces of its own.
- **Document.** One file of content, converted. It is not a page, which is an address, nor markdown, which is one format a document can come from. As a converter leaves it, it is a baked document: HTML with slots, its raw frontmatter and its headings.
- **Slot.** A place in a baked document that something goes: a picture, or a named fence. The bake fills the fences it can, and the site fills the rest when the page is composed.
- **Size.** How wide a picture is shown where it is placed: thumb, column, page, bleed.
- **Journal and record.** The index of a bundle, and one entry in it. Both are private to the baker and the bundle's reader. A record is what every baked thing has in common, with a payload for whatever its kind needs beyond that.
- **Master.** The file a picture is made from.
- **Picture.** A master and the ladder of variants made from it. "Image" is used only where somebody else's standard says it: the `<img>` tag, markdown's syntax, `og:image`.
- **Mermaid.** A diagram written as mermaid, and the name of what converts and places it. "Diagram" is left free for charts and graphs, which are a separate thing to come.
- **Site.** Features run over a source: collections, pages, chrome, files and a table of routes. It is the table a feature declares on, and knows no server. Over an empty source it is a site with no bundle.
- **Server.** It holds entries, each a root, a bundle and a site, and owns the mux, the caches and the services' lifecycle.

Converter, codec, widget, feature, service and pack are under their own headings below.

The verbs:

1. **Declare.** A feature says what the site contains. It is the same call in a bake and when serving.
2. **Collect.** Run the features over a baker, so the site asks for what it needs.
3. **Bake.** The baker converts and journals, then writes the bundle of what was referred to.
4. **Build.** Run the features over a bundle, which makes the site.
5. **Mount.** Put a site at a root, in a server or in a mux that is somebody else's. The word means nothing else.
6. **Serve.** The server answers requests for the sites it holds.

```
features + baker   --bake-->   bundle
features + bundle  --build-->  site
site               --mount-->  a server, or a mux of your own
```

Collecting links the site's Go, because a feature reads its documents while it declares.
So bake and serve are separate packages and phases, not programs that never meet.
A CMS skips collecting and makes the baker's requests from its own logic.

### The tree

Each package imports only what is below it, so the top is plain to see.

```
main, quire        the helpers, which compose everything
server             serves sites and runs services
features, packs    declare on a site
site               the table a feature declares on
model, streams     data, and a library for live updates
```

The bake side has the same shape: a converter imports the baker, and the baker imports the bundle and the model.

The site knows no feature, no service and no server.
The root package `quire` is the helpers, and nothing but a site's `main` imports it.
A helper is one function for a common pattern, such as a content folder baked and served as a blog.
Helpers call only what is public, and are where the smallest site stays small.
Nothing in the parts calls a helper, and a site is free to compose the parts itself.

`quire.Run` is the site engine in one binary: it bakes, builds and serves.
`Collect` and `Build` are not helpers. They are the verbs, and the only way a site is made.

The parts need no build tag. A program with its own bake program never links a codec into what it ships.
`quire_nobake` lives on inside the helpers only, to stub the bake half of `Run` for a site that wants one `main`.
A serving `main` written by hand against the parts is about six lines, and earns a helper when something needs it.

### Features

A feature is the one way to add to a site.

```go
type Feature interface {
    Name() string
    Declare(*Site) error
}
```

- **The site runs each feature**, so it knows whose turn it is and can hold a feature to its name.
- **A named feature has a namespace of its own.** Its routes are under `/_name/`, its classes begin `name-`, and its streams are scoped by the name. The site refuses anything outside it.
- **A feature with no name works in the site's namespace.** It adds pages, plain routes, stylesheets, widgets and the frame, and cannot have what needs a scope. The site's own is one, and so are base and docs, which are a site's look.
- **The site's own word wins.** What it sets is never replaced by a feature it includes, whichever spoke first.
- **It only declares.** A bake runs every feature and serves nothing, so whatever a feature does in a handler cannot happen in a bake.
- **It is handed what it needs** by its constructor: its settings and its services.
- **It is included per site.** One value included in several sites shares whatever state it keeps.
- **It has no lifecycle.** A feature that needs something running is handed a service that runs it.

Streams are a library, not a feature: a feature that updates a page live makes one and publishes to it.

What a feature declares may have a half that runs in a bake, under the same name.

| Role | Runs | Does |
|-|-|-|
| Converter | at bake | bytes and settings in, one file out |
| Codec | at bake | opens an input once, answers many requests |
| Widget | at page render | places output in the page |
| Route | while serving | answers a request, opens a stream |

- The name is what joins the half that bakes to the half that renders.
- They are registered apart because they are linked apart: what converts is heavy and stays out of a serving binary, and what a feature declares is light and is in it.
- A converter or a codec sits in a package of its own, so a serving binary that imports the feature does not link it.
- A converter has a version of its own, in the key of everything it makes.
- A codec is for what is costly to open: a picture decoded once for every rung, a video opened once for every frame.
- A document converter claims a file extension, so markdown is one way to a document and not the only one.
- Core defines the few structures it reads, and does not know what made or will render them. A **document** is one, since collections are built from it. A **picture** is one, widths and the file for each, since sizes and the scale are core's layout. Both are in `model`, and nothing else of a bundle's insides is public.
- Anything else a converter makes is one file that only its widget understands. Mermaid's is an SVG its widget inlines, and charts will likely be the same.
- A live chart is one feature that sets a widget and adds a route, with its converter beside it.

### Services

A service is a plain value that runs while a site is served: a store, sessions, a mailer.
It has a lifecycle and never sees the site, where a feature declares and has none.
Nothing is both.

```go
type Service interface {
    Start(context.Context, model.Env) error
    Stop(context.Context) error
}
```

A site makes its services, adds them, and hands them to what uses them.

```go
carts := store.NewMemory()
s.AddServices(carts)
s.Include(basket.New(carts, "/basket", items...))
```

- **Handed over by constructor.** A feature, another service or the site's own handler takes what it needs as an argument, typed by what it needs. A missing one does not compile, a test passes a fake, and nothing is looked up by name or by type.
- **Every service is added**, whether or not it has anything to start yet, so one that grows a lifecycle later changes no site. The site lists them for the server, which starts them in the order they were added and stops them in reverse.
- **A constructor only stores its arguments.** Connections and goroutines are opened in `Start`. A bake never calls `Start`, so only the constructor is left to convention.
- **`Start` is told the environment**, `model.Env`: the site's identity, whether it is in dev, and its logger. What is fixed for the life of the site is settled there, once, and never worked out for a request. `Env` can grow without the interface changing.
- **A feature that scopes its data does it itself**, with `store.Under(st, "basket")`, so every feature is handed the same store.
- **The site chooses what is behind an interface**: a store in memory for a test, SQLite for a small site, the site's own for anything else.
- **A heavy implementation is a package of its own**, so a site links only the one it chose.

The standard ones are a store and sessions.
Sessions are who the visitor is, and features ask: a visitor is given a cookie the first time something asks.
A logger is not a service: it is a `*slog.Logger`, in the environment for a service and from the site for a feature.
Others, such as mail, take the same shape as the store.

The interfaces are right when a basket, a sign-up form, comments and sign-in could each be written against them and nothing else.

### Groups

A group is a root, a gate and, in time, a cache of its own.

- **Typed, not spelt.** `site.Members` and `site.Admin` are constants, and a site's own group is one more.
- **Features add to it** with `s.In(site.Admin)`, at `/admin/<feature>/`, so every feature's admin pages sit together behind one gateway.
- **The site gives it its gate**, a wrapper round every handler in it. Authorisation is the gate's: "has a token" for members, "has a token with the admin flag" for admin. Quire knows nothing of tokens.
- **Safe by default.** A group with no gate is not served, and nothing else may be added under a served group's root.
- **The gate is outside any cache**, so a page kept for a group is only reached through its gate.

A named feature's `/_name/` and the site's `/` are the same idea with no gate, and rooting a site at a path is giving `/` a root.
Quire ships no gate yet: sign-in is a service to come.

### Events

A bus lets a feature say what happened without knowing who acts on it: a basket publishes an order, and fulfilment and mail each listen.
It is not built yet. It is a service like any other, and these hold so that it can be added:

- **Subscribing is registering**, like adding a route. A feature subscribes in `Declare` and a service in its constructor. The bus keeps handlers before it starts and delivers only while it runs, so a bake delivers nothing.
- **It is added first**, so it has started before anything publishes.
- **An event is data**: a subject and a value that can be written out, never a pointer or a function. A bus in memory and one over NATS are then the same to whoever uses them.
- **Delivery is at least once, in no order, and not while the publisher waits.** A handler takes a repeat without harm. A bus in memory does better, and nothing relies on it.
- **Subjects are NATS's.** Dotted tokens that begin with the publisher's name, `basket.ordered`, and a listener filters as NATS does: `basket.*` for one token, `basket.>` for all that follow. The bus is NATS or something that mimics it.
- **Events only.** Something that needs an answer is handed the service and calls it.

### Caching

A route states how its response may be kept, and the server does not decide by what kind of route it is.

- **Shared**: one body for every visitor, as a page is today. A feature's route can ask for it.
- **None**: as a raw route is today.
- **Per group**: a group has a cache of its own, inside its gate, so what is kept for admins is never served to anyone else.
- **Per visitor**: keyed by the session, in a budget of its own so it cannot push out shared pages. Only when something measurably needs it: a shared page with its personal part filled live, as the basket link is, covers most cases.

Whatever changes what a kept page shows can drop it.
Nothing is dropped today, because nothing can change while a site runs.

### Packs

A pack provides any number of features, services, converters, components, stylesheets, fonts and palettes.
It is how they are shipped together, and not a mechanism of its own.
Changes to the base colour system are a pack's to make too.
A pack whose feature has pages is a whole kind of site, as `pack/docs` is.

What Quire ships is one pack, **base**, which holds every default: markdown, the WebP codec, mermaid, the share card, the default look and its components, and the default font files and palettes.
Search, sessions and a store are mechanisms, not look: each stays outside base, and base brings only how search is drawn.
Each is a package of its own inside it.
Without base Quire is naked: it serves what a program declares and converts nothing.

Base is written against the interfaces any other pack uses, and a test holds it to it.
Turning something off is not registering it, never a flag: what is not imported is not linked.
A helper gives a site the standard set, so the smallest site stays as small as it is.

What base weighs in a binary is mostly its fonts, 210KB with their widths, then Datastar at 36KB, which base asks for on every page because its theme toggle and copy buttons are written with it.
Its stylesheets are 27KB and its templates less.

### One server, several sites

A site writes its addresses as if it were at `/`, and a place says where it is served: a root for its pages, features and groups, and where the engine's `/_quire/` is.
The place is applied only at the edges, to the route table coming in and to what is sent going out, so a site, a feature and a template never mention it.
A placed site can be mounted in somebody else's server, beside routes Quire does not manage, or sit behind a proxy at a path.

A server holds any number of placed sites, each with its own bundle, and the first is the box's own.
Everything Quire serves for itself is under `/_quire/`, each site in a part of its own.
"Several sites" is nothing more than several places: the documentation with its examples under `/examples/`, a subdomain each, or a box of tenants.
The operator's level is the site at `/` and its `Admin` group; a tenant's is its own.

### Inside another program

A program with a web layer of its own uses Quire for as much as it wants and no more.

- **A whole site at a root.** Its documentation, built from a bundle and mounted in its own mux, with no server of Quire's.
- **Its own pages from the pieces.** The shell and components, the assembled stylesheet, hashed URLs and their cache headers, live reload, sessions and streams, each a handler or a wrapper it can take alone.
- **Nothing baked.** A site whose pages are all Go has no bundle, and its assets are still embedded and served.

Whatever claims the whole address space is the server's, so a host that does not use the server is not bound by it: the catch-all not-found page, `robots.txt`, the sitemap, the health check and the redirect to the apex.
A site's own files, its pictures among them, are routes in its table and go where it is mounted.
What fails while a site is built is returned as an error, for the host to act on.

## The steps

Each step leaves everything working and is worth having if nothing after it happens.

1. **Split the engine into site and server.** Done.
2. **Make the source, the baker and the bundle public.** Done.
3. **Declare widgets once for a site.** Done.
4. **Packs provide the parts.** Done.
5. **One table, and one way to add to it.** Done.
6. **Services and groups.** Done.
7. **Place a site at a path.** Done.
8. **Several sites in one server.** Done.
9. **Let a route ask to be shared.** Done.
10. **The helper layer.** Done.
11. **Make the examples templates as well as tutorials.** Done.
12. **Show the examples on the documentation site.** Done.

## To decide

- **The footer's text and links.**
  They are the default frame's, yet the site holds them, because a pack and the site each add links to one footer and only the site can gather them.
  Either they stay as what any frame may show, as the nav's controls are, or the frame gathers them some other way.
- **Datastar on a site with nothing live.**
  Base asks for it on every page, because its theme toggle, palette picker and copy buttons are written with it. In plain script they would not need it, and an ordinary site would be 36KB lighter.
- **A widget against a language.**
  A site's widget beats a pack's quietly, but a widget named for a language the bake highlights is an error. The reason is mechanical: the fence's text is gone once it is highlighted.
- **A name for a site's description.**
  `site.Describe` returns a `Description`: the identity, the site's own feature and the options, which is collected over a baker and built over a bundle. The name is a placeholder.
- **The strapline.**
  "Composable parts for static and live sites in Go" is a placeholder, in the README, the site's description, the home page's kicker and the sample of it on the Packs page.
  The README's second line and the home page's title and lede still carry the older pitch.
- **"Diagram" in the docs.**
  The pages say a `mermaid` fence is a diagram, and the section is `#diagrams`.
  Either that stays as plain English, or the docs say mermaid and leave the word for charts.
- **Who holds a subject to its publisher's name.**
  A publisher holds the bus itself, so the site cannot.
  It is a convention, or the bus is handed over already scoped.
- **Many sites from many repositories on one machine.**
  Entries cover sites built into one binary.
  Sites that ship as separate binaries would need a front that routes by hostname and starts each on its first request, perhaps as `quire host`.

## Alongside

These depend on none of the steps.

- **Feeds** — RSS/Atom. `Collection` already holds everything required.
  It would live in base.
- **Inputs a pack embeds.** The baker reads only the site's content folder, so a pack cannot ship a picture, and a font's widths are written into `fonts.json` by `quire fonts` and committed.
  With a way for a pack to hand the baker files of its own, a pack's pictures are baked like a site's, and a TTF beside a set's woff2 is a bake input: widths are derived and hashed like anything else, and the tool and the committed widths go.
  Until then a site with its own fonts that changes a woff2 and does not run `quire fonts` again gets diagrams laid out for the old one, with no error.
- **A crash in Go's compiler**, [golang/go#81627](https://github.com/golang/go/issues/81627), open with a fix in review.
  A generic function from another package, with its type inferred, crashes the compiler as an element of a composite literal: a struct field, or a slice or map element.
  Writing the type argument out avoids it. Nothing in Quire needs the crashing form now; drop this note when a point release carries the fix.
  Found 2026-11-20. Look once a month, next on 2026-12-20.
- **Hand-written lexers for more of Lexis's languages.** [The benchmarks](./www/content/docs/benchmarks.md) list those run from rules slowest first; Java, Makefile and bash lead, and Rust and Lua are wanted.
- **More languages in Lexis**, as they are wanted. Swift, Ruby and PHP have rules in Chroma to import, each needing more rewriting than those done.
- **A language from a file's first line.** A script with no extension, or XML under an extension Lexis does not list, is shown plain; when no filename matches, Lexis could look at the first line for `#!` or `<?xml`.
- **zsh of its own.** It is lexed as bash, so what only zsh has is coloured roughly.
- **Structure from Lexis**, for archr's anchoring and its `gd` and `gr`: Go's parser for Go, and a scan for the enclosing function in C and C++.
- **Folders in the docs sidebar.** The docs pack's sidebar is one flat list, set in `docs.sections`, and its documents sit in one folder.
  A document in a folder, `docs/lexis/languages.md`, would be served at `/docs/lexis/languages` and listed under the folder's own page, closed unless the visitor is in it.
  The order stays the one `docs.sections` gives, as order on disk is alphabetical.
  Whether a collection takes a slug with a folder in it is the first thing to find out.
  With it, each of Lexis's languages gets a page of its whole file, under the page of excerpts.
- **A page that is not in the navigation.** `AddPage` always adds its page to the nav and the sitemap, so a site has no way to serve a page that should only be reached by its address: a privacy policy, a terms page, a landing page for a campaign.
  The portfolio serves Storm the Train's privacy policy as a static HTML file through `AddFile` for that reason, outside the site's shell and stylesheet.
  With it, such a page is a markdown file placed with `Markdown` in the shell like any other.
  Whether it stays out of the sitemap as well is a second choice, and may not be the same answer for every page.
- **Full-bleed and the scrollbar.** `.figure--bleed` measures in `cqw` against `main`, because `100vw` counts the scrollbar and would put a horizontal scrollbar on every desktop. The price is `container-type: inline-size` on `main`, whose layout containment makes it the containing block for any `position: fixed` descendant. Revisit against the two alternatives: hiding the document scrollbar in CSS (`scrollbar-width: none`, which costs every visitor their scroll position indicator), or setting the gutter width once from the classic script. Measured; no decision taken.
- **A bleed hero on a short screen.** Full width, a hero can fill a laptop screen with no text in sight. The usual answers: cap its height (50–60% of the screen, cropped with `object-fit: cover`) so text always shows below; a focal point per picture so the crop keeps what matters; different crops per screen size through `<picture>`. Height cap and focal point are options on the bleed figure; laying title and buttons over the picture is a site's layout choice. Tried with 884px art, which also showed the page padding left between nav and a bleed hero.
- **Settings at the call site.** A picture's conversion is chosen where it is placed, in the markdown or the template: lossless, a quality, a second format such as AVIF, a shorter ladder. Variant names already cover every setting and a record already lists formats, so the bake's `Master` takes the options and nothing else moves. Every variant is lossy WebP until then.
  A second format is then a codec a pack can bring.
- **A file of source as a code block.** A document names a file and the bake puts its text on the page, highlighted, whole or a range of its lines.
  The file may be outside the content folder, as Lexis's corpus is, so the bake has to read it and count it as an input, to bake the page again when it changes.
  Lexis's languages page is then a document written by hand, and `just lexis languages` goes.
- **A ladder from use.** The bake renders its pages too, records which rungs each placement can pick, and encodes only those rather than the whole ladder for every master. It is the same record as the call-site settings.
- **An analytics pack.** Platoon's analytics and feature flags, built again on Quire as a pack: a PostHog a site hosts itself.
  It needs most of what a service can be, so it is the test of the design.
  The same anonymous visitor on every page is a call from the page itself, which leaves every cache alone, to a route that asks sessions who it is.
  What every other feature did is the bus, listened to with `>`.
  Its queries are past a key-value store, so it brings a database of its own as a service.
  Its dashboards are in the `Admin` group.
  Between taking an event in and working on it is a queue, as Platoon had in NATS.
  A flag changes what one visitor is shown, so flags wait on per-visitor caching, or fill the flagged part live.
- **Warnings.** The site, the baker and every service write to one logger. These are the gremlins to warn on.
  **A master in bad shape.** A bake reads whatever master content refers to and corrects it in memory, so nothing here breaks a page — it costs work on every changed master, and it costs the chance to copy rather than re-encode.
  Only `quire ingest` writes a master, so every one of these says the same thing: run it, or fix the file at the source.
  - **Master over the cap.** Every bake fits it in memory before it can measure or encode, and the committed file is bigger than anything that will ever be served from it. Expensive if it is a print original somebody added without thinking.
  - **Master with an EXIF orientation flag.** The bake rotates it in memory each time, and the widest rung can no longer be copied from it, so that rung is re-encoded for nothing. Rotating once at ingest settles it.
  - **Master still carrying EXIF, XMP or an ICC profile.** Quire strips what it writes, but a file added by hand is committed, embedded and served as it is — GPS from a phone included. Nothing warns today, and nothing else will notice.
  - **Master that is neither JPEG nor WebP.** A PNG master works: it is decoded and every rung encoded. But the widest rung can never be a copy, and the repo carries a file several times the size of anything served.
  - **Master with a non-sRGB ICC profile.** Quire strips profiles, and no variant format it emits could carry one anyway. The pixels are then read as sRGB, so a Display P3 or Adobe RGB original renders oversaturated and nobody can tell it was avoidable. Convert before ingest.
  - **Master narrower than the widest size that uses it.** The ladder stops at the master, so nothing breaks — the hero is just soft, and nobody finds out until they look at it on a dense screen.
  **Everything else.**
  - **Bake of hundreds of pictures into a FROM-scratch binary.** A bake writes one file per rung per format, seven for a full ladder. Tens of pictures is fine. Hundreds makes the artefact fat and the build slow — the person will notice; a warning should say so.
  - **A size wider than the 2560 floor.** The cap follows the widest size, so one size silently makes every master on the site bigger. Intended when someone wants a full-bleed hero on a large desktop; an accident when a width was typed with an extra digit.
  - **A public-bucket URL for a variant the bucket does not hold.** Once the browser is sent to the bucket, Quire is out of the loop: the visitor gets a broken image. The bake did not run.
- **Sign-in.** A service that knows who a visitor is beyond a cookie, with the gates `Members` and `Admin` need: has a token, and has a token with the admin flag.
  Its sign-in page and callback are a feature that is handed it.
  With a gate, a group gets a cache of its own inside it, so what is kept for admins is never served to anyone else.
- **Pre-rendered pages.** The bake runs the features, so after the make a server over the new bundle can render every page the page cache would render once for everyone into the bundle, as Astro's `dist/` and Hugo's `public/` do, and the shipped binary serves finished HTML. Plugin routes and anything per visitor stay live, the per-route split Astro and Next make. It is a third axis beside `quire_nobake` and the ship overlay: whether its pages are pre-rendered (a light site's would be). A site with no live routes at all needs no server, only its bundle on a static host. Documents are already converted in the bake, so the gain is boot time and static hosting rather than weight.
  A static host often serves under a path, which is what a place is for.
- **Public-bucket URLs, in the markup.** Emit the URL of variants somebody else serves in `srcset` so the browser never asks the origin at all. Edge-cached, no egress charge. The constraint is unchanged: every referenced variant must be in the bucket before it is referenced.
  A picture's URL goes out through the same place every other address does.
- **A site placed by host.** `shop.hiddenbridge.io` for one of several sites in a server. The mux already routes by host; what is the box's own today, the `www` redirect and `robots.txt`, becomes each host's.
- **A group gathered above the sites.** `/admin/<site>/` behind the server's own gate, for a platform whose operator owns admin. A group travels with its site until something wants this.
- **Dev re-bakes in-process.** A site already bakes before it serves; with `--dev` it could also re-bake when content or masters change, so a watcher is only for Go and templ. A CMS needs the same thing: re-bake after an upload and swap the bundle, rather than a second bundle beside the first.
  Swapping a bundle is replacing an entry's and dropping what was kept from the old one, so a way to drop what the page cache keeps is built with it.
