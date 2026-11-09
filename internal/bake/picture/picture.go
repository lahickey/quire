package picture

import (
	"cmp"
	"fmt"
	"image"
	"path"
	"slices"
	"strings"

	"github.com/lahickey/quire/internal/journal"
)

var masterExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	".gif": true, ".tif": true, ".tiff": true,
}

func IsMaster(name string) bool { return masterExt[strings.ToLower(path.Ext(name))] }

func total(vs journal.Variants) int64 {
	var n int64
	for _, v := range vs.Out {
		n += v.Size
	}
	return n
}

func Order(fmts []journal.Variants) []journal.Variants {
	slices.SortStableFunc(fmts[:len(fmts)-1], func(a, b journal.Variants) int { return cmp.Compare(total(a), total(b)) })
	return fmts
}

type Encoders map[string]encoder

func NewEncoders(offered []journal.Format, avifSpeed int) Encoders {
	encs := builtinEncoders()
	if slices.ContainsFunc(offered, func(f journal.Format) bool { return f.Ext == formatAVIF.Ext }) {
		encs[formatAVIF.Ext] = avifEncoder(avifSpeed)
	}
	return encs
}

func (e Encoders) Codecs() map[string]string {
	out := make(map[string]string, len(e))
	for ext, enc := range e {
		out[ext] = enc.codec()
	}
	return out
}

func (e Encoders) Encode(img image.Image, width int, f journal.Format) ([]byte, error) {
	enc, ok := e[f.Ext]
	if !ok {
		return nil, fmt.Errorf("quire/bake: no encoder for %s", f.Ext)
	}
	encodes.Add(1)
	return enc.encode(resizeTo(img, width), f)
}

func Variants(m journal.MasterMeta, lossless bool, offered []journal.Format, codecs map[string]string) []journal.Variants {
	formats := formatsFrom(lossless, m.Alpha, offered)
	out := make([]journal.Variants, 0, len(formats))
	rungs := journal.LadderFor(m.W)
	for _, f := range formats {
		vs := journal.Variants{Ext: f.Ext, Type: f.Type, Q: f.Quality, Less: f.Lossless}
		for _, w := range rungs {
			vs.Out = append(vs.Out, journal.Variant{Name: fileName(m.Hash, w, f, codecs[f.Ext]), W: w})
		}
		out = append(out, vs)
	}
	return out
}

func master(name string, src []byte, lossless bool) (journal.MasterMeta, string, []byte, error) {
	frame, err := Read(name, src, journal.MasterCap)
	if err != nil {
		return journal.MasterMeta{}, "", nil, err
	}
	m := Describe(frame, src, "")

	mf := formatJPEG
	if m.Alpha || lossless {
		mf = formatWebPLossless
	}

	body, err := builtinEncoders()[mf.Ext].encode(frame, mf)
	if err != nil {
		return journal.MasterMeta{}, "", nil, fmt.Errorf("quire/bake: %s: %w", name, err)
	}
	m.Hash = journal.Hash(body)
	return m, mf.Ext, body, nil
}
