---
title: The look
summary: Change the colours, fonts and CSS a site starts with, or replace a component or a page.
---

A new site already has a look: a palette, fonts, a page grid, a stylesheet, components and pages.
Each of those is a layer. Each is complete on its own and replaceable without touching the others, so dropping a layer is not a rewrite.

This page goes from the smallest change to the largest.

## Change the colours

The palette is roles, each a hue. These are this site's, drawn by the `palette` fence around the word `all`:

```palette
all
```

`neutral` and `accent` are ranges, with all twelve steps of their hue.
Every other role has four steps, named for what each is for.

Pick another palette with an option:

```go
app.Run(identity, blueprint, quire.WithPalette(model.PaletteMoss))
```

The built-in palettes are Ember (the default), Brass, Moss, Tide, Ink, Dusk, Bloom, Wine and Ash; `model.Palettes` is all of them.
Each is drawn here whole, by a fence such as `palette moss`.

```palette ember
all
```

```palette brass
all
```

```palette moss
all
```

```palette tide
all
```

```palette ink
all
```

```palette dusk
all
```

```palette bloom
all
```

```palette wine
all
```

```palette ash
all
```

`WithTheme(model.ThemeDark)` makes dark the scheme a visitor sees when neither they nor their system has chosen one.

A `Palette` names the Radix hue behind each role, so a palette of your own is a struct of hue names.
Each role needs a hue of its own, or two callouts or two chart series would come out the same colour; a palette that repeats one fails at startup.
The tones are in an order: round the colour wheel from the accent, then taken alternately from each half, so each is far from the one before it.
Chart series and the parts of code take them in that order, so neighbours stay easy to tell apart.

```go
var harbour = model.Palette{
  Name: "Harbour", Neutral: "olive", Accent: "teal", Spot: "tomato",
  Tone: [6]string{"sky", "pink", "indigo", "orange", "violet", "brown"},
  OK:     "grass", Warn: "amber", Error: "red",
}
```

The two ranges are twelve steps each, `--qr-neutral-1` to `--qr-neutral-12`:

- `--qr-neutral-*` for surfaces, borders and text;
- `--qr-accent-*` for the brand and everything interactive.

A step's number is its job: 1–2 backgrounds, 3–5 component backgrounds, 6–8 borders, 9–10 solid fills, 11–12 text.

The other roles are four steps each, named for the job:

- `--qr-spot-*` for the one thing that must stand out, such as a draft badge or an important callout;
- `--qr-tone1-*`…`6` for telling things apart: chart series, diagram nodes, the parts of highlighted code;
- `--qr-ok-*`, `--qr-warn-*` and `--qr-error-*` for meaning, whatever hue a palette gives them.

| Step | Is for | Of twelve |
|-|-|-|
| `--qr-warn-tint` | A background | 3 |
| `--qr-warn-line` | A border | 7 |
| `--qr-warn-solid` | A solid fill | 9 |
| `--qr-warn-text` | Text | 11 |

`--qr-<role>-contrast` is the text on a role's solid fill.

## Let the visitor choose

`WithPalettePicker` puts a control in the navigation that switches the palette, and the browser remembers the choice.
This site has one: the drop beside the theme toggle.

```go
app.Run(identity, blueprint, quire.WithPalettePicker())
```

With no arguments it offers every built-in palette. Name the ones to offer, your own among them, to offer fewer:

```go
quire.WithPalettePicker(harbour, model.PaletteMoss, model.PaletteDusk)
```

- The site's own palette, from `WithPalette`, is always offered and is what a new visitor sees.
- A palette in the picker needs a `Name`.
- The stylesheet carries only the palettes offered, about 1 KB compressed for each.
- Share cards are drawn at bake time, so they stay in the site's own palette.
- A header of your own, from `WithHeader`, draws the control itself with `ui.PalettePicker(c.Palettes)`, as it does `ui.ThemeToggle()`.

The `themed` example offers its own palette and two built-in ones.

## Add your own CSS

Embed a stylesheet and hand it to the site:

