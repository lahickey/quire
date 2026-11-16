package langs

import (
	"regexp"
	"strings"
)

var luaLongOpen = regexp.MustCompile(`\A\[(=*)\[`)

// luaLong matches Lua's long string after before, which is -- for a comment:
// [[ to ]] with as many = between the brackets at each end as it likes.
func luaLong(t string, pos int, before string) []int {
	if !strings.HasPrefix(t[pos:], before+"[") {
		return nil
	}
	m := luaLongOpen.FindStringSubmatchIndex(t[pos+len(before):])
	if m == nil {
		return nil
	}
	for i := range m {
		m[i] += len(before)
	}
	closing := "]" + t[pos+m[2]:pos+m[3]] + "]"
	i := strings.Index(t[pos+m[1]:], closing)
	if i < 0 {
		return nil
	}
	return []int{pos, pos + m[1] + i + len(closing)}
}
