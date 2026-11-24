// A portfolio: a site with its own look. It has a home page, a projects page, a
// contact page and a not-found page of its own, tags, a markdown widget, and its
// own palette, theme, stylesheets, icon and footer.
// Everything else is the default look, including each article and the writing index.
//
// The site is the package portfolio, in the folder of that name: start reading
// at portfolio.go, then pages.templ.
package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/examples/3-portfolio/portfolio"
	"github.com/lahickey/quire/site"
)

func main() {
	quire.Run(portfolio.Identity, site.FeatureFunc(portfolio.Declare), portfolio.Options()...)
}
