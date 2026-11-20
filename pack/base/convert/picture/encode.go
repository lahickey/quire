package picture

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	webp "github.com/gen2brain/webp"

	"github.com/lahickey/quire/bake"
	"github.com/lahickey/quire/bundle"
)

const (
	Version    = 1
	webpMethod = 4
)

var (
	formatJPEG         = bundle.Format{Ext: "jpg", Type: "image/jpeg", Quality: 82}
	formatWebPLossless = bundle.Format{Ext: "webp", Type: "image/webp", Lossless: true}

	formatWebP = bundle.Format{Ext: "webp", Type: "image/webp", Quality: 78}

	encodes atomic.Int64

	buildInfo = sync.OnceValues(debug.ReadBuildInfo)

	encoders = map[string]encoder{
		formatJPEG.Ext: jpegEncoder{},
		formatWebP.Ext: webpEncoder{},
	}
)

type encoder interface {
	encode(img image.Image, f bundle.Format) ([]byte, error)

	codec() string
}

type jpegEncoder struct{}

// Codec reads a master in any format Go decodes and writes its rungs as WebP.
type Codec struct{}

type opened struct {
	frame *image.NRGBA
	src   []byte
}

type webpEncoder struct{}

func moduleVersion(path string) string {
	bi, ok := buildInfo()
	if !ok {
		return ""
	}
	if bi.Main.Path == path {
		return bi.Main.Version
	}
	for _, d := range bi.Deps {
		if d.Path != path {
			continue
		}
		if d.Replace != nil {
			return strings.TrimPrefix(d.Replace.Path+"@"+d.Replace.Version, "@")
		}
		return d.Version
	}
	return ""
}

func (jpegEncoder) codec() string { return "stdlib/jpeg@" + runtime.Version() }

func (jpegEncoder) encode(img image.Image, f bundle.Format) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: f.Quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (webpEncoder) codec() string {
	return "gen2brain/webp@" + moduleVersion("github.com/gen2brain/webp") +
		" m=" + strconv.Itoa(webpMethod)
}

func (webpEncoder) encode(img image.Image, f bundle.Format) ([]byte, error) {
	var buf bytes.Buffer
	opts := webp.Options{Quality: f.Quality, Lossless: f.Lossless, Method: webpMethod}
	if err := webp.Encode(&buf, img, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func id(f bundle.Format) string {
	return fmt.Sprintf("%s|%d|%t|%s", f.Ext, f.Quality, f.Lossless, encoders[f.Ext].codec())
}

func master(name string, src []byte) (bundle.MasterMeta, string, []byte, error) {
	frame, err := read(name, src)
	if err != nil {
		return bundle.MasterMeta{}, "", nil, err
	}
	m := describe(frame, src)

	mf := formatJPEG
	if m.Alpha {
		mf = formatWebPLossless
	}

	body, err := encoders[mf.Ext].encode(frame, mf)
	if err != nil {
		return bundle.MasterMeta{}, "", nil, fmt.Errorf("quire/bake: %s: %w", name, err)
	}
	m.Hash = bundle.Hash(body)
	return m, mf.Ext, body, nil
}

func Encodes() int64 { return encodes.Load() }

func (Codec) Version() int { return Version }

func (Codec) Format() bundle.Format { return formatWebP }

func (Codec) ID() string { return id(formatWebP) }

func (Codec) Open(name string, src []byte) (bake.Master, error) {
	frame, err := read(name, src)
	return opened{frame, src}, err
}

func (o opened) Meta() bundle.MasterMeta { return describe(o.frame, o.src) }

func (o opened) Rung(width int) ([]byte, error) {
	encodes.Add(1)
	return encoders[formatWebP.Ext].encode(resizeWidth(o.frame, width), formatWebP)
}
