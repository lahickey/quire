package base

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base/ui"
	"github.com/lahickey/quire/site"
	"go.yaml.in/yaml/v3"
)

func cards(_, body string) (string, error) {
	var written []struct{ Href, Kicker, Title, Description, CTA string }
	dec := yaml.NewDecoder(strings.NewReader(body))
	dec.KnownFields(true)
	if err := dec.Decode(&written); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	list := make([]ui.CardData, len(written))
	for i, c := range written {
		list[i] = ui.CardData{Href: c.Href, Kicker: c.Kicker, Title: c.Title, Description: c.Description, CTA: c.CTA}
	}
	var b strings.Builder
	err := ui.CardGrid(list).Render(context.Background(), &b)
	return b.String(), err
}

// paletteGrid draws a grid of colours: down the side and across the top are
// what it is labelled with, and each cell is the custom property named.
func paletteGrid(b *strings.Builder, name string, down, across []string, property func(down, across string) string) {
	if len(down) == 0 || len(across) == 0 {
		return
	}
	attr := ""
	if name != "" {
		attr = ` data-palette="` + name + `"`
	}
	fmt.Fprintf(b, `<div class="palette" style="grid-template-columns:max-content repeat(%d,1fr)"%s><span class="palette-label">%s</span>`, len(across), attr, name)
	for _, a := range across {
		fmt.Fprintf(b, `<span>%s</span>`, a)
	}
	for _, d := range down {
		fmt.Fprintf(b, `<span class="palette-label">%s</span>`, d)
		for _, a := range across {
			fmt.Fprintf(b, `<span style="background:var(--qr-%s)"></span>`, property(d, a))
		}
	}
	b.WriteString(`</div>`)
}

// palette draws the roles its body names: each range as a row of its twelve
// steps, and the rest side by side with a row for each of their steps.
func palette(name, body string) (string, error) {
	roles := (model.Palette{}).Roles()
	var names, ranges, others, twelve, steps []string
	for _, r := range roles {
		names = append(names, r.Name)
	}
	for word := range strings.FieldsSeq(body) {
		i := slices.Index(names, word)
		switch {
		case word == "all":
			for _, r := range roles {
				if r.Range {
					ranges = append(ranges, r.Name)
				} else {
					others = append(others, r.Name)
				}
			}
		case i < 0:
			return "", fmt.Errorf("palette: unknown role %q, want all or one of %s", word, strings.Join(names, " "))
		case roles[i].Range:
			ranges = append(ranges, word)
		default:
			others = append(others, word)
		}
	}
	for i := range 12 {
		twelve = append(twelve, strconv.Itoa(i+1))
	}
	for _, s := range model.Steps {
		steps = append(steps, s.Name)
	}
	var b strings.Builder
	name = html.EscapeString(name)
	paletteGrid(&b, name, ranges, twelve, func(role, step string) string { return role + "-" + step })
	paletteGrid(&b, name, steps, others, func(step, role string) string { return role + "-" + step })
	return b.String(), nil
}

func mermaid(s *site.Site) model.Widget {
	return func(_, body string) (string, error) {
		file, err := s.Converted("mermaid", []byte(strings.TrimSpace(body)), s.FontSetID())
		if err != nil {
			return "", err
		}
		svg, err := s.Body(file)
		return `<figure class="diagram">` + string(svg) + "</figure>", err
	}
}

func picture(pic model.Picture, sl model.PictureSlot) templ.Component {
	if sl.Figure {
		return ui.LazyFigure(pic, sl.Alt, sl.Caption)
	}
	return ui.Img(pic, model.SizePage, sl.Alt)
}

func video(src string, poster model.Picture, sl model.PictureSlot) templ.Component {
	return ui.VideoFigure(src, poster, sl.Alt, sl.Caption)
}
