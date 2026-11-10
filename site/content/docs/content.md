---
title: Content
summary: Markdown, frontmatter, dates, collections, tags and widgets.
---

## Markdown

Markdown is YAML frontmatter between `---` fences, then body.
Lists are YAML lists (`tag: [Go, NATS]`).
Unknown keys are a load error.
A body `---` is an ordinary rule.

`Meta` is `title` `draft` `featured` `date` `updated` `summary` `image` `tag`.
Goldmark renders raw HTML, because the markdown is the site's own. It runs in the bake: a shipped binary holds the HTML, not the markdown.
A site type embeds it with `yaml:",inline"`.

`image` is a picture beside the document, relative to it like a link (`archr/columns.png`), or from the content root when it starts with `/`; base renders it as a listing thumbnail, an article hero, and the share image, with no CSS of your own.
A body `![](path)` uses the same resolution: a paragraph of its own is a `.figure`, inline stays `<img>`, `http(s):` is left alone, and a missing file fails the load.
A figure's text is both its caption and its alt text, as in Pandoc; `![alt](path "caption")` is for when the two should differ.
`e.Image(name)` returns a master this engine has already loaded.

`[text]{.class}` is a span with those classes, Pandoc's bracketed span: a few characters styled by the site's CSS without writing HTML. The text is literal and the braces take classes only.

A blockquote opening with GitHub's `[!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]` or `[!CAUTION]` line renders as a `.callout`; anywhere else it still reads as a quote.

## Diagrams

A ```` ```mermaid ```` fence is a diagram: drawn by the bake, found in the bundle, and inlined as SVG so the palette and the fonts reach it.
One the bundle does not hold fails the load, as does one measured in other fonts than the site serves.
Nodes take a light tint of the accent; `class n swatch2` or `n:::swatch2` tints one with any role (`accent` `spot` `swatch1`…`4` `red` `green` `amber`), and a forge's preview just ignores the class.

## Widgets

Named fences are site-supplied widgets: `Spec.Widgets` maps a fence's name to a `Widget`, which is handed the rest of the fence's first line and its body and returns HTML.
```` ```mermaid ```` and ```` ```palette ```` are Quire's own, and a site's widget of either name replaces it. `palette` draws each role it names (`neutral accent spot`, …) as its twelve steps, from the page's own custom properties.
A fence shown inside a longer fence stays code.
A widget runs when the page is composed, not in the bake, so changing one needs no rebake; the bake records only where each fence sits. A widget that returns an error fails the load, same as bad frontmatter.

## Dates

Dates are ISO `YYYY-MM-DD`, parsed once at load into `Date`; anything else fails the build rather than reaching a page.
`Site.Dates` is the layouts dates render in, day first by default, and a collection's `LongDate` `ShortDate` `ShortMonthYear` format a `Date` to it.
`Date.ISO()` is the machine form, and sitemaps and structured data always use it.
A custom formatter takes a parsed `Date`, so it only formats.

## State

`draft` and `featured` are the two the engine acts on: drafts are filtered from the build and the sitemap unless `WithDrafts`; featured is only ever selected and badged.
Anything else a document says about its own state is the site's vocabulary — override `Badges() []Marker` on the site type and quire renders what it is handed, knowing none of the words.

## Collections

`AddCollection` loads a markdown glob (`Markdown`, default `{id}/*.md`), drops drafts unless `WithDrafts`, sorts date-descending, mounts `/{id}` and `/{id}/{slug}`, and records them in the sitemap.
Empty after the draft filter: not mounted.
`model.Basic` is `Spec[Meta]`, the spec for stock frontmatter; a site with its own frontmatter type uses `Spec[T]`.

- Nil `Index` / `Item` render `Listing` / `Article`, which show the title, date, tags, hero, contents and body, and link back to the index. Either is handed `model.Shell` as well as the collection, so it needs no closure over the engine. A missing slug is the site 404; two documents with one slug fail the load.
- `Kicker` labels each document in `Entries` rows and cards; nil uses `ShortMonthYear`. `Entries(c, docs)` renders the same rows anywhere.
- `Featured()` selects from a collection.

## Tags

`Spec.Classifier` is the site's tag vocabulary. When set, every `tag` must classify or the load fails, so a new term cannot ship unstyled.
`NewClassifier` takes `Exact(category, needles…)` rules, tried in order, first match wins; a needle matches the whole value, ignoring case.
`Collection.Tags(d)` is a document's tags as markers carrying their category, ready for `ui.Tags`.
