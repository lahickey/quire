// Package faces is the default font set's faces for measuring and drawing text, which only a bake needs.
package faces

import (
	"embed"
	"io/fs"

	"github.com/lahickey/quire/bake/measure"
	"github.com/lahickey/quire/model"
	basefonts "github.com/lahickey/quire/pack/base/fonts"
)

//go:embed fonts
var files embed.FS

func Default() *measure.Faces {
	set, err := model.ReadFontSet(basefonts.Web())
	if err != nil {
		panic("quire: default fonts: " + err.Error())
	}
	ttf, err := fs.Sub(files, "fonts")
	if err != nil {
		panic(err)
	}
	f, err := measure.Read(set, ttf)
	if err != nil {
		panic("quire: default fonts: " + err.Error())
	}
	return f
}
