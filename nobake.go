//go:build quire_nobake

package quire

import (
	"errors"

	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/site"
)

const canBake = false

func Bake(*site.Description, model.BakeSpec) error {
	return errors.New("quire: built with quire_nobake, so this binary serves a bundle and cannot bake one")
}
