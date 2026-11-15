# Quire — plan

Where Quire is going, the steps that get it there, and what else is waiting.
The model as it is today is [the design page](./site/content/docs/architecture.md).

## Where it is going

The engine today is two things in one type: the description of a site, and the server.
A bake makes one with no server in it, and serving makes another.
The plan separates the parts so that each has one job and a public interface, and so that nothing Quire ships is a special case.

### The parts

- **Source.** The requests a site makes for content: documents, a picture, the output of a converter. Everything below either answers them or asks them.
- **Baker.** A source that converts. It writes a journal and content-addressed files into a cache, and knows nothing of sites, routes or servers.
- **Bundle.** A journal and the files it names. It is a source that only reads.
- **Site.** What a mount builds over a source: collections, pages, chrome, files. It has no server.
- **Engine.** The server. It holds entries, each a root, a bundle and a site, and owns the mux, the caches and the services' lifecycle.

### The verbs

1. **Collect.** Run the mount over a baker, so the site asks for what it needs.
2. **Bake.** The baker converts and journals.
3. **Bundle.** Copy out what was referred to, and write `ship.go`.
4. **Serve.** Build the site over its bundle and attach it to an engine.

Collecting links the site's Go, because a mount reads its documents while it declares.
So bake and serve are separate packages and phases, not programs that never meet.
A CMS skips collecting and makes the baker's requests from its own logic.

### What a feature is made of

| Role | Runs | Does |
|-|-|-|
| Converter | at bake | turns input into stored output |
| Widget | at page render | places output in the page |
| Plugin | while serving | routes, streams, sessions |

- A plugin is for live features only.
- A converter has a version of its own, in the key of everything it makes.
- A converter sits in a package of its own, so a serving binary that imports the widget and the plugin does not link it.
- A fence may need all three: a live chart converts at bake, is placed by a widget, and is updated by a plugin.
- A plugin is attached per site. One instance attached to several sites shares whatever state it keeps.

### Services

A service is something a plugin needs and the site provides.
The plugin names the interface and the site chooses what is behind it: a store in memory for a test, SQLite for a small site, the site's own for anything else.

- **One interface, with a lifecycle.** The engine starts services in the order they were added and stops them in reverse.
- **Reached through the handle.** A plugin asks the handle it is attached with, which also gives it a logger carrying its name. A service is handed the same.
- **Kept apart by name.** A plugin's data is under its own name, as its routes and classes already are, and under its site's when an engine holds several.
- **Missing is an error at boot.** A plugin that needs a store on a site with none stops the site starting. Nothing falls back to memory unasked.
- **An implementation is a package of its own**, so a site links only the one it chose.

The standard ones are a store, sessions and a log.
Sessions are a service because they have no routes or pages: they are who the visitor is, and plugins ask.
The log is the one the baker's warnings go to as well.
Others, such as mail, take the same shape.

The interfaces are right when a basket, a sign-up form, comments and sign-in could each be written against them and nothing else.

### Caching

A route states how its response may be kept, and the engine does not decide by what kind of route it is.

- **Shared**: one body for every visitor, as a page is today. A plugin's route can ask for it.
- **None**: as a plugin's route is today.
- **Per visitor**: keyed by the session, in a budget of its own so it cannot push out shared pages. Only when something measurably needs it: a shared page with its personal part filled live, as the basket link is, covers most cases.

Whatever changes what a kept page shows can drop it.
Nothing is dropped today, because nothing can change while a site runs.

### Packs

A pack provides any number of converters, widgets, plugins, services, components, stylesheets, fonts and palettes, and optionally a mount.
Changes to the base colour system are a pack's to make too.
A pack with a mount is a whole kind of site, as `pack/docs` is.

What Quire ships is two packs, under working names:

- **Core** cannot be turned off: documents from markdown, and the picture record with its ladder. Everything else fills the holes these leave or reads what they record.
- **Base** can: mermaid, the share card, the default look and its components, the default fonts and palettes, search, sessions and the store in memory.

Both are written against the interfaces any other pack uses, and a test holds them to it.
Turning something off is not registering it, never a flag: what is not imported is not linked.
A helper gives a site the standard set, so the smallest site stays as small as it is.

### One engine, several sites

An engine with one entry at `/` is a site as it is today.
Several entries are several sites, or one site in parts, each under its own root and each with its own bundle or sharing one.
A site with a root other than `/` can also be mounted in somebody else's server, beside routes Quire does not manage, or sit behind a proxy at a path.

