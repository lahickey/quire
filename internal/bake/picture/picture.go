package picture

import (
	"fmt"
	"image"

	"github.com/lahickey/quire/internal/journal"
)

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

func Encode(img image.Image, width int, f journal.Format) ([]byte, error) {
	encodes.Add(1)
	return encoders[f.Ext].encode(resizeWidth(img, width), f)
}

func Variants(m journal.MasterMeta) []journal.Variants {
	f := formatWebP
	vs := journal.Variants{Ext: f.Ext, Type: f.Type, Q: f.Quality}
	for _, w := range journal.LadderFor(m.W) {
		vs.Out = append(vs.Out, journal.Variant{Name: fileName(m.Hash, w, f, encoders[f.Ext].codec()), W: w})
	}
	return []journal.Variants{vs}
}
