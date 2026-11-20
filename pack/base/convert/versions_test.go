//go:build release

package convert

import (
	"testing"

	"github.com/lahickey/quire/bake/baketest"
)

var fingerprints = map[string]string{
	"card":     "748a1d3740481de7",
	"markdown": "d290d2d3bcf46dd4",
	"mermaid":  "f9a51c7cbea42c13",
	"picture":  "5e040776d4436245",
}

func TestConverterFingerprints(t *testing.T) { baketest.Fingerprints(t, fingerprints) }
