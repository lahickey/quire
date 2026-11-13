package journal

import "fmt"

const (
	MasterCap  = 2560
	ShareWidth = 1200
	ladderSame = 1.15
)

var ladder = []int{64, 128, 232, 408, 680, 1096, 1704, MasterCap}

func PictureKey(version int, srcDigest string) string {
	return Hash(fmt.Appendf(nil, "v=%d\nsrc=%s\n", version, srcDigest))
}

func LadderFor(masterW int) []int {
	if masterW < 1 {
		return nil
	}
	out := make([]int, 0, len(ladder)+1)
	for _, w := range ladder {
		if w >= masterW {
			break
		}
		out = append(out, w)
	}
	if n := len(out); n > 0 && float64(masterW) < ladderSame*float64(out[n-1]) {
		out = out[:n-1]
	}
	return append(out, masterW)
}

func Snap(ladder []int, want int) int {
	for _, w := range ladder {
		if w >= want {
			return w
		}
	}
	if len(ladder) == 0 {
		return want
	}
	return ladder[len(ladder)-1]
}
