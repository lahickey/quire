---
title: Plugins
summary: Add behaviour the engine does not have, and update pages live with Datastar.
---

A plugin is a unit of behaviour the engine does not have: a basket, a search box, a comment thread.
It brings its own routes, pages, styles and state, and a site opts in with one line.

## Using a plugin

```go
func mount(e *quire.Engine) error {
  // collections and pages first
  return e.Use(basket.New(items...))
}
```

`Use` attaches the plugin: its routes, its page, its stylesheets and its control in the nav.
A plugin's pages live under `/_<name>` unless the site moves them:

```go
e.Use(plugin.At("/basket", b))
```

`examples/4-plugin` is a shop built this way.

## Search

Quire ships one plugin, in `plugin/search`: a search box in the nav whose results arrive as the visitor types.

```go
s := search.New()
s.Add(pages)
return e.Use(plugin.At("/search", s))
```

`Add` indexes a collection, one entry for each document's opening and one for each H2 and H3, held in memory. Call it for every collection the box should find.
In the box, `/` focuses it, up and down move through the results, and enter opens one.
Without JavaScript the form loads a results page instead, at `/_search` unless the site moves it.

## Writing a plugin

A `Plugin` is two methods:

```go
type Plugin interface {
  Name() string
  Attach(*Mount) error
}
```

`Name` is one word, and it is where the plugin lives: its routes are under `/_<name>/`, its CSS classes start `<name>-`, and its signals are `$<name>`.
`Attach` is called once and declares what the plugin adds:

```go
func (p *Plugin) Attach(m *plugin.Mount) error {
  p.mount = m
  p.streams = m.NewStreams()
  m.UseSessions()
  m.AddStylesheets(css, "basket.css")
  m.AddNavEnd(p.nav())
  m.Route("GET", "events", p.events)
  m.Route("POST", "add", p.add)
  m.Page("GET", "", p.show)
  return nil
}
```

`e.Use(p...)` runs `Attach` during the site's mount: after content loads, so a plugin can read collections, and before the handler is taken, so its assets reach the stylesheet.
The plugin API is package `plugin`, and a plugin imports it, `model` and `ui`, never the engine.

A plugin holds its own state, embeds its own assets, and declares an interface for whatever the host must supply.
Lifting one into Quire is moving its file; nothing about how it is wired changes.

## What a plugin can do

`plugin.Mount` is the grant: the engine's doors and nothing else of it.
`go doc github.com/lahickey/quire/plugin.Mount` is the allow-list.

### Routes and pages

- `m.Route(method, path, h)` is private, under `/_basket/`. It is named from the plugin, so two plugins cannot pick the same namespace; `quire` and `site` are reserved. `m.URL(path)` is where it ended up.
- `m.Page(method, path, h)` is a page a visitor sees and may bookmark, never cached. `path` is relative, so a plugin claims nothing of the site's: `""` is `/_basket` and `"checkout"` is `/_basket/checkout`. A site moves them with `e.Use(plugin.At("/basket", b))`, to `/basket` and `/basket/checkout`. A path already claimed by the site or another plugin is a boot error naming the owner.
- `m.PageURL(path)` is where that page ended up, for links and `PageSpec.Path`.
- `m.Shell(PageSpec)` is the document chrome, so a plugin page is still a site page.

### Forms

`@m.Form(path) { … }` is a form that posts to a route: a plain post without JavaScript, and a Datastar post that stays on the page with it.
End its handler with `plugin.Done(w, r)`, which answers either kind of caller: `204` to a live page, whose stream already carries the change, and a redirect for a plain form submission, back to the page it came from if that is on this site, otherwise to `/`.

### Live updates

- `m.NewStreams()` is Datastar SSE with a subscriber list.
- `Subscribe(w, r, topic, send)` is the whole handler: it opens a connection, sends the current state and holds it open.
- `Publish(topic, send)` reaches every connection on that topic, so a change made in one tab lands in all of them.
- `Signals(v)` marshals `v` and merges it into the page's signals under the plugin's name. The basket sends `{"count": 3}` and the page holds `$basket.count`, so two plugins cannot patch each other's.
- `Elements(ctx, component)` replaces the element with the same id.
- `m.Listen(path)` is the Datastar action that subscribes to a route: `data-init={ m.Listen("events") }`. Quire ships Datastar, so it owns the options too: a subscription reconnects for as long as the page is open and closes when its element is removed.
- `m.Signals(v)` is a `data-signals` value under the plugin's name, so a page starts with what `Stream.Signals` later patches: `data-signals={ m.Signals(map[string]int{"count": 0}) }`.

### Styles, scripts, nav and sessions

- `AddStylesheets` / `AddStylesheetsAt` go into the `plugin` cascade layer, so the site can restyle a plugin and never the reverse. `AddStylesheetsAt` is CSS for one step of the scale; Scale writes the query.
- `AddScripts` adds JavaScript files to `/_quire/quire.js`, each in its own function scope.
- `AddNavEnd(component)` is a control in the nav.
- `UseSessions()` gives one `qsid` for every plugin, minted on navigation only. `plugin.Session(r)` reads it. Two cold tabs can race; last cookie wins. That is inherent to HTTP.

## CSS

The stylesheet is one sheet in three layers, `@layer quire, plugin, site`, which fixes precedence but not collision: two plugins both styling `.line` still clash.
So a plugin prefixes every class with its own name (`.basket-line`, not `.line`), and `Use` fails on a stylesheet with a class outside that prefix.
Layers beat specificity, so a bare `button {}` in a site sheet outranks a plugin's `.basket-qty-btn`. A site that mounts plugins should scope its element selectors.

## Per-visitor content on a cached page

```mermaid
flowchart LR
  form[form in one tab] -->|post| route[plugin route]
  route -->|Publish| streams[Streams]
  streams -->|signals| nav[badge in every tab]
  streams -->|elements| view[the open view]
```

A prepared page is one body for everyone, so it cannot carry a basket count.
It carries an empty badge instead, and a publish patches it per visitor.
The page stays immutable; only the patch differs.

## Datastar

Datastar is the small library Quire ships for live behaviour. A page declares what it wants in `data-*` attributes, and the server sends HTML or values over a stream.

### Syntax

Attribute syntax is `data-plugin:key`, colon not hyphen: `data-on:submit`, not `data-on-submit`.
A hyphen there does nothing at all and reports nothing. Keyless attributes (`data-text`, `data-signals`) work either way, which is what lets the mistake survive a casual test.

### Signals or elements

Choose the payload by what the audience can receive, and give each audience its own topic.
Signals are page-global and land wherever a tab happens to be, which suits a count in the nav.
Element patches need the element to exist, and Datastar warns `PatchElementsNoTargetsFound` otherwise, so a view-specific patch gets a topic only that view subscribes to.

### Where a subscription lives

Put a page-global subscription (a count, a badge) on chrome that is not swapped: a soft-nav replaces `main` and nothing else.
A view-specific subscription lives on the view's own element, inside `main`, so leaving the page cancels it.

Two standing streams per tab stall HTTP/1.1 (six connections per host) at three tabs; a second stream only while that view is open does not. Production over TLS is HTTP/2 and multiplexes, so this bites in dev first and looks like the site hanging.
A hidden tab drops its stream and resubscribes on return.

### Push, always

Signalling a page so it turns around and fetches its own update is a round trip that should not exist.

## In depth

A publisher writes into another request's connection, so every write to a stream is serialised and carries a five-second deadline: one stalled reader cannot hold up the request that published to it.
A publish writes to all of a topic's streams at once.
An idle stream is sent a comment every 25 seconds, because proxies drop a silent connection.
