// Package fonts is the default font set: Inter and JetBrains Mono, as a browser is sent them.
package fonts

import (
	"embed"
	"io/fs"
)

//go:embed web
var web embed.FS

// Web is the set's description and its faces, for a site to draw in.
func Web() fs.FS {
	sub, err := fs.Sub(web, "web")
	if err != nil {
		panic("quire: default fonts: " + err.Error())
	}
	return sub
}
