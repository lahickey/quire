//go:build quire_embed

package main

import (
	"embed"

	"github.com/lahickey/quire"
)

//go:embed all:.quire/bundle
var bundle embed.FS

func init() { quire.Ship(bundle) }
