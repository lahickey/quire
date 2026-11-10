# Quire — backlog

One list. The model is [the design page](./site/content/docs/architecture.md).

## Next

- **Feeds** — RSS/Atom. `Collection` already holds everything required.

## Later

- **Public-bucket URLs, in the markup.** Register a journal of variants somebody else serves, and emit their URL in `srcset` so the browser never asks the origin at all. Edge-cached, no egress charge. The constraint is unchanged: every referenced variant must be in the bucket before it is referenced.
- **Full-bleed and the scrollbar.** `.figure--bleed` measures in `cqw` against `main`, because `100vw` counts the scrollbar and would put a horizontal scrollbar on every desktop. The price is `container-type: inline-size` on `main`, whose layout containment makes it the containing block for any `position: fixed` descendant. Revisit against the two alternatives: hiding the document scrollbar in CSS (`scrollbar-width: none`, which costs every visitor their scroll position indicator), or setting the gutter width once from the classic script. Measured and written up in this session; no decision taken.
- **A bleed hero on a short screen.** Full width, a hero can fill a laptop screen with no text in sight. The usual answers: cap its height (50–60% of the screen, cropped with `object-fit: cover`) so text always shows below; a focal point per image so the crop keeps what matters; different crops per screen size through `<picture>`. Height cap and focal point are engine options on the bleed figure; laying title and buttons over the picture is a site's layout choice. Tried with 884px art, which also showed the page padding left between nav and a bleed hero.
- **Logging.** None yet. When there is a logger, these are the gremlins to warn on.

  **A master in bad shape.** A bake reads whatever master content refers to and corrects it in memory, so nothing here breaks a page — it costs work on every changed master, and it costs the chance to copy rather than re-encode.
  Only `quire ingest` writes a master, so every one of these says the same thing: run it, or fix the file at the source.

  - **Master over the cap.** Every bake fits it in memory before it can measure or encode, and the committed file is bigger than anything that will ever be served from it. Expensive if it is a print original somebody added without thinking.
  - **Master with an EXIF orientation flag.** The bake rotates it in memory each time, and the widest rung can no longer be copied from it, so that rung is re-encoded for nothing. Rotating once at ingest settles it.
  - **Master still carrying EXIF, XMP or an ICC profile.** Quire strips what it writes, but a file added by hand is committed, embedded and served as it is — GPS from a phone included. Nothing warns today, and nothing else will notice.
  - **Master in a container no codec uses.** A PNG master on a WebP-only site works: it is decoded and every rung encoded. But the widest rung can never be a copy, and the repo carries a file several times the size of anything served.
  - **Master with a non-sRGB ICC profile.** Quire strips profiles, and no variant format it emits could carry one anyway. The pixels are then read as sRGB, so a Display P3 or Adobe RGB original renders oversaturated and nobody can tell it was avoidable. Convert before ingest.
  - **Master narrower than the widest slot that uses it.** The ladder stops at the master, so nothing breaks — the hero is just soft, and nobody finds out until they look at it on a dense screen.

  **Everything else.**

  - **Bake of hundreds of images into a FROM-scratch binary.** A bake writes one file per rung per format — seven rungs of one codec is seven files an image, and three codecs is twenty-one. Tens of images is fine. Hundreds makes the artefact fat and the build slow — the person will notice; a warning should say so.
  - **A slot wider than the 2560 floor.** The cap follows the widest slot, so one slot silently makes every master on the site bigger. Intended when someone wants a full-bleed hero on a large desktop; an accident when a width was typed with an extra digit.
  - **A lossless variant over a few hundred KB.** Lossless art lands in the tens of KB. A large one means a rule keeps a photograph lossless, at several times the bytes of the lossy ladder.
  - **A public-bucket URL for a variant the bucket does not hold.** Once the browser is sent to the bucket, the engine is out of the loop: the visitor gets a broken image. The bake did not run, or it ran with different codecs.

- **First public release.** Create `github.com/lahickey/quire` empty, with `master` as its default branch. Make one root commit of the tree, keeping the old history under a tag. Install `govulncheck`, `git remote add public git@github.com:lahickey/quire.git`, then `just release v0.1.0` and run the push it prints.
- **`quire new`.** Generate a site, naked or from a starter: `go.mod`, a `site.go` around `quire.Run`, `content/`, `media/`, and a Dockerfile already building with `-tags quire_nobake -overlay .quire/ship.json`. The conventions every site repeats live in one place instead. A starter is a template pack, so the rule under Parked holds.
- **Dev re-bakes in-process.** A site already bakes before it serves; with `--dev` it could also re-bake when content or masters change, so a watcher is only for Go and templ.
- **Pre-rendered pages.** The bake mounts the site, so after the make a serving engine over the new bundle can render every page the page cache would render once for everyone into the bundle, as Astro's `dist/` and Hugo's `public/` do, and the shipped binary serves finished HTML. Plugin routes and anything per visitor stay live, the per-route split Astro and Next make. It is a third axis beside `quire_nobake` and the ship overlay: whether its pages are pre-rendered (a light site's would be). A site with no live routes at all needs no server, only its bundle on a static host. Documents are already converted in the bake, so the gain is boot time and static hosting rather than weight.
- **A ladder from use.** The bake renders its mount too, records which rungs each placement can pick, and encodes only those rather than the whole ladder for every master. Per-placement settings (lossless, a size) are the same record.

## Parked

Do not start a template pack by lifting a live site into Quire.
A pack is opt-in, bringing its own Go and CSS, and whichever site motivates it is not the pack.
`examples/` has `1-minimal`; a pack would bring its own.
The justfile already takes a name.
