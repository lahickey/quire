package model

const (
	MasterCap  = 2560
	ShareWidth = 1200
	ladderSame = 1.15
)

var ladder = []int{64, 128, 232, 408, 680, 1096, 1704, MasterCap}

// Ladder is the widths a master is encoded at: the standard rungs below it, and the master itself.
func Ladder(masterW int) []int {
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

func snap(ladder []int, want int) int {
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
