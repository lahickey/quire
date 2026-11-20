package picture

import (
	"bytes"
	"encoding/binary"
	"image"
	"io"

	xdraw "golang.org/x/image/draw"
)

var orientations = map[int]struct {
	swap bool
	from func(x, y, w, h int) (int, int)
}{
	2: {false, func(x, y, w, h int) (int, int) { return w - 1 - x, y }},
	3: {false, func(x, y, w, h int) (int, int) { return w - 1 - x, h - 1 - y }},
	4: {false, func(x, y, w, h int) (int, int) { return x, h - 1 - y }},
	5: {true, func(x, y, w, h int) (int, int) { return y, x }},
	6: {true, func(x, y, w, h int) (int, int) { return y, h - 1 - x }},
	7: {true, func(x, y, w, h int) (int, int) { return w - 1 - y, h - 1 - x }},
	8: {true, func(x, y, w, h int) (int, int) { return w - 1 - y, x }},
}

func exifOrientation(r *bytes.Reader, n int64) int {
	start, _ := r.Seek(0, io.SeekCurrent)
	defer func() { _, _ = r.Seek(start+n, io.SeekStart) }()
	var header uint32
	if binary.Read(r, binary.BigEndian, &header) != nil || header != 0x45786966 {
		return 1
	}
	if _, err := r.Seek(2, io.SeekCurrent); err != nil {
		return 1
	}
	var order uint16
	if binary.Read(r, binary.BigEndian, &order) != nil {
		return 1
	}
	var endian binary.ByteOrder
	switch order {
	case 0x4d4d:
		endian = binary.BigEndian
	case 0x4949:
		endian = binary.LittleEndian
	default:
		return 1
	}
	if _, err := r.Seek(2, io.SeekCurrent); err != nil {
		return 1
	}
	var off uint32
	if binary.Read(r, endian, &off) != nil || off < 8 {
		return 1
	}
	if _, err := r.Seek(int64(off-8), io.SeekCurrent); err != nil {
		return 1
	}
	var tags uint16
	if binary.Read(r, endian, &tags) != nil {
		return 1
	}
	for range int(tags) {
		var tag uint16
		if binary.Read(r, endian, &tag) != nil {
			return 1
		}
		if tag != 0x0112 {
			if _, err := r.Seek(10, io.SeekCurrent); err != nil {
				return 1
			}
			continue
		}
		if _, err := r.Seek(6, io.SeekCurrent); err != nil {
			return 1
		}
		var v uint16
		if binary.Read(r, endian, &v) != nil || v < 1 || v > 8 {
			return 1
		}
		return int(v)
	}
	return 1
}

func jpegOrientation(src []byte) int {
	r := bytes.NewReader(src)
	var soi uint16
	if binary.Read(r, binary.BigEndian, &soi) != nil || soi != 0xffd8 {
		return 1
	}
	for {
		var marker, size uint16
		if binary.Read(r, binary.BigEndian, &marker) != nil || marker>>8 != 0xff {
			return 1
		}
		if binary.Read(r, binary.BigEndian, &size) != nil || size < 2 {
			return 1
		}
		payload := int64(size - 2)
		if marker == 0xffe1 {
			return exifOrientation(r, payload)
		}
		if _, err := r.Seek(payload, io.SeekCurrent); err != nil {
			return 1
		}
	}
}

func toNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok && n.Rect.Min == (image.Point{}) {
		return n
	}
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	xdraw.Draw(dst, dst.Bounds(), src, b.Min, xdraw.Src)
	return dst
}

func pixAt(src *image.NRGBA, x, y int) []byte {
	i := src.PixOffset(x, y)
	return src.Pix[i : i+4]
}

func applyOrientation(img image.Image, o int) image.Image {
	m, ok := orientations[o]
	if !ok {
		return img
	}
	s := toNRGBA(img)
	w, h := s.Bounds().Dx(), s.Bounds().Dy()
	dw, dh := w, h
	if m.swap {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := range dh {
		for x := range dw {
			sx, sy := m.from(x, y, w, h)
			copy(pixAt(dst, x, y), pixAt(s, sx, sy))
		}
	}
	return dst
}
