// Package card draws a share card as a PNG, in Inter or in the two faces it is handed.
package card

import (
	"bytes"
	_ "embed"
	"encoding/json/v2"
	"fmt"
	"image"
	"image/color"
	imgdraw "image/draw"
	"image/png"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/lahickey/quire/bake"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/model/radix"
)

const (
	Version = 2
	height  = 630
)

type pen struct {
	dst  imgdraw.Image
	font *sfnt.Font
}

type Converter struct{ regular, bold *sfnt.Font }

var (
	//go:embed fonts/inter-400.ttf
	inter400 []byte
	//go:embed fonts/inter-700.ttf
	inter700 []byte
)

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
	for w := range strings.FieldsSeq(s) {
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

func draw(c model.ShareCard, plain, heavy *sfnt.Font) (image.Image, error) {
	const margin, width = 96, model.ShareWidth - 2*96
	ink := map[string]color.Color{}
	n := radix.Scales[c.Palette.Neutral].Dark
	for name, hex := range map[string]string{"bg": n[0], "text": n[11], "secondary": n[10], "muted": n[9], "accent": radix.Scales[c.Palette.Accent].Dark[10]} {
		col, err := hexColor(hex)
		if err != nil {
			return nil, err
		}
		ink[name] = col
	}
	img := image.NewRGBA(image.Rect(0, 0, model.ShareWidth, height))
	imgdraw.Draw(img, img.Bounds(), image.NewUniform(ink["bg"]), image.Point{}, imgdraw.Src)
	bold, regular := pen{img, heavy}, pen{img, plain}

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
		if y > height-130 {
			break
		}
		if err := regular.text(l, 26, ink["secondary"], margin, y, 0, false); err != nil {
			return nil, err
		}
		y += 36
	}
	if err := regular.text(c.Domain, 20, ink["muted"], model.ShareWidth-margin, height-88, 0, true); err != nil {
		return nil, err
	}
	return img, nil
}

// With is the converter drawing in a regular and a bold face, each a TTF.
func With(regular, bold []byte) (Converter, error) {
	var c Converter
	var err error
	if c.regular, err = sfnt.Parse(regular); err != nil {
		return c, fmt.Errorf("card: regular face: %w", err)
	}
	if c.bold, err = sfnt.Parse(bold); err != nil {
		return c, fmt.Errorf("card: bold face: %w", err)
	}
	return c, nil
}

// New is the converter drawing in Inter.
func New() Converter {
	c, err := With(inter400, inter700)
	if err != nil {
		panic(err)
	}
	return c
}

func (Converter) Name() string { return "card" }

func (Converter) Version() int { return Version }

// Convert takes a model.ShareCard as JSON.
func (c Converter) Convert(in bake.Input) (bake.Output, error) {
	var card model.ShareCard
	if err := json.Unmarshal(in.Body, &card); err != nil {
		return bake.Output{}, err
	}
	img, err := draw(card, c.regular, c.bold)
	if err != nil {
		return bake.Output{}, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return bake.Output{}, err
	}
	return bake.Output{Body: buf.Bytes(), Ext: "png", Type: "image/png"}, nil
}
