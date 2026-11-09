package assets

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/lahickey/quire/internal/radix"
	"github.com/lahickey/quire/model"
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

func media(s model.Scale, w model.Width) string {
	switch w {
	case model.WidthNarrow:
		return fmt.Sprintf("@media (width < %dpx)", s.Measure())
	case model.WidthWide:
		return fmt.Sprintf("@media (width >= %dpx)", s.Wide())
	case model.WidthFull:
		return fmt.Sprintf("@media (width >= %dpx)", s.Full())
	}
	return ""
}

func scaleCSS(s model.Scale) []byte {
	tokens := []cssVar{
		{"--qr-text-root", strconv.Itoa(s.Root) + "px"},
		{"--qr-col", s.Rem(s.Col)},
		{"--qr-gutter", "1.5rem"},
		{"--qr-measure", strconv.Itoa(s.Measure()) + "px"},
		{"--qr-thumb", "4rem"},
		{"--qr-nav-columns", "1fr auto 1fr"},
		{"--qr-nav-links-column", "2"},
		{"--qr-nav-end-column", "3"},
		{"--qr-nav-justify", "center"},
	}
	if s.Widest >= model.WidthWide {
		tokens = append(tokens, cssVar{"--qr-measure-wide", strconv.Itoa(s.Wide()) + "px"})
	}
	if s.Widest >= model.WidthFull {
		tokens = append(tokens, cssVar{"--qr-measure-full", strconv.Itoa(s.Full()) + "px"})
	}
	tokens = append(tokens, cssVar{"--qr-bleed", "calc(50% - 50cqw)"})
	return rootVars(tokens)
}

func breakpointCSS(s model.Scale) []byte {
	inner := rootVars([]cssVar{
		{"--qr-text-xl", "2rem"},
		{"--qr-text-lg", "1.5rem"},
		{"--qr-nav-columns", "minmax(0, 1fr) auto"},
		{"--qr-nav-links-column", "1"},
		{"--qr-nav-end-column", "2"},
		{"--qr-nav-justify", "flex-start"},
	})
	return fmt.Appendf(nil, "%s {\n%s}\n", media(s, model.WidthNarrow), inner)
}

var bright = []string{"sky", "mint", "lime", "yellow", "amber"}

func contrast(hue string) string {
	if slices.Contains(bright, hue) {
		return radix.Scales[hue].Dark[0]
	}
	return "#fff"
}

func paletteCSS(p model.Palette) []byte {
	var b strings.Builder
	block := func(sel, scheme string, steps func(radix.Scale) [12]string, contrasts bool) {
		fmt.Fprintf(&b, "%s {\n", sel)
		for _, r := range p.Roles() {
			for i, v := range steps(radix.Scales[r.Hue]) {
				fmt.Fprintf(&b, "  --qr-%s-%d: %s;\n", r.Name, i+1, v)
			}
			if contrasts {
				fmt.Fprintf(&b, "  --qr-%s-contrast: %s;\n", r.Name, contrast(r.Hue))
			}
		}
		fmt.Fprintf(&b, "  color-scheme: %s;\n}\n", scheme)
	}
	block(":root", "dark", func(s radix.Scale) [12]string { return s.Dark }, true)
	b.WriteString("\n")
	block(`[data-theme="light"]`, "light", func(s radix.Scale) [12]string { return s.Light }, false)
	return []byte(b.String())
}
