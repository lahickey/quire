package langs

import (
	"regexp"
	"strings"
)

var bashHeredocOpen = regexp.MustCompile(`\A<<-?\s*'?\\?(\w+)`)

// bashHeredoc matches a shell's here-document: from << and the word after it to
// where that word next comes.
func bashHeredoc(t string, pos int) []int {
	if !strings.HasPrefix(t[pos:], "<<") {
		return nil
	}
	m := bashHeredocOpen.FindStringSubmatchIndex(t[pos:])
	if m == nil || pos+m[1] >= len(t) {
		return nil
	}
	word, from := t[pos+m[2]:pos+m[3]], pos+m[1]+1
	i := strings.Index(t[from:], word)
	if i < 0 {
		return nil
	}
	return []int{pos, from + i + len(word)}
}
