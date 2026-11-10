---
title: The look
summary: Palette, fonts, scale, tokens, and the CSS, components and pages a site starts with.
---

Each layer is complete on its own and replaceable without touching the others.
`Shell` is only the last composition; dropping a layer is not a rewrite.

## Palette

A `Palette` names the Radix hue behind each role; default Ember, or `WithPalette(PaletteMoss)` (also Ink, Dusk), or a palette of your own.
Each role is a twelve-step scale: `--qr-neutral-*` for surfaces, borders and text, `--qr-accent-*` for the brand and everything interactive, `--qr-spot-*` for a second brand colour, `--qr-swatch1-*`…`4` for telling things apart, and `--qr-red-*` `--qr-green-*` `--qr-amber-*` for meaning.
A step's number is its job: 1–2 backgrounds, 3–5 component backgrounds, 6–8 borders, 9–10 solid fills, 11–12 text; `--qr-<role>-contrast` is the text on a step-9 fill.

## Fonts

Inter for text and JetBrains Mono for code by default, served as woff2 split by unicode subset; the latin text face is preloaded.
Sheets read `--qr-font` and `--qr-font-mono`.

## Scale

`Root`, `Col`, `Widest`. Gutter and thumb are rem from Root. A `.page` is two columns of Col, `.page--wide` three, `.page--full` four; Col defaults to 336.
`Widest` is the widest page the site builds (`WidthWide`, `WidthFull`; zero is a plain `.page`). A page token past it is not emitted, so `.page--wide` on a site that never declared it stays a `.page`.
It gates the CSS only: every image offers every slot.
`WithScale` sets all three; page widths, the breakpoint and the slot sizes follow.
Host CSS for a step is `WithStylesheetsAt(WidthNarrow|WidthWide|WidthFull, …)` — Scale writes the query.
Read the properties with `var()`; never restate the number.

## Tokens

`--qr-text-xs`…`xl` `--qr-space-*` `--qr-weight-*` `--qr-lead-*` `--qr-track-*` `--qr-measure-text` `--qr-section-gap` on `:root` in `base.css`.
`--qr-section-gap` is the space between a page's sections; a rule dividing them pads by it, so the space either side matches.
Override the custom properties.
`--qr-bg` is the colour behind whatever is drawn there: a surface that paints its own background sets it (as `.callout` does), and diagrams read it to mask lines behind their labels.
`.prose` is a breakout grid: body text holds `--qr-measure-text` on any page, a `.figure` or `.diagram` in it spans the page inner.

## Primitives

`base` `toggle` `nav` `layout` `prose` `highlight`, in that order.
Site CSS composited last overrides them.
`.kicker` is the small accent label cards and entries use for their dates; put it on any element.

## Components

Package `ui`, as is everything below.


`Card` `Entries` `HeroFigure` `Contents` `DraftBadge` `Badges` `Tags` `BackLink` `ForwardLink`.
`DocCard(c, d)` fills a `Card` from a document: its link, the collection's kicker, title, summary, draft and featured flags, image, badges and tags. A site changes the fields it wants different.

## Pages

`Home` `Listing` `Article` `NotFoundPage` `Footer`. `AddPage("/", …)` replaces `Home`, `SetNotFound` replaces `NotFoundPage`, and `WithFooter` replaces `Footer`.
Set `Spec.Index` / `Spec.Item` to replace `Listing` / `Article`. Each is handed the shell, so a template with the same signature as the default is named directly: `Index: projectsPage`.

## Document

`e.Shell(PageSpec)`.
An active nav link matches `/` only as itself, and any other path as its subtree on a boundary, so `/project` does not light `/projects`.
The table of contents lists H2 and H3 only.
