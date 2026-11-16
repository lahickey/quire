package langs

import (
	"strings"
)

// yamlBlock matches a block scalar: | or >, then the lines indented as the
// first one after it is.
func yamlBlock(t string, pos int) []int {
	at := pos + 1
	if at < len(t) && (t[at] == '+' || t[at] == '-') {
		at++
	}
	if t[pos] != '|' && t[pos] != '>' || at >= len(t) || t[at] != '\n' {
		return nil
	}
	line := func(from int) int {
		if i := strings.IndexByte(t[from:], '\n'); i >= 0 {
			return from + i
		}
		return len(t)
	}
	indent := strings.Repeat(" ", len(t[at+1:])-len(strings.TrimLeft(t[at+1:], " ")))
	if indent == "" {
		return nil
	}
	end := line(at + 1)
	for next := end; next < len(t); {
		for next < len(t) && t[next] == '\n' {
			next++
		}
		if next == end || !strings.HasPrefix(t[next:], indent) {
			break
		}
		end = line(next)
		next = end
	}
	return []int{pos, end, pos, at, at, end}
}
