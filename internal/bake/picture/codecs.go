package picture

import (
	"fmt"

	"github.com/lahickey/quire/internal/journal"
)

var (
	formatJPEG         = journal.Format{Ext: "jpg", Type: "image/jpeg", Quality: 82}
	formatWebPLossless = journal.Format{Ext: "webp", Type: "image/webp", Lossless: true}

	FormatWebP = journal.Format{Ext: "webp", Type: "image/webp", Quality: 78}
	FormatPNG  = journal.Format{Ext: "png", Type: "image/png", Lossless: true}
)

func fileName(master string, width int, f journal.Format, codec string) string {
	return journal.Hash(fmt.Appendf(nil, "%s|%d|%s|%d|%t|%s", master, width, f.Ext, f.Quality, f.Lossless, codec)) + "." + f.Ext
}
