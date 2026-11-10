---
title: Plugins
summary: Behaviour the engine does not have, and how to stream it live with Datastar.
---

A `Plugin` is a unit of behaviour the engine does not have — a basket, a search box, a comment thread.
It is two methods:

```go
type Plugin interface {
	Name() string
	Attach(*Mount) error
}
```

`e.Use(p...)` runs `Attach` during the site's mount: after content loads, so a plugin can read collections, and before the handler is taken, so its assets reach the stylesheet.
The plugin API is package `plugin`, and a plugin imports it, `model` and `ui`, never the engine.
`plugin.Mount` is the grant: the engine's doors and nothing else of it.
`go doc github.com/lahickey/quire/plugin.Mount` is the allow-list.

- `m.Route(method, path, h)` — private, under `/_basket/`. Named from the plugin, so two plugins cannot pick the same namespace; `quire` is reserved for Quire's own. `m.URL(path)` is where it ended up.
- `m.Page(method, path, h)` — a page a visitor sees and may bookmark, never cached. `path` is relative, so a plugin claims nothing of the site's: `""` is `/_basket` and `"checkout"` is `/_basket/checkout`. A site moves them with `e.Use(plugin.At("/basket", b))`, to `/basket` and `/basket/checkout`; a path already claimed by the site or another plugin is a boot error naming the owner.
- `m.PageURL(path)` — where that page ended up, for links and `PageSpec.Path`.
- `m.NewStreams()` — Datastar SSE with a subscriber list. `Subscribe(w, r, topic, send)` is the whole handler: it opens a connection, sends the current state and holds it open. `Publish(topic, send)` reaches every connection on that topic, so a change made in one tab lands in all of them. `Signals(v)` marshals `v` and merges it into the page's signals under the plugin's name — the basket sends `{"count": 3}` and the page holds `$basket.count` — so two plugins cannot patch each other's. `Elements(ctx, component)` replaces the element with the same id.
- `m.Listen(path)` — the Datastar action that subscribes to a route: `data-init={ m.Listen("events") }`. Quire ships Datastar, so it owns the options too: a subscription reconnects for as long as the page is open and closes when its element is removed.
- `@m.Form(path) { … }` — a form that posts to a route: a plain post without JavaScript, a Datastar post that stays on the page with it. End its handler with `plugin.Done(w, r)`.
- `m.Signals(v)` — a `data-signals` value under the plugin's name, so a page starts with what `Stream.Signals` later patches: `data-signals={ m.Signals(map[string]int{"count": 0}) }`.

The rest are forwards to the engine:

- `AddStylesheets` / `AddStylesheetsAt` — into the `plugin` cascade layer, so the site can restyle a plugin and never the reverse. `AddStylesheetsAt` is CSS for one step of the scale; Scale writes the query.
- `AddNavEnd(component)` — a control in the nav.
- `UseSessions()` — one `qsid` for every plugin, minted on navigation only. Two cold tabs can race; last cookie wins. Inherent to HTTP.
- `Shell(PageSpec)` — the document chrome, so a plugin page is still a site page.

A plugin holds its own state, embeds its own assets, and declares an interface for whatever the host must supply.
Lifting one into Quire is moving its file; nothing about how it is wired changes.

## CSS

The stylesheet is one sheet in three layers — `@layer quire, plugin, site` — which fixes precedence but not collision: two plugins both styling `.line` still clash.
So a plugin prefixes every class with its own name (`.basket-line`, not `.line`), and `Use` fails on a stylesheet with a class outside that prefix.
Layers beat specificity, so a bare `button {}` in a site sheet outranks a plugin's `.basket-qty-btn`; a site that mounts plugins should scope its element selectors.

## Per-visitor content on a cached page

A prepared page is one body for everyone, so it cannot carry a basket count.
It carries an empty badge instead, and a publish patches it per visitor.
The page stays immutable; only the patch differs.

## Datastar

`plugin.Done(w, r)` ends a form's handler for either kind of caller: `204` to a live page, whose stream already carries the change, and a redirect for a plain form submission, back to the page it came from if that is on this site, otherwise to `/`.

Attribute syntax is `data-plugin:key`, colon not hyphen — `data-on:submit`, not `data-on-submit`.
A hyphen there does nothing at all and reports nothing; keyless attributes (`data-text`, `data-signals`) work either way, which is what lets the mistake survive a casual test.

Choose the payload by what the audience can receive, and give each audience its own topic.
Signals are page-global and land wherever a tab happens to be, which suits a count in the nav.
Element patches need the element to exist — Datastar warns `PatchElementsNoTargetsFound` otherwise — so a view-specific patch gets a topic only that view subscribes to.

Put a page-global subscription (a count, a badge) on chrome that is not swapped — a soft-nav replaces `main` and nothing else.
A view-specific subscription lives on the view's own element, inside `main`, so leaving the page cancels it.
Two standing streams per tab stall HTTP/1.1 (six connections per host) at three tabs; a second stream only while that view is open does not. Production over TLS is HTTP/2 and multiplexes, so this bites in dev first and looks like the site hanging.
A hidden tab drops its stream and resubscribes on return.

Push, always.
Signalling a page so it turns around and fetches its own update is a round trip that should not exist.
