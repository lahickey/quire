//go:build quire_nobake

package quire

import "errors"

const canBake = false

func (e *Engine) bake(func(*Engine) error) error {
	return errors.New("quire: built with quire_nobake, so this binary serves a bundle and cannot bake one")
}
