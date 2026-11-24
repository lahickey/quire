// Package convert is base's bake side: the converters and the codec a bake of the default look is handed.
package convert

import (
	"github.com/lahickey/quire/bake"
	"github.com/lahickey/quire/pack/base/convert/card"
	"github.com/lahickey/quire/pack/base/convert/markdown"
	"github.com/lahickey/quire/pack/base/convert/mermaid"
	"github.com/lahickey/quire/pack/base/convert/picture"
)

func Standard() bake.Tools {
	return bake.Tools{
		Documents:  map[string]bake.Converter{".md": markdown.Converter{}},
		Converters: []bake.Converter{mermaid.Converter{}, card.New()},
		Pictures:   picture.Codec{},
	}
}
