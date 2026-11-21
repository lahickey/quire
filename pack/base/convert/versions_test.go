//go:build release

package convert

import (
	"testing"

	"github.com/lahickey/quire/bake/baketest"
)

var fingerprints = map[string]string{
	"card":     "08c6db2929b9549f",
	"markdown": "c01eed1df3c0fd1d",
	"mermaid":  "b20e6ad7e3b96564",
	"picture":  "4ecc223ebcac3fa6",
}

func TestConverterFingerprints(t *testing.T) { baketest.Fingerprints(t, fingerprints) }
