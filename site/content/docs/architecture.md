---
title: Design
summary: Why Quire is shaped this way.
---

The other pages say how to use Quire. This one says why.

## Scale

The column is 336 because that is the widest one that still fits two of them on a 744px tablet, the narrowest worth keeping off the phone layout.

`Widest` gates page tokens only. How wide an image goes is where it is placed, so one wide page does not make every thumbnail pay for that width.

`sizes` sees the viewport, not the container. Between the two-column width and the three-column width a `.page` is already capped and a `.page--wide` is still growing, so the cells differ in that band.

`SlotBleed` measures in `cqw` against `main`. `100vw` counts the scrollbar.

A sheet reads the scale's variables and never writes a width query. `WithStylesheetsAt` attaches CSS to a step.

`--qr-measure-text` is `ch`, because a comfortable line belongs to the font. `.prose` is a breakout grid so a figure is not capped with the text.

Every custom property is `--qr-*`. A plugin uses its own prefix. Overriding a `--qr-*` token is a tweak, the same act as `WithScale`.

## One mount

Dev may run a step lazily. It may not skip one. A variant is in a journal or it is an error, in every mode.

Bake and serve run the same mount. The folder is not a second answer to what the site contains. A file reaches the bundle because the mount read it or referred to it. `AddFile` ships anything else.

A bake reads the source tree. What that mount renders is thrown away: source order is the measured bytes, and those exist only after the encode.

The bake converts and serving composes. Markdown, masters and mermaid are converted once, by the bake; a template places what came out. A document reaches the bundle as HTML with a hole for each picture, diagram and site widget, and serving fills each hole: a picture from the journal with the markup `ui` makes today, a widget by running it. So a change to the scale, to `ui` or to a widget needs no rebake, and one master referred to twice is one embed.
Everything the bake makes is one kind of thing: a journal record, keyed by what it was made from, naming content-addressed files. A document, a diagram and a share card are each one file; a master is its ladder. One check decides what is remade, one copy decides what ships, and one lookup finds it again.
Something that must convert per request is a plugin that imports its converter.

## Journal

A journal is an authority. One that will not write fails the build. It is written whole and renamed into place. A variant is one content-addressed file, so a crash leaves it whole or absent.

The bundle is linked from the cache. Nothing edits a bundle file in place. Each `WithBundle` is one journal, and the first to name a variant owns it.

A key holds everything its output depends on: the input, the settings, and the converter's version. The versions sit in one table and change by hand, so a refactor rebuilds nothing; a release check fails on any converter whose code or dependencies moved since its version was set.

`quire_nobake` swaps the bake for a stub, so a serving binary cannot convert: no codec, no markdown, no highlighter.
Where the bundle lives is a separate choice. A site's `ship.go`, under its own `quire_embed` tag, embeds it and hands it to `Ship`; without one, the bundle is read off disk. A shipped static site sets both tags, a CMS may embed and still bake, and a binary can serve a bundle mounted beside it.

## Packages

Layers only point down, and a test holds them to it.

`model` is the data. Of Quire's it imports only the bundle format (`internal/journal`) and the colour scales.
`ui` and `plugin` import `model` and nothing else of Quire's.
`quire` is what a site calls. A site or a command is the only importer.

A plugin sits beside `ui`, and a site opts in with `Use`.

## Colour, fonts, diagrams

A palette is roles. Sheets read the role, never a hue, and do not mix tints. The scales are Radix, because the light and dark pairs are already tuned. A test holds text contrast.

A browser face is woff2. Measuring text needs TTF, and only a bake measures, so the two files stay apart. The engine writes `--qr-font` and `--qr-font-mono`.

Mermaid is the diagram format editors already preview. The renderer is heavy, so it lives on the bake side. A diagram is inlined so it can see the page's custom properties. Its name includes the font set, so a bake and a site on different fonts fail the load.

## Layers

Base is the engine and the default look. `examples/1-minimal` is the proof. A template pack is opt-in and is not the base. A site is not a pack.

Quire ships a class only if it ships the rules for it.

## Serving

What happens to a response follows the response, not the mode.

A fixed page is rendered once when `Handler` is taken. A collection item is cached under its path and coding. Anything else goes straight to its handler. The cache key has no method and no cookie, so a stored body is the same for every visitor.

`Handler` finalises. A route added after it panics.
