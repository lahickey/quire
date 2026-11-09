package picture

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/lahickey/quire/internal/journal"
)

var (
	formatJPEG         = journal.Format{Ext: "jpg", Type: "image/jpeg", Quality: 82}
	formatAVIF         = journal.Format{Ext: "avif", Type: "image/avif", Quality: 55, Alpha: true}
	formatWebP         = journal.Format{Ext: "webp", Type: "image/webp", Quality: 78, Alpha: true}
	formatWebPLossless = journal.Format{Ext: "webp", Type: "image/webp", Alpha: true, Lossless: true}

	FormatPNG = journal.Format{Ext: "png", Type: "image/png", Alpha: true, Lossless: true}
)

var codecs = []struct {
	name   string
	format journal.Format
}{
	{"avif", formatAVIF},
	{"webp", formatWebP},
	{"jpg", formatJPEG},
}

func formatsFrom(lossless, alpha bool, offered []journal.Format) []journal.Format {
	if lossless {
		return []journal.Format{formatWebPLossless}
	}
	var out []journal.Format
	for _, f := range offered {
		if f.Alpha || !alpha {
			out = append(out, f)
		}
	}
	return out
}

func fileName(master string, width int, f journal.Format, codec string) string {
	return journal.Hash(fmt.Appendf(nil, "%s|%d|%s|%d|%t|%s", master, width, f.Ext, f.Quality, f.Lossless, codec)) + "." + f.Ext
}

func codecNames() []string {
	out := make([]string, 0, len(codecs))
	for _, c := range codecs {
		out = append(out, c.name)
	}
	return out
}

func Codecs(names ...string) []journal.Format {
	var out []journal.Format
	for _, c := range codecs {
		if slices.Contains(names, c.name) {
			out = append(out, c.format)
		}
	}
	return out
}

func ParseCodecs(list []string) ([]journal.Format, error) {
	quality := map[string]int{}
	var names []string
	for _, part := range list {
		name, q, hasQ := strings.Cut(strings.TrimSpace(part), "@")
		if !slices.Contains(codecNames(), name) {
			return nil, fmt.Errorf("quire/bake: unknown codec %q, want one of %s", name, strings.Join(codecNames(), ", "))
		}
		if hasQ {
			n, err := strconv.Atoi(q)
			if err != nil || n < 1 || n > 100 {
				return nil, fmt.Errorf("quire/bake: %s quality %q, want 1 to 100", name, q)
			}
			quality[name] = n
		}
		names = append(names, name)
	}
	out := Codecs(names...)
	if len(out) == 0 {
		return nil, fmt.Errorf("quire/bake: no codecs in %q", list)
	}
	for i := range out {
		if q, ok := quality[out[i].Ext]; ok {
			out[i].Quality = q
		}
	}
	return out, nil
}
