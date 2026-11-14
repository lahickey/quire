// Package examples holds complete sites, and in scaffold the files quire new adds to a copy of one.
package examples

import "embed"

//go:embed [0-9]-* all:scaffold
var Files embed.FS