## The steps

Each step leaves everything working and is worth having if nothing after it happens.

1. **Split the engine into site and server.**
   Building a site over a source becomes one thing and serving it another.
   A bake builds a site and never makes an engine.
   No behaviour changes, and the existing tests hold it.
2. **Make the source, the baker and the bundle public.**
   The engine stops importing the bake.
   Converters become a public interface, and mermaid, the share card and the picture codec move onto it.
   `Run` becomes a helper that composes the parts.
3. **Declare widgets once for a site.**
   They are per collection today, and the bake is told fence names so it knows which holes to leave.
   A pack that brings a fence declares it once.
4. **Packs provide the parts.**
   A pack can carry converters, widgets, plugins, components, stylesheets, fonts and palettes.
   What the engine does as a special case moves into core and base: the mermaid and picture cases where holes are filled, the share card, and the default home page, index, article and not-found page.
   The layer test grows a rule: core and base import only what an outside pack could.
5. **Services.**
   The interface and its lifecycle, the handle that gives a plugin its services and its logger, and a store with an implementation in memory.
   Sessions move out of the engine and become one, which needs a way to wrap a request that plugins do not have today.
6. **Give the engine entries.**
   Several of root, bundle and site.
   `Ship` registers bundles by name, and one `ship.go` embeds every bundle under `.quire`.
7. **Let a route state its cache policy.**
   Shared or none, on any route, and a way to drop what is kept.
8. **Root a site at a path.**
   The root is prepended wherever a URL is built, and a rendered page has its root-absolute URLs prefixed before it is cached, so a hand-written `/writing` is still right.
   A site at `/` skips all of it.
   The session cookie and the saved theme and palette are scoped to the root.
   `examples_test.go` serves every example a second time under a root and fails on any URL that escapes it.
9. **Show the examples on the documentation site.**
   An Examples page with a screenshot of each, what it adds, its `quire new` line and its source.
   Then each example live, as an entry under `/examples/`.
10. **Tidy the helper layer.**
   `Run`, `model.Basic` and the other shortcuts sit together above the parts, and say which parts they compose.

The Examples page with screenshots in step 9 depends on nothing before it, and nor does making the examples look their best.

## To decide

- **Two entry points, or `quire_nobake`.**
  A serving `main` that does not import the baker links no codecs, with no build tag, and it is the only way to keep a pack's converter out of a serving binary without the pack's author tagging their files.
  The price is a site with one `main` to bake and one to serve.
  Leaning to two entry points; decide in step 2.
- **Names for core and base.**
- **How the documentation site imports the examples.**
  Each is a `package main`, which nothing can import.
  Either the site's build generates importable copies and the examples stay as they are, or the examples become a package with a small `main` beside it and `quire new` flattens the two.
  Decide in step 9.
- **How a plugin reaches a service.**
  It asks its handle for one by type, or the site hands it to the plugin's constructor.
  Either way it is checked by the compiler, not looked up by reflection.
  Decide in step 5.
- **Many sites from many repositories on one machine.**
  Entries cover sites built into one binary.
  Sites that ship as separate binaries would need a front that routes by hostname and starts each on its first request, perhaps as `quire host`.

## Alongside

These depend on none of the steps.

- **Feeds** — RSS/Atom. `Collection` already holds everything required.
  It would live in base once step 4 lands.
- **lexis in place of chroma.** archr's highlighter: faster, fixes edge cases chroma has, and reads templ properly.
- **Full-bleed and the scrollbar.** `.figure--bleed` measures in `cqw` against `main`, because `100vw` counts the scrollbar and would put a horizontal scrollbar on every desktop. The price is `container-type: inline-size` on `main`, whose layout containment makes it the containing block for any `position: fixed` descendant. Revisit against the two alternatives: hiding the document scrollbar in CSS (`scrollbar-width: none`, which costs every visitor their scroll position indicator), or setting the gutter width once from the classic script. Measured and written up in this session; no decision taken.
- **A bleed hero on a short screen.** Full width, a hero can fill a laptop screen with no text in sight. The usual answers: cap its height (50–60% of the screen, cropped with `object-fit: cover`) so text always shows below; a focal point per image so the crop keeps what matters; different crops per screen size through `<picture>`. Height cap and focal point are engine options on the bleed figure; laying title and buttons over the picture is a site's layout choice. Tried with 884px art, which also showed the page padding left between nav and a bleed hero.

