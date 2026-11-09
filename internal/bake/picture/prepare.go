package picture

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/png"
	"sync/atomic"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/lahickey/quire/internal/journal"
)

var decodes atomic.Int64

func decodeImg(src []byte) (image.Image, error) {
	decodes.Add(1)
	img, _, err := image.Decode(bytes.NewReader(src))
	return img, err
}

func scale(src image.Image, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

func fit(src image.Image, maxEdge int) *image.NRGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return image.NewNRGBA(image.Rectangle{})
	}
	if w <= maxEdge && h <= maxEdge {
		return toNRGBA(src)
	}
	if w >= h {
		return scale(src, maxEdge, max(1, (h*maxEdge+w/2)/w))
	}
	return scale(src, max(1, (w*maxEdge+h/2)/h), maxEdge)
}

func resizeWidth(src image.Image, width int) *image.NRGBA {
	b := src.Bounds()
	if b.Dx() == width {
		return toNRGBA(src)
	}
	return scale(src, width, max(1, (b.Dy()*width+b.Dx()/2)/b.Dx()))
}

func Read(name string, src []byte, cap int) (*image.NRGBA, error) {
	if len(src) == 0 {
		return nil, fmt.Errorf("quire: image %s: empty", name)
	}
	img, err := decodeImg(src)
	if err != nil {
		return nil, fmt.Errorf("quire: image %s: %w", name, err)
	}
	frame := fit(applyOrientation(img, jpegOrientation(src)), cap)
	if frame.Bounds().Dx() < 1 || frame.Bounds().Dy() < 1 {
		return nil, fmt.Errorf("quire: image %s: empty frame", name)
	}
	return frame, nil
}

func Describe(frame *image.NRGBA, src []byte, hash string) journal.MasterMeta {
	m := journal.MasterMeta{
		Hash:  hash,
		W:     frame.Bounds().Dx(),
		H:     frame.Bounds().Dy(),
		Alpha: !frame.Opaque(),
	}

	if cfg, _, err := image.DecodeConfig(bytes.NewReader(src)); err == nil {
		m.SrcW, m.SrcH = cfg.Width, cfg.Height
		if o := jpegOrientation(src); o >= 5 && o <= 8 {
			m.SrcW, m.SrcH = m.SrcH, m.SrcW
		}
	}
	return m
}

func Decodes() int64 { return decodes.Load() }
