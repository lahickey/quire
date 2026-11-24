//go:build release

package convert

import (
	"testing"

	"github.com/lahickey/quire/bake/baketest"
)

var fingerprints = map[string]string{
	"card":     "d87fa0eb79cf3f23",
	"markdown": "73287a7887174f84",
	"mermaid":  "9ff625b35933cc8f",
	"picture":  "1d7c2d0eda324b09",
}

func TestConverterFingerprints(t *testing.T) { baketest.Fingerprints(t, fingerprints) }