## Waiting on a step

Each of these is easier, or only possible, once the step it names is done.

- **After step 2: the baker is public.**

  - **Settings at the call site.** A picture's conversion is chosen where it is placed, in the markdown or the template: lossless, a quality, a second format such as AVIF, a shorter ladder. Variant names already cover every setting and a record already lists formats, so the bake's `Master` takes the options and nothing else moves. Every variant is lossy WebP until then.
    A second format is then a codec a pack can bring.
  - **A ladder from use.** The bake renders its mount too, records which rungs each placement can pick, and encodes only those rather than the whole ladder for every master. It is the same record as the call-site settings.

- **After step 5: there is a log.**

  - **Logging.** None yet. When there is a logger, these are the gremlins to warn on.

    **A master in bad shape.** A bake reads whatever master content refers to and corrects it in memory, so nothing here breaks a page — it costs work on every changed master, and it costs the chance to copy rather than re-encode.
    Only `quire ingest` writes a master, so every one of these says the same thing: run it, or fix the file at the source.

    - **Master over the cap.** Every bake fits it in memory before it can measure or encode, and the committed file is bigger than anything that will ever be served from it. Expensive if it is a print original somebody added without thinking.
    - **Master with an EXIF orientation flag.** The bake rotates it in memory each time, and the widest rung can no longer be copied from it, so that rung is re-encoded for nothing. Rotating once at ingest settles it.
    - **Master still carrying EXIF, XMP or an ICC profile.** Quire strips what it writes, but a file added by hand is committed, embedded and served as it is — GPS from a phone included. Nothing warns today, and nothing else will notice.
    - **Master that is neither JPEG nor WebP.** A PNG master works: it is decoded and every rung encoded. But the widest rung can never be a copy, and the repo carries a file several times the size of anything served.
    - **Master with a non-sRGB ICC profile.** Quire strips profiles, and no variant format it emits could carry one anyway. The pixels are then read as sRGB, so a Display P3 or Adobe RGB original renders oversaturated and nobody can tell it was avoidable. Convert before ingest.
    - **Master narrower than the widest slot that uses it.** The ladder stops at the master, so nothing breaks — the hero is just soft, and nobody finds out until they look at it on a dense screen.

    **Everything else.**

    - **Bake of hundreds of images into a FROM-scratch binary.** A bake writes one file per rung per format, seven for a full ladder. Tens of images is fine. Hundreds makes the artefact fat and the build slow — the person will notice; a warning should say so.
    - **A slot wider than the 2560 floor.** The cap follows the widest slot, so one slot silently makes every master on the site bigger. Intended when someone wants a full-bleed hero on a large desktop; an accident when a width was typed with an extra digit.
    - **A public-bucket URL for a variant the bucket does not hold.** Once the browser is sent to the bucket, the engine is out of the loop: the visitor gets a broken image. The bake did not run.

- **After step 7: what is kept can be dropped.**

  - **Dev re-bakes in-process.** A site already bakes before it serves; with `--dev` it could also re-bake when content or masters change, so a watcher is only for Go and templ. A CMS needs the same thing: re-bake after an upload and swap the bundle, rather than a second bundle beside the first.
    Swapping a bundle is replacing an entry's and dropping what was kept from the old one.

- **After step 8: a site can be rooted at a path.**

  - **Pre-rendered pages.** The bake mounts the site, so after the make a serving engine over the new bundle can render every page the page cache would render once for everyone into the bundle, as Astro's `dist/` and Hugo's `public/` do, and the shipped binary serves finished HTML. Plugin routes and anything per visitor stay live, the per-route split Astro and Next make. It is a third axis beside `quire_nobake` and the ship overlay: whether its pages are pre-rendered (a light site's would be). A site with no live routes at all needs no server, only its bundle on a static host. Documents are already converted in the bake, so the gain is boot time and static hosting rather than weight.
    A static host often serves under a path, which is what the root is for.
  - **Public-bucket URLs, in the markup.** Emit the URL of variants somebody else serves in `srcset` so the browser never asks the origin at all. Edge-cached, no egress charge. The constraint is unchanged: every referenced variant must be in the bucket before it is referenced.
    The image URL is built in the same place the root is applied.
