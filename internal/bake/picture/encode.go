package picture

import (
	"bytes"
	"image"
	"image/jpeg"
	"runtime"
	"strconv"
	"sync/atomic"

	webp "github.com/gen2brain/webp"

	"github.com/lahickey/quire/internal/journal"
)

const webpMethod = 4

var encodes atomic.Int64

type encoder interface {
	ext() string
	encode(img image.Image, f journal.Format) ([]byte, error)

	codec() string
}

type jpegEncoder struct{}

type webpEncoder struct{}

func (jpegEncoder) ext() string { return formatJPEG.Ext }

func (jpegEncoder) codec() string { return "stdlib/jpeg@" + runtime.Version() }

func (jpegEncoder) encode(img image.Image, f journal.Format) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: f.Quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (webpEncoder) ext() string { return formatWebP.Ext }

func (webpEncoder) codec() string {
	return "gen2brain/webp@" + moduleVersion("github.com/gen2brain/webp") +
		" m=" + strconv.Itoa(webpMethod)
}

func (webpEncoder) encode(img image.Image, f journal.Format) ([]byte, error) {
	var buf bytes.Buffer
	opts := webp.Options{Quality: f.Quality, Lossless: f.Lossless, Method: webpMethod}
	if err := webp.Encode(&buf, img, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func builtinEncoders() map[string]encoder {
	return map[string]encoder{
		formatJPEG.Ext: jpegEncoder{},
		formatWebP.Ext: webpEncoder{},
	}
}

func resizeTo(src image.Image, width int) image.Image {
	if src.Bounds().Dx() == width {
		return src
	}
	return resizeWidth(src, width)
}

func Encodes() int64 { return encodes.Load() }
