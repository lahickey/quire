//go:build release

package convert

import (
	"testing"

	"github.com/lahickey/quire/bake/baketest"
)

var fingerprints = map[string]string{
	"card":     "3a75fa5790f113a6",
	"markdown": "ed6080588da846dc",
	"mermaid":  "b7ab7089fcdc6402",
	"picture":  "d76217852cd98cee",
}

func TestConverterFingerprints(t *testing.T) { baketest.Fingerprints(t, fingerprints) }
