---
title: The parts
summary: Compose a site, a server and a bake yourself - a main of your own, a site at a path, several sites in one server.
---

`quire.Run` is a helper: it bakes, builds and serves in one call.
Everything it calls is public, so a program that wants more than `Run` gives, or less, composes the parts itself.

```
features + baker   --bake-->   bundle
features + bundle  --build-->  site
site               --mount-->  a server, or a mux of your own
```

## The server

A site has no server in it: its routes are a table, which `Routes()` reads.
The server, in package `server`, makes a mux from that table and adds what claims the whole address space: the not-found page for every other path, `robots.txt`, the sitemap, the health check and the `www` redirect.
It also runs the site's services: `Start` starts them in the order they were added, and `Stop` stops them in reverse.

Building the site renders every fixed page, so a template error stops the binary at boot.
A site with routes of its own adds them with `AddRoute`, in its own feature or one it includes. Both go through the site's route claims, so a clash is an error at boot.

A test of a feature needs no bundle: `site.Describe(identity, feature).Build(nil)` and `server.New` give its handler.

## Without Run

A site is described once, and the same description is baked and served. Serving is three calls on it, and a program that wants a `main` of its own makes them itself:

```go
//go:embed all:.quire/bundle
var embedded embed.FS

func main() {
  files, err := fs.Sub(embedded, ".quire/bundle")
  if err != nil {
    log.Fatal(err)
  }
  baked, err := bundle.Open(files)
  if err != nil {
    log.Fatal(err)
  }
  built, err := site.Describe(identity, feature, site.WithFeatures(base.Pack{})).Build(baked)
  if err != nil {
    log.Fatal(err)
  }
  handler, err := server.New(built)
  if err != nil {
    log.Fatal(err)
  }
  if err := handler.Start(context.Background()); err != nil {
    log.Fatal(err)
  }
  log.Fatal(http.ListenAndServe(":8090", handler))
}
```

- `bundle.Open` reads a bundle from any `fs.FS`: one embedded by hand, as here, or a folder through `os.DirFS`.
- `site.Describe` gathers what makes a site but its source: the identity, the site's own feature and the options, as `Run` takes them.
- `Build` runs the features over the bundle and returns the site.
- `base.Pack{}`, from `pack/base`, is what `Run` puts beneath every site: its stylesheets, fonts and frame, what a document's fences and pictures become, the share card, and the home and not-found pages. Without it a page has no stylesheet, a `mermaid` fence is plain code, a picture is a bare `<img>`, no page has a share image but a document's own, and there is no page at `/` until the site adds one.
- `server.New` returns the server, which is an `http.Handler`. `Start` starts the site's services, and `Stop` stops them; `Run` calls `Stop` when the process is interrupted, after closing its connections.

A program with a mux of its own needs no server: it adds each of `built.Routes()` to its mux, and `built.NotFound()` is the page for anything else.

`Build(nil)` builds a site with nothing baked. Its pages are all Go, it needs no bundle, and its stylesheet, scripts and fonts are served as for any other site.
What a site asks of a bundle is `site.Source`, and anything with those six methods can stand in for one.

## Placing a site

A site writes its own addresses as if it were at `/`: its pages, its links, its features' endpoints and its groups.
Where it is served is a separate thing, a `model.Place`, and nothing in the site changes when that does.

```go
at := model.Place{Root: "/docs", Engine: "/docs/_quire"}
built, err := site.Describe(identity, feature, site.WithPlace(at)).Build(baked)
```

- `Root` goes in front of the site's pages, its features' `/_<name>/` and its groups: `/docs/install`, `/docs/_search/live`, `/docs/admin/`.
- `Engine` is where what the site has under `/_quire/` goes: its stylesheet, scripts, fonts and pictures. Left empty that is `/_quire` itself, which is right for a site alone. Under a host that owns the rest of the address space it is beneath the root, as here.

The place is applied at the two edges, and nowhere between:

