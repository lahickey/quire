# Quire — plan

Where Quire is going, the steps that get it there, and what else is waiting.
The model as it is today is [the design page](./site/content/docs/architecture.md).

## Where it is going

Quire's site, server, baker and bundle are parts with public interfaces, and `app.Run` is the site engine in one binary over them.
What it ships by default is still wired into the site and the helper, and is not yet a pack.
The plan separates the parts so that each has one job and a public interface, and so that nothing Quire ships is a special case.
Quire is then the parts a static or live site is composed from, and a site engine in one binary is a helper over them.

### The words

Each word means one thing, in the code and in the docs.

The nouns:

- **Identity.** A site's name, base URL and description. Data only.
- **Blueprint.** The code that says what a site is made of: which documents it reads, which pages it has and how it looks. It does nothing by itself. Several go into one site: the site's own, and one from each pack that has pages.
- **Core.** What cannot be removed: the site, the server, the baker's machinery and the bundle's format. It converts nothing and draws nothing by itself.
- **Source.** The requests a site makes for content: a document, a picture, the output of a converter. Everything below either answers them or asks them.
- **Baker.** A source that converts. It keeps the cache, the keys and the journal, runs the work and writes the bundle. It knows no formats: every conversion is a converter or a codec registered with it. It knows nothing of sites, routes or servers.
- **Bundle.** A journal and the files it names. It is a source that only reads.
- **Assets.** Stylesheets, scripts and fonts. They are embedded with the code, assembled and hashed at boot, and never baked.
- **Font set.** The fonts a site uses, chosen once. A bake measures text with it and the browser draws with it, so both are handed the same set; only the file format differs.
- **Document.** One file of content, converted. It is not a page, which is an address, nor markdown, which is one format a document can come from. As a converter leaves it, it is a baked document: HTML with slots, its raw frontmatter and its headings.
- **Slot.** A place in a baked document that something goes: a picture, or a named fence. The bake fills the fences it can, and the site fills the rest when the page is composed.
- **Size.** How wide a picture is shown where it is placed: thumb, column, page, bleed.
- **Journal and record.** The index of a bundle, and one entry in it. Both are private to the baker and the bundle's reader. A record is what every baked thing has in common, with a payload for whatever its kind needs beyond that.
- **Master.** The file a picture is made from.
- **Picture.** A master and the ladder of variants made from it. "Image" is used only where somebody else's standard says it: the `<img>` tag, markdown's syntax, `og:image`.
- **Mermaid.** A diagram written as mermaid, and the name of what converts and places it. "Diagram" is left free for charts and graphs, which are a separate thing to come.
- **Site.** Blueprints run over a source: collections, pages, chrome, files and a table of routes. It has no server. Over an empty source it is a site with no bundle.
- **Server.** It holds entries, each a root, a bundle and a site, and owns the mux, the caches and the services' lifecycle.

Converter, codec, widget, plugin, service and pack are under their own headings below.

The verbs:

1. **Declare.** A blueprint says what the site contains. It is the same call in a bake and when serving.
2. **Collect.** Run the blueprints over a baker, so the site asks for what it needs.
3. **Bake.** The baker converts and journals, then writes the bundle of what was referred to.
4. **Build.** Run the blueprints over a bundle, which makes the site.
5. **Mount.** Put a site at a root, in a server or in a mux that is somebody else's. The word means nothing else.
6. **Serve.** The server answers requests for the sites it holds.

```
blueprints + baker   --bake-->   bundle
blueprints + bundle  --build-->  site
site                 --mount-->  a server, or a mux of your own
```

Collecting links the site's Go, because a blueprint reads its documents while it declares.
So bake and serve are separate packages and phases, not programs that never meet.
A CMS skips collecting and makes the baker's requests from its own logic.

### Two layers

The parts are the machinery, and each is usable without the others.
A helper is one function for a common pattern, such as a content folder baked and served as a blog.
Helpers sit in a layer above the parts, call only what is public, and are where the smallest site stays small.
Nothing in the parts calls a helper.

The helpers are a package of their own, `app` for now, so the compiler holds the layers apart: `quire` cannot import it.
`app.Run` is the site engine in one binary: it bakes, builds and serves.
`Collect` and `Build` are not helpers. They are the verbs, and the only way a site is made.

The parts need no build tag. A program with its own bake program never links a codec into what it ships.
`quire_nobake` lives on inside `app` only, to stub the bake half of `Run` for a site that wants one `main`.
A serving `main` written by hand against the parts is about six lines, and earns a helper when something needs it.

### What a feature is made of

