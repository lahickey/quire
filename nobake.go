//go:build quire_nobake

package quire

import "errors"

const canBake = false

func bakeSite(*settings, Part) error {
	return errors.New("quire: built with quire_nobake, so this binary serves a bundle and cannot bake one")
}
