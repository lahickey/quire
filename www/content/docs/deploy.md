---
title: Deploying
summary: Build the binary, put it behind TLS, and what it needs once it is running.
---

A shipped site is one static binary with its content inside. It needs no files beside it, no database and no runtime.

## Build it

```sh
go run . bake
CGO_ENABLED=0 go build -tags quire_nobake -overlay .quire/ship.json -o server .
```

The bake runs where the content is, on your machine or in CI, and the build embeds what it made. See [Baking](./bake.md#the-shipped-binary).
Set `GOOS` and `GOARCH` to build for another machine.

A site started with `quire new` has a Dockerfile that does the build and a justfile that bakes first:

```sh
just build   # bake, then build the image
just run     # run it on :8080
```

The image is `FROM scratch`: the binary and nothing else, running as a user that is not root.

## Run it

```sh
./server --addr :8080
```

The binary serves plain HTTP. Put it behind whatever terminates TLS for you: a platform's proxy, a load balancer or a reverse proxy.
Behind one that hands the site a path and not a whole host, start it with `--root /that/path`; see [Placing a site](./parts.md#placing-a-site).

Set `Identity.BaseURL` to the address visitors use. It decides:

- the canonical URL and share tags of every page, and whether `sitemap.xml` and `robots.txt` exist;
- the `www` redirect, which sends `www.` on that host to the bare name;
- whether the session cookie a feature uses is marked secure, which it is when the address is `https`.

## Redirects

A trailing slash redirects to the path without one. `www` redirects to the host in `BaseURL`, and only to that host.
Both are sent with `Cache-Control: no-cache`, so a browser cannot pin one.

## Health

`/_quire/health` answers `200` with a line of text. It renders no page and is never cached, so it is the path to point a health check at.

## What it costs to run

Pages are rendered once and held compressed, pictures are streamed from the binary, and nothing is converted while serving.
This site runs on one shared CPU with 256MB of memory.

What is kept, and what a browser may cache, is [below](#caching-and-compression).

## Caching and compression

- A fixed page is rendered once, when the site is built.
- A collection's document page is rendered on first request and held in a 4MB page cache, keyed by its path. So is a route added with `AddSharedRoute`.
- A raw route goes straight to its handler and is never cached.
- `robots.txt` and `sitemap.xml` are built once, at boot.
- An asset linked with its content hash (`?h=`) is cached by the browser for good. Anything else is revalidated.

A cached body is compressed once, as brotli or gzip, and stored that way. A body under 256 bytes is sent as it is.
`--dev` sends pages `no-store`.

## Checking before you ship

`go run . --drafts` shows what is not published yet, and `go run .` shows the site exactly as it ships, with production's caching.
A broken page, a missing picture or an unknown frontmatter field stops the bake, so a build that succeeds has every page.