```go
//go:embed *.css
var static embed.FS

app.Run(identity, blueprint,
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

The components it can use, also in `ui`: `Card` `Entries` `Img` `LazyFigure` `HeroFigure` `Code` `Contents` `DraftBadge` `Badges` `Tags` `BackLink` `ForwardLink` `Link` `Pager` `Nav` `ThemeToggle` `PalettePicker`.
`Code(text)` is a code block with the copy button a fenced block gets; the install line on this site's home page is one.
`Pager(c, d)` links to the documents before and after `d` in its collection. `Nav(c, p)` is the header's links: the site's name as the link home (see `WithBrand`), then each section.
`DocCard(c, d)` fills a `Card` from a document: its link, the collection's kicker, title, summary, draft and featured flags, picture, badges and tags. A site changes the fields it wants different.

### Links

`ui.Link(l, class)` draws a `model.Link`, a label and a URL, as an anchor with the class it is given: a button with `btn-primary` or `btn-secondary`, plain text with none.
A link to GitHub or LinkedIn gets that site's mark before its label.
The mark is drawn in the text's own colour and at its size, so it follows the theme and the class.
The Source link in this site's header is one, as is the Source button on its home page.

```templ
@ui.Link(model.Link{Label: "GitHub", URL: "https://github.com/lahickey/quire"}, "btn-secondary")
```

With the class `link-mark` a link that has a mark shows the mark alone, as the footer's links do (the one at the foot of this page is this site's repository); its label stays for screen readers and as the tooltip, and a link with no mark keeps its text.
A link that leaves the site opens in a new tab.
A store that requires its own badge is the one case a label and a mark do not cover: set `Badge` to the URL of the badge's image and the link is drawn as that image alone, with the label as its alt text.
`model.Link` has `label` and `url` yaml keys, so a site's own frontmatter can carry a list of them.

`model.BuiltWithQuire` is a link to Quire's site, for a footer that says what the site is made with.
The "Built with Quire" at the foot of this page is it:

```go
quire.WithFooterLinks(model.BuiltWithQuire)
```

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
- `--qr-surface` is the background of a quiet surface such as a code block or a card, and `--qr-surface-strong` of a stronger one such as inline code or a tag. With `--qr-bg` they set how bright a site is: light starts a step further from white than dark does from black, and a site that wants otherwise overrides the three under `[data-theme="light"]` or `:root`.
- `--qr-header` is the height of the header, which stays at the top of the window. Anchors and the docs sidebar sit below it by this much, so a header of your own that is taller or shorter sets it.
- `--qr-bg` is the colour behind whatever is drawn there. A surface that paints its own background sets it (as `.callout` does), and diagrams read it to mask lines behind their labels.

`.prose` is a breakout grid: body text holds `--qr-measure-text` on any page, and a `.figure` or `.diagram` in it spans the page inner.

### Code

Highlighted code takes its colours from the palette, so it changes with it.
Each kind of thing in the code has a variable, and the `highlight` sheet sets each to a role.
A site's own sheet sets any of them again.

```css
:root, [data-palette] {
  --qr-code-keyword: var(--qr-spot-text);
  --qr-code-string: var(--qr-ok-text);
}
```

| Variable | Is by default |
|-|-|
| `--qr-code-keyword` | `accent` |
| `--qr-code-builtin` | `tone1` |
| `--qr-code-function` | `tone2` |
| `--qr-code-type` | `tone3` |
| `--qr-code-tag` | `tone4` |
| `--qr-code-string` | `tone5` |
| `--qr-code-number` | `tone6` |
| `--qr-code-preproc` | `spot` |
| `--qr-code-comment`, `--qr-code-symbol` | `neutral` |
| `--qr-code-heading` | `accent` |
| `--qr-code-inserted`, `--qr-code-deleted` | `ok`, `error` |

The roles code draws on, in this site's palette; it uses the `text` step of each:

```palette
spot tone1 tone2 tone3 tone4 tone5 tone6
```

A variable is set on `[data-palette]` as well as `:root` so that it follows a palette set on part of a page, as on this code, which is under Moss whatever the page is under:

<div data-palette="moss">

```go
// Greet says hello, n times.
func Greet(name string, n int) []string {
  out := make([]string, 0, n)
  for range n {
    out = append(out, "Hello, "+name)
  }
  return out
}
```

</div>

For anything a variable cannot do, such as a weight, each piece of code is a `span` with a class for its kind: `lx-keyword`, `lx-string` and so on, one for each of [Lexis's kinds](./lexis.md#kinds).

### Primitives

Quire's own sheets are `base` `toggle` `nav` `layout` `prose` `highlight`, in that order.
A class Quire ships works in a template of your own and in raw HTML in a document.

`.kicker` is the small accent label cards and entries use for their dates; put it on any element.

```html
<div class="kicker">Feb 2026</div>
```

<div class="kicker">Feb 2026</div>

`.btn-primary` and `.btn-secondary` are links drawn as buttons, and `.btn-row` spaces them.

```html
<div class="btn-row">
  <a class="btn-primary" href="/docs">Read the docs</a>
  <a class="btn-secondary" href="/">Home</a>
</div>
```

<div class="btn-row">
  <a class="btn-primary" href="/docs">Read the docs</a>
  <a class="btn-secondary" href="/">Home</a>
</div>

`.card-grid` lays out `ui.Card`s in columns of `Col`. The [documentation index](/docs) is one, and so are the cards on the home page, from a [`cards` fence](./content.md#cards).

`.page` is the page's width, with `.page--wide` and `.page--full` for three and four columns. This page is a `.page--wide` with a sidebar in it.

### The document

`site.Shell(PageSpec)` is the document: head, navigation, the page body, footer.
An active nav link matches `/` only as itself, and any other path as its subtree on a boundary, so `/project` does not light `/projects`.
The table of contents lists H2 and H3 only.

## In depth

The stylesheet is one file in four cascade layers, `@layer quire, plugin, pack, site`. That fixes precedence: a site can restyle anything, and neither a plugin nor a pack can overrule the site it is a guest in. See [Plugins](./plugins.md#css).

Why the column is 336, why the text measure is in `ch`, and why the scales are Radix are on the [Design](./architecture.md) page.
