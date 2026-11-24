// A shop: prints with fields of their own in the frontmatter, pages of the
// site's own, and a basket feature that adds routes, a page and live updates,
// and is handed the services it uses: a store for its carts, and sessions.
//
// The site is the package shop, in the folder of that name: start reading at
// shop.go, then pages.templ. The basket is a package of its own because a
// feature is reusable: any site could include it.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/examples/4-shop/shop"
)

func main() {
	quire.Run(shop.Identity, shop.Feature(), shop.Options()...)
}
