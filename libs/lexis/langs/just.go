package langs

import (
	"regexp"
	"strings"

	"github.com/lahickey/quire/libs/lexis"
)

var (
	// justPiece is what a line outside a recipe's body is made of: a
	// string, an attribute, a name, or anything else a character at a time.
	justPiece = regexp.MustCompile("^(?:(\"(?:[^\"\\\\\n]|\\\\.)*\"?|'[^'\n]*'?|`[^`\n]*`?)|(\\[[^\\]\n]*\\])|([A-Za-z_][A-Za-z0-9_-]*)|\\s+|.)")
	justWords = words(map[lexis.Kind]string{lexis.Keyword: "alias else export if import mod set"})
)

// justLexer is the lexer for a justfile, a line at a time: settings and
// recipes' first lines here, and a recipe's body, the indented lines under
// it, as the shell's with its {{ }} marked.
type justLexer struct {
	stepper
}

// step takes a line outside a body whole, and a body's line up to the end
// of its next {{ }}.
func (l *justLexer) step() {
	t, pos := l.text, l.pos
	end := lineEnd(t, pos)
	fresh := pos == 0 || t[pos-1] == '\n'
	line := strings.TrimRight(t[pos:end], "\r\n")
	switch {
	case fresh && line != "" && isSpace(line[0]) && !strings.HasPrefix(strings.TrimLeft(line, " \t"), "#"):
		l.add(lexis.Text, pos+len(line)-len(strings.TrimLeft(line, " \t@-")))
	case !fresh:
		if open := strings.Index(t[pos:end], "{{"); open == 0 {
			shut := end
			if i := strings.Index(t[pos:end], "}}"); i >= 0 {
				shut = pos + i + 2
			}
			l.add(lexis.Builtin, shut)
		} else {
			if open > 0 {
				end = pos + open
			}
			l.other("sh", end)
		}
	case strings.HasPrefix(strings.TrimLeft(line, " \t"), "#"):
		l.add(lexis.Comment, pos+len(line))
		l.add(lexis.Text, end)
	default:
		recipe := strings.Contains(strings.ReplaceAll(line, ":=", ""), ":")
		for first := true; l.pos < end; {
			m := justPiece.FindStringSubmatchIndex(t[l.pos:end])
			to := l.pos + m[1]
			switch word := t[l.pos:to]; {
			case m[2] >= 0:
				l.add(lexis.String, to)
			case m[4] >= 0 && first:
				l.add(lexis.Type, to)
			case m[4] >= 0:
				l.add(lexis.Punct, l.pos+1)
			case m[6] < 0 && isSpace(t[l.pos]):
				l.add(lexis.Text, to)
			case m[6] < 0:
				l.add(symbol(t[l.pos]), to)
			case justWords[word] != lexis.Text && (first || word == "if" || word == "else"):
				l.add(lexis.Keyword, to)
				first = false
			case first && recipe || to < end && t[to] == '(':
				l.add(lexis.Function, to)
				first = false
			default:
				l.add(lexis.Text, to)
				first = false
			}
		}
	}
}

func justLanguage(set *lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"just", "justfile"}, Files: []string{"justfile", "Justfile", ".justfile", "*.just"}, Lexer: func(text string) lexis.Lexer {
		l := &justLexer{set: set, text: text}
		l.more = l.step
		return l.next
	}}
}