| Role | Runs | Does |
|-|-|-|
| Converter | at bake | bytes and settings in, one file out |
| Codec | at bake | opens an input once, answers many requests |
| Widget | at page render | places output in the page |
| Plugin | while serving | routes, streams, sessions |

- A feature is these under one name. The name is what joins the half that bakes to the half that renders.
- They are registered apart because they are linked apart: what converts is heavy and stays out of a serving binary, and what renders is light and is in it.
- A converter or a codec sits in a package of its own, so a serving binary that imports the widget and the plugin does not link it.
- A converter has a version of its own, in the key of everything it makes.
- A codec is for what is costly to open: a picture decoded once for every rung, a video opened once for every frame.
- A document converter claims a file extension, so markdown is one way to a document and not the only one.
- Core defines the few structures it reads, and does not know what made or will render them. A **document** is one, since collections are built from it. A **picture** is one, widths and the file for each, since sizes and the scale are core's layout. Both are in `model`, and nothing else of a bundle's insides is public.
- Anything else a converter makes is one file that only its widget understands. Mermaid's is an SVG its widget inlines, and charts will likely be the same.
- A plugin is for live features only.
- A fence may need all three: a live chart converts at bake, is placed by a widget, and is updated by a plugin.
- A plugin is attached per site. One instance attached to several sites shares whatever state it keeps.

### Services

A service is something a plugin needs and the site provides.
The plugin names the interface and the site chooses what is behind it: a store in memory for a test, SQLite for a small site, the site's own for anything else.

- **One interface, with a lifecycle.** The server starts services in the order they were added and stops them in reverse.
- **Reached through the handle.** A plugin asks the handle it is attached with, which also gives it a logger carrying its name. A service is handed the same.
- **Kept apart by name.** A plugin's data is under its own name, as its routes and classes already are, and under its site's when a server holds several.
- **Missing is an error at boot.** A plugin that needs a store on a site with none stops the site starting. Nothing falls back to memory unasked.
- **An implementation is a package of its own**, so a site links only the one it chose.

The standard ones are a store, sessions and a log.
Sessions are a service because they have no routes or pages: they are who the visitor is, and plugins ask.
The log is the one the baker's warnings go to as well.
Others, such as mail, take the same shape.

The interfaces are right when a basket, a sign-up form, comments and sign-in could each be written against them and nothing else.

### Caching

A route states how its response may be kept, and the server does not decide by what kind of route it is.

- **Shared**: one body for every visitor, as a page is today. A plugin's route can ask for it.
- **None**: as a plugin's route is today.
- **Per visitor**: keyed by the session, in a budget of its own so it cannot push out shared pages. Only when something measurably needs it: a shared page with its personal part filled live, as the basket link is, covers most cases.

Whatever changes what a kept page shows can drop it.
Nothing is dropped today, because nothing can change while a site runs.

### Packs

A pack provides any number of converters, widgets, plugins, services, components, stylesheets, fonts and palettes, and optionally a blueprint.
Changes to the base colour system are a pack's to make too.
A pack with a blueprint is a whole kind of site, as `pack/docs` is.

What Quire ships is one pack, **base**, which holds every default: markdown, the WebP codec, mermaid, the share card, the default look and its components, and the default font files and palettes.
Search, sessions and a store are mechanisms, not look: each stays outside base, and base brings only how search is drawn.
Each is a package of its own inside it.
Without base Quire is naked: it serves what a program declares and converts nothing.

Base is written against the interfaces any other pack uses, and a test holds it to it.
Turning something off is not registering it, never a flag: what is not imported is not linked.
A helper gives a site the standard set, so the smallest site stays as small as it is.

What base weighs in a binary is mostly its fonts, 212KB, then Datastar at 36KB, which base asks for on every page because its theme toggle and copy buttons are written with it.
Its stylesheets are 27KB and its templates less.

### One server, several sites

A server with one entry at `/` is a site as it is today.
Several entries are several sites, or one site in parts, each under its own root and each with its own bundle or sharing one.
A site with a root other than `/` can also be mounted in somebody else's server, beside routes Quire does not manage, or sit behind a proxy at a path.

### Inside another program

A program with a web layer of its own uses Quire for as much as it wants and no more.

- **A whole site at a root.** Its documentation, built from a bundle and mounted in its own mux, with no server of Quire's.
- **Its own pages from the pieces.** The shell and components, the assembled stylesheet, hashed URLs and their cache headers, live reload, sessions and streams, each a handler or a wrapper it can take alone.
- **Nothing baked.** A site whose pages are all Go has no bundle, and its assets are still embedded and served.

