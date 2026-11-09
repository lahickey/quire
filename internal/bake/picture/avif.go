package picture

import (
	"bytes"
	"image"
	"strconv"

	"github.com/gen2brain/avif"
	"github.com/lahickey/quire/internal/journal"
)

// Measured over a whole ladder: 7 costs twice the bake for 14% fewer bytes,
// below 7 buys nothing, and 9 is 45% bigger.
const defaultSpeed = 8

type avifEnc struct{ speed int }

func avifEncoder(speed int) avifEnc {
	if speed == 0 {
		speed = defaultSpeed
	}
	return avifEnc{speed: min(max(speed, 1), 9)}
}

func (avifEnc) ext() string { return formatAVIF.Ext }

func (e avifEnc) codec() string {
	return "gen2brain/avif@" + moduleVersion("github.com/gen2brain/avif") +
		" speed=" + strconv.Itoa(e.speed)
}

func (e avifEnc) encode(img image.Image, f journal.Format) ([]byte, error) {
	var buf bytes.Buffer
	if err := avif.Encode(&buf, img, avif.Options{Quality: f.Quality, Speed: e.speed}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
