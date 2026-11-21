//go:build quire_nobake

package app

import (
	"errors"

	"github.com/lahickey/quire"
)

const canBake = false

func bakeSite(*quire.Description) error {
	return errors.New("quire: built with quire_nobake, so this binary serves a bundle and cannot bake one")
}
