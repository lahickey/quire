// Package ui is Quire's templ components: the page shell, cards, listings, figures and their helpers.
package ui

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
)

const (
	toggleTheme = `document.documentElement.dataset.theme = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark'; localStorage.setItem('theme', document.documentElement.dataset.theme)`
	pickPalette = `document.documentElement.dataset.palette = el.value; localStorage.setItem('palette', el.value); el.closest('details').open = false`
)

func navActive(path, href string) bool {
	if href == "/" {
		return path == "/"
	}
	return path == href || strings.HasPrefix(path, href+"/")
}

func navClass(path, href string) string {
	if navActive(path, href) {
		return "nav-link nav-link--active"
	}
	return "nav-link"
}

func figureClass(size model.Size) string {
	if size == model.SizeBleed {
		return "figure figure--bleed"
	}
	return "figure"
}

func Render(c templ.Component) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := c.Render(r.Context(), w); err != nil {
			log.Printf("render: %v", err)
		}
	}
}

func Code(text string) templ.Component {
	return templ.Raw(model.CodeBlock(templ.EscapeString(text)))
}

func CurrentYear() string { return strconv.Itoa(time.Now().Year()) }

func ReadingTime(words int) string {
	if m := max(1, (words+199)/200); m > 1 {
		return fmt.Sprintf("%d min read", m)
	}
	return "1 min read"
}