- **Coming in.** `Routes()` gives the placed patterns, and a handler is handed the request with the site's own address in it.
- **Going out.** A page, any markup a handler answers with, and a stream's element patch have their addresses placed, in `href`, `src`, `srcset`, `action`, `poster` and what a Datastar action fetches. So are a redirect, the stylesheet's `url()`s, the canonical and share addresses, the sitemap and the session cookie's path.

An address built in a script, or written into an attribute not listed, is not placed; a link in a page always is.
The saved theme and palette are kept under the root, so two sites on one host do not share them.

`--root /docs` gives `Run` a place with everything under it, for a site behind a proxy that hands it one path.
`examples/mounted` is a program with a mux of its own that mounts a site under `/docs`, beside a page of its own drawn with the site's document.

## Several sites

`quire.RunSites` is `Run` for several sites in one server:

```go
quire.RunSites(
  quire.Site{Identity: model.Identity{Name: "Front"}, Feature: base.Content(), Content: "front"},
  quire.Site{Identity: model.Identity{Name: "Notes"}, Feature: base.Content(), Content: "notes", Root: "/notes"},
)
```

- `Root` is the path a site is served under. The first has none.
- `Content` is the site's content folder, `content` if it names none, so each of several names its own.
- `Options` are the site's own, as the last arguments of `Run` are.
- Each site is baked into a bundle of its own: the first into `.quire`, and one at `/notes` into `.quire/sites/notes`. The one `ship.go` embeds them all, so shipping is the same two commands.
- The flags are `Run`'s, and apply to every site. `--root` puts all of them under one path.
- Every site has base beneath it. Sites with no pack beneath them are composed by hand.

`examples/several` is two sites made this way.

Beneath it, a server takes any number of placed sites:

```go
v, err := server.New(
  front,                                                             // model.Place{}
  notes,                                                             // model.Place{Root: "/notes", Engine: "/_quire/notes"}
)
```

- **Each site is whole.** It has its own identity, features, look, groups and bundle, and is written as if it were alone at `/`. One of several can be given a server of its own later and nothing in it changes.
- **Everything Quire serves for itself is under `/_quire/`**, each site in a part of its own, so one rule at a proxy covers every site on the box.
- **The first site is the box's own.** The health check, `robots.txt`, the `www` redirect and the page for an address no site has are its. Every other site has its own not-found page under its root, and each has its own sitemap, which the one `robots.txt` lists.
- **Two sites with a route at one address** fail when the server is made.
- **Every site's services are started**, in the order the sites were given. One handed to several sites is started once, and told the environment of the first.

## A bake of your own

Baking is three calls too:

```go
func main() {
  baker, err := bake.Start(model.BakeSpec{Content: "content", Dir: ".quire"}, convert.Standard())
  if err != nil {
    log.Fatal(err)
  }
  if err := site.Describe(identity, feature, site.WithFeatures(base.Pack{})).Collect(baker); err != nil {
    log.Fatal(err)
  }
  if _, err := baker.Finish(); err != nil {
    log.Fatal(err)
  }
}
```

`quire.Bake(description, spec)` is those three calls with base's converters and any added with `quire.Converters`, and it also writes the `ship.go` that embeds what was baked.
A program with several sites bakes each into a folder of its own beneath `.quire`, and the one `ship.go` embeds them all.
`quire.Bundle(spec)` then opens a site's bundle: from the binary when it was built with that `ship.go`, and from disk otherwise.

- `bake.Start` is told where the content and the output live, and handed what it can convert. `convert.Standard()`, from `pack/base/convert`, is the default set: markdown, mermaid, the share card and WebP pictures. Several sets are combined, and two converters of one name are refused.
- `Collect` runs the features over the baker, so the site asks for everything it needs. It is the same description `Build` is called on, so the bake and the site cannot ask for different things.
- `Finish` encodes the pictures, writes the bundle to `.quire/bundle`, and returns it.

A baker asked for a picture has not made it yet, because pictures are encoded together in `Finish`. Until then its answer is the master's name and nothing more, and the record is in the bundle `Finish` returns.

A program that serves with `site` and `server` and bakes in a program of its own, as these two do, links no codec, no markdown and no highlighter into what it ships, with no build tag.
