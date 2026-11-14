---
title: The look
summary: Change the colours, fonts and CSS a site starts with, or replace a component or a page.
---

A new site already has a look: a palette, fonts, a page grid, a stylesheet, components and pages.
Each of those is a layer. Each is complete on its own and replaceable without touching the others, so dropping a layer is not a rewrite.

This page goes from the smallest change to the largest.

## Change the colours

Pick another palette with an option:

```go
quire.Run(site, mount, quire.WithPalette(model.PaletteMoss))
```

The built-in palettes are Ember (the default), Moss, Ink and Dusk. `WithTheme(model.ThemeDark)` makes dark the scheme a visitor sees when neither they nor their system has chosen one.

A `Palette` names the Radix hue behind each role, so a palette of your own is a struct of hue names.
Each role is a twelve-step scale:

- `--qr-neutral-*` for surfaces, borders and text;
- `--qr-accent-*` for the brand and everything interactive;
- `--qr-spot-*` for a second brand colour;
- `--qr-swatch1-*`…`4` for telling things apart;
- `--qr-red-*`, `--qr-green-*` and `--qr-amber-*` for meaning.

A step's number is its job: 1–2 backgrounds, 3–5 component backgrounds, 6–8 borders, 9–10 solid fills, 11–12 text. `--qr-<role>-contrast` is the text on a step-9 fill.

## Add your own CSS

Embed a stylesheet and hand it to the site:

```go
//go:embed *.css
var static embed.FS

quire.Run(site, mount,
  quire.WithStylesheets(static, "site.css"),
  quire.WithStylesheetsAt(model.WidthNarrow, static, "site-narrow.css"),
)
```

Site CSS is composited last, so it overrides Quire's.
`WithStylesheetsAt` is CSS for one step of the page grid, `WidthNarrow`, `WidthWide` or `WidthFull`. Quire writes the media query, so a sheet never states a pixel width.

Most changes are an override of a custom property rather than a new rule. The properties are listed under [Tokens](#tokens).

## Replace a page or a component

The default pages are `Home`, `Listing`, `Article`, `NotFound` and `Footer`, in package `ui`.

- `AddPage("/", …)` replaces `Home`.
- `SetNotFound` replaces `NotFound`.
- `WithFooter` replaces `Footer`, and `WithHeader` the header.
- `Spec.Index` and `Spec.Item` replace `Listing` and `Article` for one collection. Each is handed the shell, so a template with the same signature as the default is named directly: `Index: projectsPage`.

A page of your own wraps its body in the shell, which supplies the document, navigation and footer:

```templ
templ projectsPage(shell model.Shell, c *model.Collection[model.Meta]) {
  @shell(c.IndexPage()) {
    <article class="page page--wide">
      <h1>{ c.Title() }</h1>
      <div class="card-grid">
        for _, d := range c.Docs() {
          @ui.Card(ui.DocCard(c, d))
        }
      </div>
    </article>
  }
}
```

The components it can use, also in `ui`: `Card` `Entries` `HeroFigure` `Contents` `DraftBadge` `Badges` `Tags` `BackLink` `ForwardLink` `Pager` `Nav` `ThemeToggle`.
`Pager(c, d)` links to the documents before and after `d` in its collection. `Nav(c, p)` is the header's links: the site's name as the link home (see `WithBrand`), then each section.
`DocCard(c, d)` fills a `Card` from a document: its link, the collection's kicker, title, summary, draft and featured flags, image, badges and tags. A site changes the fields it wants different.

## Reference

### Fonts

Inter for text and JetBrains Mono for code by default, served as woff2 split by unicode subset. The latin text face is preloaded.
Sheets read `--qr-font` and `--qr-font-mono`.

### Scale

The page grid is two numbers: `Root` and `Col`.

- Gutter and thumb are rem from `Root`.
- A `.page` is two columns of `Col`, `.page--wide` three, `.page--full` four. `Col` defaults to 336.

`WithScale` sets both. Page widths, the breakpoint and the slot sizes follow.
Read the properties with `var()`; never restate the number.

### Tokens

These sit on `:root` in `base.css`. Override the custom properties.

- `--qr-text-xs`…`xl`, `--qr-space-*`, `--qr-weight-*`, `--qr-lead-*`, `--qr-track-*` and `--qr-measure-text`.
- `--qr-section-gap` is the space between a page's sections. A rule dividing them pads by it, so the space either side matches.
- `--qr-bg` is the colour behind whatever is drawn there. A surface that paints its own background sets it (as `.callout` does), and diagrams read it to mask lines behind their labels.

`.prose` is a breakout grid: body text holds `--qr-measure-text` on any page, and a `.figure` or `.diagram` in it spans the page inner.

### Primitives

Quire's own sheets are `base` `toggle` `nav` `layout` `prose` `highlight`, in that order.
`.kicker` is the small accent label cards and entries use for their dates; put it on any element.

### The document

`e.Shell(PageSpec)` is the document: head, navigation, the page body, footer.
An active nav link matches `/` only as itself, and any other path as its subtree on a boundary, so `/project` does not light `/projects`.
The table of contents lists H2 and H3 only.

## In depth

The stylesheet is one file in three cascade layers, `@layer quire, plugin, site`. That fixes precedence: a site can restyle anything, and a plugin can never overrule the site it is a guest in. See [Plugins](./plugins.md#css).

Why the column is 336, why the text measure is in `ch`, and why the scales are Radix are on the [Design](./architecture.md) page.
