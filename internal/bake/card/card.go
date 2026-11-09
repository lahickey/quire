package card

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/lahickey/quire/internal/bake/diagram"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/model"
)

type pen struct {
	dst  draw.Image
	font *sfnt.Font
}

func hexColor(s string) (color.Color, error) {
	s = strings.TrimPrefix(s, "#")
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil || len(s) != 6 {
		return nil, fmt.Errorf("quire: card colour %q", s)
	}
	return color.RGBA{uint8(n >> 16), uint8(n >> 8), uint8(n), 0xff}, nil
}

func (p pen) face(size float64) (font.Face, error) {
	return opentype.NewFace(p.font, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
}

func wrap(f font.Face, s string, width fixed.Int26_6, max int) []string {
	var lines []string
	line := ""
	for _, w := range strings.Fields(s) {
		next := strings.TrimSpace(line + " " + w)
		if line != "" && font.MeasureString(f, next) > width {
			lines = append(lines, line)
			line = w
			continue
		}
		line = next
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) <= max {
		return lines
	}
	last := lines[max-1] + "…"
	for font.MeasureString(f, last) > width && utf8.RuneCountInString(last) > 1 {
		r := []rune(strings.TrimSuffix(last, "…"))
		last = strings.TrimSpace(string(r[:len(r)-1])) + "…"
	}
	return append(lines[:max-1], last)
}

func (p pen) text(s string, size float64, c color.Color, x, y int, track float64, right bool) error {
	f, err := p.face(size)
	if err != nil {
		return err
	}
	d := font.Drawer{Dst: p.dst, Src: image.NewUniform(c), Face: f}
	t := fixed.Int26_6(track * 64)
	if right {
		x -= (font.MeasureString(f, s) + t*fixed.Int26_6(utf8.RuneCountInString(s))).Round()
	}
	d.Dot = fixed.P(x, y)
	for _, r := range s {
		d.DrawString(string(r))
		d.Dot.X += t
	}
	return nil
}

func Draw(c model.ShareCard, r *diagram.Renderer) (image.Image, error) {
	const margin, width = 96, journal.ShareWidth - 2*96
	ink := map[string]color.Color{}
	for name, hex := range map[string]string{"bg": c.Ink.Bg, "text": c.Ink.Text, "secondary": c.Ink.Secondary, "muted": c.Ink.Muted, "accent": c.Ink.Accent} {
		col, err := hexColor(hex)
		if err != nil {
			return nil, err
		}
		ink[name] = col
	}
	img := image.NewRGBA(image.Rect(0, 0, journal.ShareWidth, journal.CardHeight))
	draw.Draw(img, img.Bounds(), image.NewUniform(ink["bg"]), image.Point{}, draw.Src)
	bold, regular := pen{img, r.Text(700)}, pen{img, r.Text(400)}

	y := 150
	if c.Kicker != "" {
		if err := bold.text(strings.ToUpper(c.Kicker), 22, ink["accent"], margin, y, 2.5, false); err != nil {
			return nil, err
		}
		y += 90
	}
	title, err := bold.face(64)
	if err != nil {
		return nil, err
	}
	for _, l := range wrap(title, c.Title, fixed.I(width), 3) {
		if err := bold.text(l, 64, ink["text"], margin, y, -1, false); err != nil {
			return nil, err
		}
		y += 76
	}
	body, err := regular.face(26)
	if err != nil {
		return nil, err
	}
	y += 20
	var lines []string
	for _, l := range c.Lines {
		lines = append(lines, wrap(body, l, fixed.I(width), 3)...)
	}
	for _, l := range lines {
		if y > journal.CardHeight-130 {
			break
		}
		if err := regular.text(l, 26, ink["secondary"], margin, y, 0, false); err != nil {
			return nil, err
		}
		y += 36
	}
	if err := regular.text(c.Domain, 20, ink["muted"], journal.ShareWidth-margin, journal.CardHeight-88, 0, true); err != nil {
		return nil, err
	}
	return img, nil
}
