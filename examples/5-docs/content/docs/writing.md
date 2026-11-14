---
title: Writing a page
summary: Where a page goes, and how it finds its place in the sidebar.
---

A page is a markdown file in `content/docs`. Its file name is its address.

## Frontmatter

| Field | Does |
|-|-|
| `title` | The heading, and the link in the sidebar |
| `summary` | The line under the heading |

## The sidebar

The sidebar is `docs.sections` in `content/index.md`: each section's title, and its pages by file name, in order.
Add a page's name there to place it, and move the name to move the page.

A page that is not listed, or a name with no page, stops the site from starting, so a typo cannot hide a page.

## Code

```sh
go run . --dev --drafts
```
