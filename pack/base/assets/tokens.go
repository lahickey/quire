// Package assets is the default look's stylesheets and scripts, and the CSS made from a site's scale and palette.
package assets

import (
	"embed"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/model/radix"
)

//go:embed *.css *.js
var Files embed.FS

var (
	Sheets  = []string{"base.css", "toggle.css", "nav.css", "layout.css", "prose.css", "highlight.css"}
	Scripts = []string{"theme.js", "nav.js", "video.js"}

	bright = []string{"sky", "mint", "lime", "yellow", "amber"}
)

type cssVar struct{ name, val string }

func rootVars(tokens []cssVar) []byte {
	var b strings.Builder
	b.WriteString(":root {\n")
	for _, t := range tokens {
		fmt.Fprintf(&b, "  %s: %s;\n", t.name, t.val)
	}
	b.WriteString("}\n")
	return []byte(b.String())
}

// ScaleCSS is the custom properties every sheet reads its widths from.
func ScaleCSS(s model.Scale) []byte {
	return rootVars([]cssVar{
		{"--qr-text-root", strconv.Itoa(s.Root) + "px"},
		{"--qr-col", s.Rem(s.Col)},
		{"--qr-gutter", s.Rem(s.Gutter())},
		{"--qr-measure", strconv.Itoa(s.Measure()) + "px"},
		{"--qr-measure-wide", strconv.Itoa(s.Wide()) + "px"},
		{"--qr-measure-full", strconv.Itoa(s.Full()) + "px"},
		{"--qr-thumb", s.Rem(s.Thumb())},
		{"--qr-nav-columns", "1fr auto 1fr"},
		{"--qr-nav-links-column", "2"},
		{"--qr-nav-end-column", "3"},
		{"--qr-nav-justify", "center"},
		{"--qr-bleed", "calc(50% - 50cqw)"},
	})
}

// BreakpointCSS is what changes on a narrow screen. It goes after the sheets it overrides.
func BreakpointCSS(s model.Scale) []byte {
	inner := rootVars([]cssVar{
		{"--qr-text-xl", "2rem"},
		{"--qr-text-lg", "1.5rem"},
		{"--qr-nav-columns", "minmax(0, 1fr) auto"},
		{"--qr-nav-links-column", "1"},
		{"--qr-nav-end-column", "2"},
		{"--qr-nav-justify", "flex-start"},
	})
	return fmt.Appendf(nil, "@media (width < %dpx) {\n%s}\n", s.Measure(), inner)
}

func contrast(hue string) string {
	if slices.Contains(bright, hue) {
		return radix.Scales[hue].Dark[0]
	}
	return "#fff"
}

func paletteBlock(b *strings.Builder, sel string, p model.Palette, light bool) {
	fmt.Fprintf(b, "%s {\n", sel)
	for _, r := range p.Roles() {
		steps := radix.Scales[r.Hue].Dark
		if light {
			steps = radix.Scales[r.Hue].Light
		}
		for i, v := range steps {
			if r.Range {
				fmt.Fprintf(b, "  --qr-%s-%d: %s;\n", r.Name, i+1, v)
			}
		}
		for _, s := range model.Steps {
			if !r.Range {
				fmt.Fprintf(b, "  --qr-%s-%s: %s;\n", r.Name, s.Name, steps[s.N-1])
			}
		}
		if !light {
			fmt.Fprintf(b, "  --qr-%s-contrast: %s;\n", r.Name, contrast(r.Hue))
		}
	}
	b.WriteString("}\n")
}

// PaletteCSS is the role properties for the site's palette, and for each the picker offers.
func PaletteCSS(site model.Palette, offered []model.Palette) []byte {
	var b strings.Builder
	const light = `[data-theme="light"]`
	named := func(p model.Palette) string { return fmt.Sprintf("[data-palette=%q]", p.ID()) }
	root, rootLight := ":root", light
	if slices.Contains(offered, site) {
		root += ", " + named(site)
		rootLight += ", " + light + " " + named(site)
	}
	paletteBlock(&b, root, site, false)
	paletteBlock(&b, rootLight, site, true)
	b.WriteString(":root { color-scheme: dark; }\n[data-theme=\"light\"] { color-scheme: light; }\n")
	for _, p := range offered {
		if p != site {
			paletteBlock(&b, named(p), p, false)
			paletteBlock(&b, named(p)+light+", "+light+" "+named(p), p, true)
		}
	}
	if len(offered) > 0 {
		picked := []string{fmt.Sprintf(":root:not([data-palette]) .palette-menu > [value=%q]", site.ID())}
		for _, p := range offered {
			picked = append(picked, fmt.Sprintf("%s .palette-menu > [value=%q]", named(p), p.ID()))
		}
		fmt.Fprintf(&b, "%s { background: var(--qr-accent-3); }\n", strings.Join(picked, ",\n"))
	}
	return []byte(b.String())
}