Whatever claims the whole address space is the server's, so a host that does not use the server is not bound by it: the catch-all not-found page, `robots.txt`, the sitemap, the health check, the redirect to the apex and the session cookie.
A site's own files, its pictures among them, are routes in its table and go where it is mounted.
What fails while a site is built is returned as an error, for the host to act on.

## The steps

Each step leaves everything working and is worth having if nothing after it happens.

1. **Split the engine into site and server.** Done.
2. **Make the source, the baker and the bundle public.** Done.
3. **Declare widgets once for a site.** Done.
4. **Packs provide the parts.** Done.
5. **Services.**
   The interface and its lifecycle, the handle that gives a plugin its services and its logger, and a store with an implementation in memory.
   Whether search's index is a service, so a site can bring a search engine of its own, is settled here.
   Sessions move out of the server and become one, which needs a way to wrap a request that plugins do not have today.
6. **Give the server entries.**
   Several of root, bundle and site.
   `Ship` registers bundles by name, and one `ship.go` embeds every bundle under `.quire`.
7. **Let a route state its cache policy.**
   Shared or none, on any route, and a way to drop what is kept.
8. **Root a site at a path.**
   The root is prepended wherever a URL is built, and a rendered page has its root-absolute URLs prefixed before it is cached, so a hand-written `/writing` is still right.
   A site at `/` skips all of it.
   The session cookie and the saved theme and palette are scoped to the root.
   `examples_test.go` serves every example a second time under a root and fails on any URL that escapes it.
9. **Offer the pieces to a host.**
   The assets, the cache headers, live reload and sessions are each reachable without a site or a server.
   An example is a program with its own mux that mounts a documentation site at a root and draws a page of its own from the pieces.
10. **Show the examples on the documentation site.**
   An Examples page with a screenshot of each, what it adds, its `quire new` line and its source.
   Then each example live, as an entry under `/examples/`.
11. **Tidy the helper layer.**
   `Run`, `model.Basic` and the other shortcuts sit together in `app`, and say which parts they compose.
   A helper for each common pattern, so the smallest example is one call.
12. **Make the examples templates as well as tutorials.**
   A proper pass over every example once the parts and the helpers have settled, so each is written the way a site should be.
   Each is a kind of site somebody starts from with `quire new`: a simple site, a blog, documentation, a shop.
   The shop takes what is worth keeping from the toys project, which is no longer worked on.
   Each still teaches one thing more than the one before, and reads from top to bottom.

The Examples page with screenshots in step 10 depends on nothing before it.

## To decide

- **The footer's text and links.**
  They are the default frame's, yet the site holds them, because a pack and the site each add links to one footer and only the site can gather them.
  Either they stay as what any frame may show, as the nav's controls are, or the frame gathers them some other way.
- **Datastar on a site with nothing live.**
  Base asks for it on every page, because its theme toggle, palette picker and copy buttons are written with it. In plain script they would not need it, and an ordinary site would be 36KB lighter.
- **A widget against a language.**
  A site's widget beats a pack's quietly, but a widget named for a language the bake highlights is an error. The reason is mechanical: the fence's text is gone once it is highlighted.
- **A name for the helper package.**
  `app` for now.
- **A name for a site's description.**
  `quire.Describe` returns a `Description`: the identity, the blueprint and the options, which is collected over a baker and built over a bundle. The name is a placeholder.
- **The strapline.**
  "Composable parts for static and live sites in Go" is a placeholder, in the README, the site's description, the home page's kicker and the sample of it on the Packs page.
  The README's second line and the home page's title and lede still carry the older pitch.
- **"Diagram" in the docs.**
  The pages say a `mermaid` fence is a diagram, and the section is `#diagrams`.
  Either that stays as plain English, or the docs say mermaid and leave the word for charts.
- **Letting a site choose its font set.**
  The default is wired in today, in the site and in the bake.
  Once step 2 hands the set to both, choosing one is an option like the palette.
- **How the documentation site imports the examples.**
  Each is a `package main`, which nothing can import.
  Either the site's build generates importable copies and the examples stay as they are, or the examples become a package with a small `main` beside it and `quire new` flattens the two.
  Decide in step 10.
- **How a plugin reaches a service.**
  It asks its handle for one by type, or the site hands it to the plugin's constructor.
  Either way it is checked by the compiler, not looked up by reflection.
  Decide in step 5.
- **A name for the plugin's handle.**
  It is `plugin.Mount`, and mount means one thing.
  Decide in step 5.
- **Many sites from many repositories on one machine.**
  Entries cover sites built into one binary.
  Sites that ship as separate binaries would need a front that routes by hostname and starts each on its first request, perhaps as `quire host`.

## Alongside

