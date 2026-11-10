---
title: Images
summary: Masters, slots, the ladder, and how a picture is served.
---

## Masters

A picture content refers to is a **master**, and it lives beside the documents that use it.
The bake encodes it; the site embeds the bundle and never a master.
A master is JPEG, or lossless WebP when it has alpha. Every variant is lossy WebP.

`quire ingest` writes one from a source too big to commit: 8-bit, long edge at most `max(2560, widest slot)`, EXIF/XMP/ICC stripped, and nothing beside it.
Most sites will not use it.
A source already under the cap is not upscaled; shrinking says so.

```
go run github.com/lahickey/quire/cmd/quire ingest art eight-thirty ./press.tif
```

## Slots

Slots are widths, not roles: `thumb`, `column`, `page`, `page-wide`, `page-full`, `bleed`, narrowest first, and the set is closed.
`HeroFigure(im, SlotBleed, …)` is the whole of running one to the window edge, with fetch priority.

```templ
@ui.Img(doc.Image, model.SlotPage, title)
```

## The ladder

The ladder is one global list of widths from a thumbnail up to the cap. Nothing about a layout reaches it: a width taken from the grid is wrong as soon as a site's own sheet changes the grid.
An image offers the rungs below its master plus the master itself, so art authored at an exact width is served at it and nothing is upscaled.
A slot picks which rung `src` lands on and where `srcset` stops, so a 4rem thumb lists three rungs, not ten; every image with the same master encodes the same files.

## Serving

Mounting an image looks its master up in the bundle's journal and mints a URL per slot: `/_quire/img/{key}.{ext}`. It opens no file.
A binary with no journal entry for a master cannot mount it, and says so at boot rather than 404ing a request later.

Bytes come from the bundle's files, streamed.
Files up to 64KB are held in a 12MB memory store.
An unknown name 404s, and so does a known one the bundle does not hold.

## Share images

A page's title and summary are its `og:title` and `og:description`; its image is the frontmatter `image:` at the widest rung no wider than a share card.
Every page without an image of its own shares the site card: one PNG the bake draws, the site's name and description on the palette's dark ink unless `WithSiteCard` says otherwise.
