//go:build release

package convert

import (
	"testing"

	"github.com/lahickey/quire/bake/baketest"
)

var fingerprints = map[string]string{
	"card":     "c39f0a24aabd90a6",
	"markdown": "01f90955263987dc",
	"mermaid":  "c124202d1629dadc",
	"picture":  "235a832a43d21f97",
}

func TestConverterFingerprints(t *testing.T) { baketest.Fingerprints(t, fingerprints) }
