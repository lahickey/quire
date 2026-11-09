package testkit

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"

	"github.com/lahickey/quire/internal/journal"
)

func Photo(t *testing.T, w, h int, alpha bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			a := uint8(255)
			if alpha {
				a = uint8(max(0, min(255, 320-int(math.Hypot(float64(x-w/2), float64(y-h/2))))))
			}
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(128 + 100*math.Sin(float64(x)/13)*math.Cos(float64(y)/17)),
				G: uint8(128 + 90*math.Sin(float64(x+y)/23)),
				B: uint8(128 + 80*math.Cos(float64(x-y)/29)),
				A: a,
			})
		}
	}
	return encode(t, img)
}

func Flat(t *testing.T, w, h int, alpha bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			a := uint8(255)
			if alpha && x == 0 && y == 0 {
				a = 0
			}
			img.SetNRGBA(x, y, color.NRGBA{R: 200, G: 40, B: 40, A: a})
		}
	}
	return encode(t, img)
}

func encode(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func ReadJournal(t *testing.T, dir string) map[string]journal.Record {
	t.Helper()
	recs, err := journal.LoadJournal(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	return recs
}
