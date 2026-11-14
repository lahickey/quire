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

	"github.com/lahickey/quire/internal/journal"
)

const webpMethod = 4

var (
	formatJPEG         = journal.Format{Ext: "jpg", Type: "image/jpeg", Quality: 82}
	formatWebPLossless = journal.Format{Ext: "webp", Type: "image/webp", Lossless: true}

	FormatWebP = journal.Format{Ext: "webp", Type: "image/webp", Quality: 78}
	FormatPNG  = journal.Format{Ext: "png", Type: "image/png", Lossless: true}

	encodes atomic.Int64

	buildInfo = sync.OnceValues(debug.ReadBuildInfo)

	encoders = map[string]encoder{
		formatJPEG.Ext: jpegEncoder{},
		FormatWebP.Ext: webpEncoder{},
	}
)

type encoder interface {
	encode(img image.Image, f journal.Format) ([]byte, error)

	codec() string
}

type jpegEncoder struct{}

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

func (jpegEncoder) encode(img image.Image, f journal.Format) ([]byte, error) {
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

func (webpEncoder) encode(img image.Image, f journal.Format) ([]byte, error) {
	var buf bytes.Buffer
	opts := webp.Options{Quality: f.Quality, Lossless: f.Lossless, Method: webpMethod}
	if err := webp.Encode(&buf, img, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fileName(master string, width int, f journal.Format, codec string) string {
	return journal.Hash(fmt.Appendf(nil, "%s|%d|%s|%d|%t|%s", master, width, f.Ext, f.Quality, f.Lossless, codec)) + "." + f.Ext
}

func master(name string, src []byte) (journal.MasterMeta, string, []byte, error) {
	frame, err := Read(name, src, journal.MasterCap)
	if err != nil {
		return journal.MasterMeta{}, "", nil, err
	}
	m := Describe(frame, src, "")

	mf := formatJPEG
	if m.Alpha {
		mf = formatWebPLossless
	}

	body, err := encoders[mf.Ext].encode(frame, mf)
	if err != nil {
		return journal.MasterMeta{}, "", nil, fmt.Errorf("quire/bake: %s: %w", name, err)
	}
	m.Hash = journal.Hash(body)
	return m, mf.Ext, body, nil
}

func Encodes() int64 { return encodes.Load() }

func Encode(img image.Image, width int, f journal.Format) ([]byte, error) {
	encodes.Add(1)
	return encoders[f.Ext].encode(resizeWidth(img, width), f)
}

func Variants(m journal.MasterMeta, f journal.Format) []journal.File {
	var out []journal.File
	for _, w := range journal.LadderFor(m.W) {
		out = append(out, journal.File{Name: fileName(m.Hash, w, f, encoders[f.Ext].codec()), W: w})
	}
	return out
}
