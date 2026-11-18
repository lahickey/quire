---
kicker: What kind of thing it is
title: The one line a visitor should remember.
lede: Two sentences on what the project does and who it is for. This file is content/index.md, and everything on this page comes from it.
install: go install example.com/project@latest
repo: https://github.com/example/project
footer: Example is MIT licensed.
docs:
  summary: How to use Example.
  sections:
    - title: Start here
      pages: [getting-started]
    - title: Guides
      pages: [writing, landing]
---

The body of `index.md` starts here, under the hero.
The cards below are a `cards` fence in it, so what is written before them is drawn above them.

```cards
- href: /docs/getting-started
  kicker: Start
  title: Up in a minute
  description: Each card links to the page that backs up its claim.
- href: /docs/writing
  kicker: Writing
  title: A page is a file
  description: Add markdown to content/docs and name it in the sidebar.
- href: /docs/landing
  kicker: This page
  title: Copy in content
  description: The hero, the footer and the sidebar's sections are frontmatter. The rest is the body.
```

## Below the cards

The body carries on, as any document does.
