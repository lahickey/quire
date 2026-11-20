//go:build quire_nobake

package app

import (
	"errors"

	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
)

const canBake = false

func bakeSite(model.Identity, quire.Blueprint, []quire.Option) error {
	return errors.New("quire: built with quire_nobake, so this binary serves a bundle and cannot bake one")
}
