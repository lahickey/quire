// Package ui is Quire's templ components: the page shell, cards, listings, figures and their helpers.
package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
)

const (
	toggleTheme = `document.documentElement.dataset.theme = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark'; localStorage.setItem('theme' + (document.documentElement.dataset.root || ''), document.documentElement.dataset.theme)`
	codeOpen    = `<div class="code"><button type="button" class="code-copy" aria-label="Copy code" data-on:click="navigator.clipboard.writeText(el.nextElementSibling.textContent); el.classList.add('code-copy--done'); setTimeout(() => el.classList.remove('code-copy--done'), 1500)"></button>`
	pickPalette = `document.documentElement.dataset.palette = el.value; localStorage.setItem('palette' + (document.documentElement.dataset.root || ''), el.value); el.closest('details').open = false`
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

// scale is the site's, from the render's context. A render no site began, such as a test of one component, gets the default.
func scale(ctx context.Context) model.Scale {
	if c, ok := model.ChromeOf(ctx); ok {
		return c.Scale
	}
	return model.DefaultScale()
}

// FrameWith is the default frame with a header or a footer of the site's own in place of the default's.
func FrameWith(header, footer func(model.Chrome, model.PageSpec) templ.Component) func(model.Chrome, model.PageSpec) templ.Component {
	return func(c model.Chrome, p model.PageSpec) templ.Component { return frame(c, p, header, footer) }
}

func LongDate(ctx context.Context, d model.Date) string {
	c, _ := model.ChromeOf(ctx)
	return c.Dates.LongDate(d)
}

func ShortDate(ctx context.Context, d model.Date) string {
	c, _ := model.ChromeOf(ctx)
	return c.Dates.ShortDate(d)
}

func MonthYear(ctx context.Context, d model.Date) string {
	c, _ := model.ChromeOf(ctx)
	return c.Dates.MonthYear(d)
}

// CodeBlock wraps escaped or highlighted code as the block the default look shows, with its copy button.
func CodeBlock(code string) string {
	return codeOpen + "<pre><code>" + code + "</code></pre></div>"
}

func Code(text string) templ.Component {
	return templ.Raw(CodeBlock(templ.EscapeString(text)))
}

func CurrentYear() string { return strconv.Itoa(time.Now().Year()) }

func ReadingTime(words int) string {
	if m := max(1, (words+199)/200); m > 1 {
		return fmt.Sprintf("%d min read", m)
	}
	return "1 min read"
}
