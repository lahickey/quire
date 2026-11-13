package picture

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/internal/testkit"
)

func jpegOf(t *testing.T, src []byte) []byte {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: formatJPEG.Quality}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func exifJPEG(t *testing.T, body []byte, orientation byte) []byte {
	t.Helper()
	tiff := []byte{
		'M', 'M', 0x00, 0x2a, 0x00, 0x00, 0x00, 0x08,
		0x00, 0x01,
		0x01, 0x12, 0x00, 0x03, 0x00, 0x00, 0x00, 0x01, 0x00, orientation, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	payload := append(append([]byte("Exif\x00\x00"), tiff...), []byte("GPS-51.5074N-0.1278W")...)
	n := 2 + len(payload)
	app1 := append([]byte{0xff, 0xe1, byte(n >> 8), byte(n)}, payload...)
	icc := append([]byte("ICC_PROFILE\x00\x01\x01"), []byte("DISPLAY-P3-PROFILE-BYTES")...)
	m := 2 + len(icc)
	app2 := append([]byte{0xff, 0xe2, byte(m >> 8), byte(m)}, icc...)
	return append([]byte{0xff, 0xd8}, append(append(app1, app2...), body[2:]...)...)
}

func ingested(t *testing.T, name string, src []byte) (journal.MasterMeta, []byte) {
	t.Helper()
	dir := t.TempDir()
	m, file, err := Ingest(dir, name, src)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		t.Fatal(err)
	}
	return m, body
}

func TestVariantNameCoversTheEncode(t *testing.T) {
	softer := journal.Format{Ext: "jpg", Type: "image/jpeg", Quality: 70}
	same := fileName("h", 1280, formatJPEG, "c")
	for what, got := range map[string]string{
		"quality": fileName("h", 1280, softer, "c"),
		"width":   fileName("h", 704, formatJPEG, "c"),
		"master":  fileName("other", 1280, formatJPEG, "c"),
		"format":  fileName("h", 1280, FormatWebP, "c"),
		"codec":   fileName("h", 1280, formatJPEG, "c@2"),
	} {
		if got == same {
			t.Errorf("%s does not change the variant name", what)
		}
	}
}

func TestIngestRejectsBadName(t *testing.T) {
	src := testkit.Flat(t, 8, 8, false)
	for _, name := range []string{"", ".", "..", "a/b", "/abs"} {
		if _, _, err := Ingest(t.TempDir(), name, src); err == nil {
			t.Errorf("%q accepted", name)
		}
	}
}

func TestIngestOrientation(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 4))
	for y := range 4 {
		for x := range 2 {
			src.SetNRGBA(x, y, color.NRGBA{R: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	oriented := exifJPEG(t, buf.Bytes(), 6)
	im, _, err := Ingest(t.TempDir(), "phone.jpg", oriented)
	if err != nil {
		t.Fatal(err)
	}
	if im.W != 4 || im.H != 2 {
		t.Fatalf("orientation 6: %dx%d, want 4x2", im.W, im.H)
	}
}

func TestMasterCarriesNoProvenance(t *testing.T) {
	src := exifJPEG(t, jpegOf(t, testkit.Photo(t, 400, 300, false)), 1)
	_, body := ingested(t, "holiday.jpg", src)
	for _, secret := range []string{"GPS-51.5074N", "Exif", "ICC_PROFILE", "DISPLAY-P3"} {
		if bytes.Contains(body, []byte(secret)) {
			t.Errorf("master still carries %q", secret)
		}
	}
	if len(body) >= len(src) {
		t.Errorf("master %d B is not smaller than source %d B", len(body), len(src))
	}
}
