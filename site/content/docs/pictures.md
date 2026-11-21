---
title: Pictures
summary: From a picture beside a document to responsive WebP, and how it is served.
---

You put a picture beside a document and refer to it. Quire does the rest: it encodes the picture at several widths, writes the markup that lets the browser choose one, and serves the files.

## Using a picture

There are two ways to refer to one, and both go through the same pipeline.

```markdown
---
title: The ridge at first light
picture: ridge.jpg
---

![The ridge at first light](ridge.jpg "Dawn on the ridge")
```

- `picture:` in the frontmatter is the document's picture: its thumbnail in listings, the hero on its page and its share image.
- `![](…)` in the body is a figure in the text.

The body line above is this figure. Its `srcset` offers each width the bake made, and the browser fetches one.

![The ridge at first light](ridge.jpg "Dawn on the ridge")

A picture inside a sentence, such as ![a small ridge](ridge.jpg), stays in the line and takes no caption.

The markup carries `width` and `height`, so nothing shifts while a picture loads.
In a template of your own, place a document's picture with a component:

```templ
@ui.Img(doc.Picture, model.SizePage, title)
```

- `ui.Img(picture, size, alt)` is the picture alone.
- `ui.LazyFigure(picture, alt, caption)` is a figure at page width that loads when it nears the screen. A picture in a document's body is one of these.
- `ui.HeroFigure(picture, size, alt, caption)` is a figure fetched at high priority, for the picture at the top of a page. It draws nothing when the document has no picture.

## Recordings

A video is placed with the same line as a picture, and is a figure with the same caption.

```markdown
![The bake of an example site](bake.webm "An example baking")
```

![The bake of an example site](bake.webm "An example baking")

- The file is WebM or MP4, and the bake copies it into the bundle as it is. Nothing is decoded or encoded.
- A still of the same name is its poster: `bake.png` beside `bake.webm`. The poster goes through the picture pipeline, and the video takes its `width` and `height` from it. A video with no poster fails the bake.
- A video goes in a paragraph of its own. One inside a sentence fails the bake.

It plays muted, in a loop, while it is on screen, and stops when it is scrolled away.
A reader who has asked their system for reduced motion sees the poster with the browser's controls, and so does one without scripts.

In a template of your own, `ui.VideoFigure(src, poster, alt, caption)` is the same figure.

## Masters

A picture content refers to is a **master**, and it lives beside the documents that use it.
The bake encodes it; the site embeds the bundle and never a master.
A master is JPEG, or lossless WebP when it has alpha. Every variant is lossy WebP.

Most sites commit their pictures as they are and never think about this.

> [!CAUTION]
> A picture committed by hand keeps its EXIF, XMP and ICC profile in the repository, GPS from a phone included. Quire strips them only from what it writes, so run a phone picture through `quire ingest` first.

For a source too big to commit, `quire ingest` writes a master from it: 8-bit, long edge at most `max(2560, widest size)`, EXIF/XMP/ICC stripped, and nothing beside it.
A source already under the cap is not upscaled, and shrinking one says so.

```sh
go run github.com/lahickey/quire/cmd/quire ingest art eight-thirty ./press.tif
```

## Sizes

A size says how wide a picture is shown where it is placed.
Sizes are widths, not roles: `thumb`, `column`, `page`, `page-wide`, `page-full`, `bleed`, narrowest first, and the set is closed.
A card is rendered at a column and a hero at a page; the template says where, and the size's name says how wide.

`HeroFigure(picture, SizeBleed, …)` is the whole of running one to the window edge, with fetch priority.

## The ladder

```mermaid
flowchart LR
  master[master] --> bake
  bake --> rungs[a rung per width, in WebP]
  rungs --> thumb[thumb: three rungs]
  rungs --> page[page: more of them]
  rungs --> bleed[bleed: all of them]
```

The ladder is one global list of widths from a thumbnail up to the cap. Nothing about a layout reaches it: a width taken from the grid is wrong as soon as a site's own sheet changes the grid.
A picture offers the rungs below its master plus the master itself, so art authored at an exact width is served at it and nothing is upscaled.
A size picks which rung `src` lands on and where `srcset` stops, so a 4rem thumb lists three rungs, not ten. Every picture with the same master encodes the same files.

## Share images

A page's title and summary are its `og:title` and `og:description`. Its share image is the frontmatter `picture:` at the widest rung no wider than a share card.
Every page without a picture of its own shares the site card: one PNG the base pack asks the bake to draw, with the site's name and description on the palette's dark ink unless `WithSiteCard` says otherwise.

## In depth

### How a picture is served

Placing a picture looks its master up in the bundle's journal and mints a URL per size: `/_quire/img/{key}.{ext}`. It opens no file.
A binary with no journal entry for a master cannot place it, and says so at boot rather than 404ing a request later.

Bytes come from the bundle's files, streamed.
A video is served from the same address, and a request for a range of it gets that range.
Files up to 64KB are held in a 12MB memory store.
An unknown name 404s, and so does a known one the bundle does not hold. A name is checked against the shape of a variant's name first, so a probe never reaches a directory.

### The widths

The ladder's rungs are 64, 128, 232, 408, 680, 1096, 1704 and 2560 pixels.
A rung within 15% of the master's own width is dropped, so the same picture is not encoded twice at almost one size.

### What can go wrong with a master

A bake reads whatever master content refers to and corrects it in memory, so none of these breaks a page. Each costs work on every bake, and `quire ingest` settles it once.

- A master over the cap is fitted in memory every bake, and the committed file is bigger than anything that will be served from it.
- A master with an EXIF orientation flag is rotated in memory each time.
- A master added by hand keeps its EXIF, XMP and ICC profile in the repository, GPS from a phone included. Quire strips them only from what it writes.
- A master narrower than the widest size that uses it is served at its own width. Nothing breaks; the picture is just soft on a dense screen.
