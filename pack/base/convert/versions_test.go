//go:build release

package convert

import (
	"testing"

	"github.com/lahickey/quire/bake/baketest"
)

var fingerprints = map[string]string{
	"card":     "1275529b6028e163",
	"markdown": "80ed85842f1a85c5",
	"mermaid":  "dba2043e9f912f72",
	"picture":  "99d8190da00af97c",
}

func TestConverterFingerprints(t *testing.T) { baketest.Fingerprints(t, fingerprints) }
