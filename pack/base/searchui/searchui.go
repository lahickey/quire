// Package searchui is how the default look draws a search: the box in the nav, the results as the visitor types, and the results page.
package searchui

import (
	"embed"

	"github.com/lahickey/quire/feature/search"
)

//go:embed search.css search.js
var files embed.FS

func Look() search.Look {
	return search.Look{Box: box, Results: results, Page: page, Files: files, Sheets: []string{"search.css"}, Scripts: []string{"search.js"}}
}
