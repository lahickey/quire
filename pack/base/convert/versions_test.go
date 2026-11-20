//go:build release

package convert

import (
	"testing"

	"github.com/lahickey/quire/bake/baketest"
)

var fingerprints = map[string]string{
	"card":     "d2e610632bc225eb",
	"markdown": "95e1f65c45a9d4f6",
	"mermaid":  "ec391902b229b6c6",
	"picture":  "f614115b8149a323",
}

func TestConverterFingerprints(t *testing.T) { baketest.Fingerprints(t, fingerprints) }
