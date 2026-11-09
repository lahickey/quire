//go:build quire_embed

// Only in a shipping build: go run . bake, then go build -tags quire_nobake,quire_embed.
// It compiles the baked bundle into the binary, which then serves from any
// directory and links no image or diagram tooling. The build tag keeps everyday
// builds compiling before the first bake has made .quire/bundle.
package main

import (
	"embed"

	"github.com/lahickey/quire"
)

//go:embed all:.quire/bundle
var bundle embed.FS

func init() { quire.Ship(bundle) }
