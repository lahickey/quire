// Package convert is base's bake side: the converters and the codec a bake of the default kind of site is handed.
package convert

import (
	"github.com/lahickey/quire/bake"
	"github.com/lahickey/quire/internal/fonts/measure"
	"github.com/lahickey/quire/pack/base/convert/card"
	"github.com/lahickey/quire/pack/base/convert/markdown"
	"github.com/lahickey/quire/pack/base/convert/mermaid"
	"github.com/lahickey/quire/pack/base/convert/picture"
)

func Standard() bake.Tools {
	faces := measure.Default()
	return bake.Tools{
		Documents:  map[string]bake.Converter{".md": markdown.Converter{}},
		Converters: []bake.Converter{mermaid.New(faces), card.New(faces)},
		Pictures:   picture.Codec{},
	}
}
