---
title: Baking
summary: What a bake makes, what reaches the bundle, and the controls.
---

## What it does

A site binary that can bake, and has no embedded bundle, bakes before it serves; `go run . bake` bakes and exits.
It mounts the site, reading the content tree rather than the bundle, and makes what the mount asked for:

- every variant of every master a document or the mount referred to;
- every diagram in what it loaded;
- the share card;
- every document the mount asked for, converted to HTML with its headings and raw frontmatter, one file each in the journal like any other artifact, and skipped next time unless its source, the converter or the site's widget names change;
- a bundle (`.quire/bundle`) holding those files and a journal naming only them. Everything in it is a journal record: a site reads nothing else from the bundle.

Nothing else in the content tree reaches the bundle.
A file nothing refers to — a PDF, say — is shipped with `AddFile`, and an icon with `WithIcons`.

The cache (`.quire/cache`) is incremental: over an unchanged library a bake costs a hash per master.
Files a narrower bake did not ask for stay in the cache until it is wiped; the bundle is fresh each time, so they cannot ship.
The bake writes `.quire/.gitignore` ignoring everything in it, itself included: `.quire` is never committed and can be deleted at any time.

A site reads the bundle off disk when it can bake and out of the embed when it cannot, the same artifact either way.
It never encodes: a variant it was not given is a 404.

## Controls

`WithBake` is the whole control, with paths relative to where the bake runs:

```go
quire.WithBake(model.BakeSpec{
	Content: "content", // the default
	Dir:     ".quire",  // the default; holds bundle/ and cache/
})
```

## The shipped binary

Built with `-tags quire_nobake`, a site binary carries a stub in place of the bake and links none of it, so the binary that ships cannot convert: no codec, no markdown, no diagram renderer.
Where its bundle comes from is a separate flag: built with `-overlay .quire/ship.json` too, it serves the bundle it embeds (see [Getting started](./getting-started.md)); without, it reads `.quire/bundle` off disk.
A CMS, which bakes uploads, ships with neither.
