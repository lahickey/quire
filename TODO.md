# Quire — plan

What is not built yet, what is still to decide, and what is waiting.
The shape Quire has, and why, is [the design page](./www/content/docs/architecture.md).

## Not built yet

Two parts of the shape on the design page are not built. What is said of each holds, so that it can be added.

### Events

A bus lets a feature say what happened without knowing who acts on it: a basket publishes an order, and fulfilment and mail each listen.
It is a service like any other:

- **Subscribing is registering**, like adding a route. A feature subscribes in `Declare` and a service in its constructor. The bus keeps handlers before it starts and delivers only while it runs, so a bake delivers nothing.
- **It is added first**, so it has started before anything publishes.
- **An event is data**: a subject and a value that can be written out, never a pointer or a function. A bus in memory and one over NATS are then the same to whoever uses them.
- **Delivery is at least once, in no order, and not while the publisher waits.** A handler takes a repeat without harm. A bus in memory does better, and nothing relies on it.
- **Subjects are NATS's.** Dotted tokens that begin with the publisher's name, `basket.ordered`, and a listener filters as NATS does: `basket.*` for one token, `basket.>` for all that follow. The bus is NATS or something that mimics it.
- **Events only.** Something that needs an answer is handed the service and calls it.

### Caching

A route states how its response may be kept. Two ways are built: shared, one body for every visitor, and not at all.

- **Per group**: a group has a cache of its own, inside its gate, so what is kept for admins is never served to anyone else. It waits on a gate, which is sign-in.
- **Per visitor**: keyed by the session, in a budget of its own so it cannot push out shared pages. Only when something measurably needs it: a shared page with its personal part filled live, as the basket link is, covers most cases.
- **Dropping what is kept**: whatever changes what a kept page shows can drop it. Nothing is dropped today, because nothing can change while a site runs.

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
