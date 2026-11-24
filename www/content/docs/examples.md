---
title: Examples
summary: Six complete sites, each a kind of site to start from with quire new.
---

Each example is a complete site in the repository, commented part by part.
Each adds one thing to the one before, so they read in order, and each is a kind of site to start from.

Each but the last is also running on this site, beneath `/examples/`, from the same code and content.

`quire new` copies one into a new directory as a site of its own, with a `go.mod`, a Dockerfile and a justfile.
[Getting started](./getting-started.md#start-from-an-example) has its flags.

## A simple site

A folder of markdown and one call. `index.md` is the home page, `about.md` is a page, and the folder `posts` is a listing and a page for each file.
There is no template and no CSS. [A site from the folder](./content.md#a-site-from-the-folder) is how it works.

![The home page of the simple site: its title, a line of summary and a paragraph, with Posts and About in the navigation](shots/site.jpg "The home page, which is content/index.md")

```sh
go run github.com/lahickey/quire/cmd/quire@latest new --from site mysite
```

The source is [`examples/1-site`](https://github.com/lahickey/quire/tree/master/examples/1-site).
It is [running here](/examples/site).

## A blog

The same one call, with more in its posts: a picture as the hero and the thumbnail, pictures in the body, a video, links and a diagram.
Nothing in the code asks for any of them. [A blog](./blog.md) goes through what each field does.

![A post's page: under the title its date, the date it was updated and a reading time, then its summary, two link buttons and its picture as the hero](shots/article.jpg "A post: dates, reading time, summary, links, hero, body")

```sh
go run github.com/lahickey/quire/cmd/quire@latest new --from blog mysite
```

The source is [`examples/2-blog`](https://github.com/lahickey/quire/tree/master/examples/2-blog).
It is [running here](/examples/blog).

## A portfolio

A site with a look of its own. It has its own home page, a grid of projects and a contact page, a tag vocabulary, a widget, its own palette with a picker, its own stylesheets, icon and footer.
The writing and each article are still the default look, so it shows how much a site replaces and how much it leaves. [The look](./look.md) covers each part.

![Three cards on the portfolio home page: one badged Draft, one badged Featured, one plain, each with its tags](shots/badges.jpg "The home page, run with --drafts")

```sh
go run github.com/lahickey/quire/cmd/quire@latest new --from portfolio mysite
```

The source is [`examples/3-portfolio`](https://github.com/lahickey/quire/tree/master/examples/3-portfolio).
It is [running here](/examples/portfolio).

## A shop

Prints with a price, and a basket.
Each piece keeps its medium and its price in its frontmatter, the pages are the site's own, and the basket is a feature: its count and its lines update live in every open tab, and it works as plain forms without JavaScript.
It is handed a store and sessions, and has a page for whoever runs the shop behind a gate. [Features](./features.md) is written round it.

![The shop home page: the newest print edge to edge, then each piece as a card with its medium, and a basket count in the navigation](shots/shop.jpg "The home page, with the basket in the navigation")

```sh
go run github.com/lahickey/quire/cmd/quire@latest new --from shop mysite
```

The source is [`examples/4-shop`](https://github.com/lahickey/quire/tree/master/examples/4-shop).
It is [running here](/examples/shop).

## Documentation

A documentation site from the docs pack: a landing page, the documents beside a sidebar, and search, with no templates or CSS.
This site is made the same way. [Packs](./packs.md) covers the docs pack.

![A documentation page beside its sidebar, with the current page marked and a table in the body](shots/docs.jpg "A document beside the sidebar")

```sh
go run github.com/lahickey/quire/cmd/quire@latest new --from docs mysite
```

The source is [`examples/5-docs`](https://github.com/lahickey/quire/tree/master/examples/5-docs).
It is [running here](/examples/docs).

## No look at all

The same folder of markdown as the simple site, with no pack beneath it: no stylesheet, no script, no frame.
Three small templates of its own draw a page, a listing and a document, so it is where to start for a look that is wholly yours.

![A plain unstyled page listing one post as a link](shots/naked.jpg "The listing, drawn by a template of the site's own")

```sh
go run github.com/lahickey/quire/cmd/quire@latest new --from naked mysite
```

The source is [`examples/6-naked`](https://github.com/lahickey/quire/tree/master/examples/6-naked).

## Two that are not starters

- [`mounted`](https://github.com/lahickey/quire/tree/master/examples/mounted) is a program with a mux of its own that mounts a Quire site under a path.
- [`several`](https://github.com/lahickey/quire/tree/master/examples/several) is two sites in one server, each from a folder of its own.

[Wiring](./wiring.md#in-depth) covers both.
