---
title: Shipping a site as one binary
date: 2026-04-01
summary: Why the words, the pictures and the server end up in the same file.
featured: true
tag: [Go]
---

A static site is a folder you upload. This one is a program you run, and the
folder is inside it.

## What the bake does

Every picture is encoded once, ahead of time, at the widths the layout asks
for. The pages are rendered at boot and kept in memory.

```mermaid
flowchart LR
  md[markdown] --> bake --> bundle[(bundle)] --> binary
```

## What ships

The binary that ships cannot encode anything. It serves what the bake left and
nothing else, so there is no code path that turns a request into work.

## The palette

A fence named `swatch` is a widget: the site turns its body into whatever HTML
it likes.

```swatch
ink
```

`palette` is built in: it draws each role it names as its twelve steps.

```palette
neutral accent spot
```
