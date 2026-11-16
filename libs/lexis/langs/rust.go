package langs

import (
	"regexp"
	"strings"
)

var rustRawOpen = regexp.MustCompile(`\Ab?r(#*)"`)

// rustRaw matches Rust's raw string: r, as many # as it likes, a quote, and
// on to a quote followed by as many # again.
func rustRaw(t string, pos int) []int {
	if t[pos] != 'r' && t[pos] != 'b' {
		return nil
	}
	m := rustRawOpen.FindStringSubmatchIndex(t[pos:])
	if m == nil {
		return nil
	}
	hashes, quote := t[pos+m[2]:pos+m[3]], pos+m[1]-1
	i := strings.Index(t[quote+1:], `"`+hashes)
	if i < 0 {
		return nil
	}
	end := quote + i + 2 + len(hashes)
	return []int{pos, end, pos, pos + m[2], pos + m[2], pos + m[3], quote, end}
}
