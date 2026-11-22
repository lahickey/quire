//go:build quire_nobake

package quire

import (
	"errors"

	"github.com/lahickey/quire/site"
)

const canBake = false

func bakeSite(*site.Description) error {
	return errors.New("quire: built with quire_nobake, so this binary serves a bundle and cannot bake one")
}