These depend on none of the steps.

- **Feeds** — RSS/Atom. `Collection` already holds everything required.
  It would live in base once step 4 lands.
- **A crash in Go's compiler**, [golang/go#81627](https://github.com/golang/go/issues/81627), open with a fix in review.
  A generic function from another package, with its type inferred, crashes the compiler as an element of a composite literal: a struct field, or a slice or map element.
  Writing the type out, `ui.Listing[model.Meta]`, avoids it. Nothing in Quire needs the crashing form now; drop this note when a point release carries the fix.
- **archr on Quire's Lexis.** archr still carries a copy of the highlighter.
  1. **Tag a release** with Lexis in it, so archr can require it.
  2. **Repoint archr** and delete its copy.
     One `go.work` at archr's root covers it, its site and a local Quire.
     Its notices keep Chroma and Pygments, whose rules are in its binary, as coming by way of Quire.
     Its table of styles in `highlight.go` needs entries for `lexis.Operator` and `lexis.Punct`, or it indexes past the end.
- **Hand-written lexers for more of Lexis's languages.** [The benchmarks](./site/content/docs/benchmarks.md) list those run from rules slowest first; Java, Makefile and bash lead, and Rust and Lua are wanted.
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
- **Full-bleed and the scrollbar.** `.figure--bleed` measures in `cqw` against `main`, because `100vw` counts the scrollbar and would put a horizontal scrollbar on every desktop. The price is `container-type: inline-size` on `main`, whose layout containment makes it the containing block for any `position: fixed` descendant. Revisit against the two alternatives: hiding the document scrollbar in CSS (`scrollbar-width: none`, which costs every visitor their scroll position indicator), or setting the gutter width once from the classic script. Measured and written up in this session; no decision taken.
- **A bleed hero on a short screen.** Full width, a hero can fill a laptop screen with no text in sight. The usual answers: cap its height (50–60% of the screen, cropped with `object-fit: cover`) so text always shows below; a focal point per picture so the crop keeps what matters; different crops per screen size through `<picture>`. Height cap and focal point are options on the bleed figure; laying title and buttons over the picture is a site's layout choice. Tried with 884px art, which also showed the page padding left between nav and a bleed hero.

## Waiting on a step

Each of these is easier, or only possible, once the step it names is done.

- **After step 2: the baker is public.**

  - **Settings at the call site.** A picture's conversion is chosen where it is placed, in the markdown or the template: lossless, a quality, a second format such as AVIF, a shorter ladder. Variant names already cover every setting and a record already lists formats, so the bake's `Master` takes the options and nothing else moves. Every variant is lossy WebP until then.
    A second format is then a codec a pack can bring.
  - **A file of source as a code block.** A document names a file and the bake puts its text on the page, highlighted, whole or a range of its lines.
    The file may be outside the content folder, as Lexis's corpus is, so the bake has to read it and count it as an input, to bake the page again when it changes.
    Lexis's languages page is then a document written by hand, and `just lexis languages` goes.
  - **A ladder from use.** The bake renders its pages too, records which rungs each placement can pick, and encodes only those rather than the whole ladder for every master. It is the same record as the call-site settings.

- **After step 5: there is a log.**

  - **Logging.** None yet. When there is a logger, these are the gremlins to warn on.

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

- **After step 7: what is kept can be dropped.**

  - **Dev re-bakes in-process.** A site already bakes before it serves; with `--dev` it could also re-bake when content or masters change, so a watcher is only for Go and templ. A CMS needs the same thing: re-bake after an upload and swap the bundle, rather than a second bundle beside the first.
    Swapping a bundle is replacing an entry's and dropping what was kept from the old one.

- **After step 8: a site can be rooted at a path.**

  - **Pre-rendered pages.** The bake runs the blueprints, so after the make a server over the new bundle can render every page the page cache would render once for everyone into the bundle, as Astro's `dist/` and Hugo's `public/` do, and the shipped binary serves finished HTML. Plugin routes and anything per visitor stay live, the per-route split Astro and Next make. It is a third axis beside `quire_nobake` and the ship overlay: whether its pages are pre-rendered (a light site's would be). A site with no live routes at all needs no server, only its bundle on a static host. Documents are already converted in the bake, so the gain is boot time and static hosting rather than weight.
    A static host often serves under a path, which is what the root is for.
  - **Public-bucket URLs, in the markup.** Emit the URL of variants somebody else serves in `srcset` so the browser never asks the origin at all. Edge-cached, no egress charge. The constraint is unchanged: every referenced variant must be in the bucket before it is referenced.
    A picture's URL is built in the same place the root is applied.
