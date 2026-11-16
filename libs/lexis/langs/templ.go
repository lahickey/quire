package langs

import (
	"github.com/lahickey/quire/libs/lexis"
	"regexp"
	"strings"
)

var (
	// templBlock is the line that opens a templ, css or script block.
	templBlock = regexp.MustCompile(`(?m)^(templ|css|script)(\s[^\n]*\{)[ \t]*$`)
	// templGo is a line inside a block that is a Go statement.
	templGo = regexp.MustCompile(`^(?:\} *)?(?:if|for|switch|case|default|else)\b`)
)

const (
	templTop = iota
	templHead
	templMarkup
	templTag
	templScript
)

// templLexer is the lexer for templ: Go, with blocks of markup in which
// braces hold Go again. What is Go goes to the Go lexer, a script's text to
// JavaScript's and a style's to CSS's. A block ends at a closing brace at
// the start of a line, as templ's own formatter leaves it.
type templLexer struct {
	stepper
	mode int
	// tag is the element whose opening tag the lexer is in.
	tag string
}

// templClosing is where the bracket opened at from closes, past strings.
func templClosing(t string, from int) int {
	open := t[from]
	shut := map[byte]byte{'{': '}', '(': ')'}[open]
	depth := 0
	for i := from; i < len(t); i++ {
		switch c := t[i]; c {
		case '"', '\'':
			i = quoted(t, i) - 1
		case '`':
			if end := strings.IndexByte(t[i+1:], '`'); end >= 0 {
				i += end + 1
			}
		case open:
			depth++
		case shut:
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return len(t)
}

func (l *templLexer) step() {
	t, pos := l.text, l.pos
	end := lineEnd(t, pos)
	line := strings.TrimRight(t[pos:end], "\r\n")
	fresh := pos == 0 || t[pos-1] == '\n'
	switch l.mode {
	case templTop:
		m := templBlock.FindStringIndex(t[pos:])
		if m == nil {
			l.other("go", len(t))
			return
		}
		l.other("go", pos+m[0])
		l.mode = templHead
	case templHead:
		word, _, _ := strings.Cut(line, " ")
		l.mode = templMarkup
		if word == "script" {
			l.mode = templScript
		}
		l.add(lexis.Keyword, pos+len(word))
		l.other("go", end)
	case templScript:
		body, at := t[pos:], len(t)
		if i := strings.Index("\n"+body, "\n}"); i >= 0 {
			at = pos + i
		}
		l.other("js", at)
		l.mode = templMarkup
	case templMarkup:
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		switch c := t[pos]; {
		case fresh && line == "}":
			l.add(lexis.Punct, pos+1)
			l.add(lexis.Text, end)
			l.mode = templTop
		case fresh && templGo.MatchString(line[indent:]):
			l.add(lexis.Text, pos+indent)
			l.other("go", pos+len(line))
		case strings.HasPrefix(t[pos:], "<!--"):
			at := len(t)
			if i := strings.Index(t[pos:], "-->"); i >= 0 {
				at = pos + i + 3
			}
			l.add(lexis.Comment, at)
		case c == '<' && pos+1 < len(t) && (isName(t[pos+1]) || t[pos+1] == '/'):
			l.add(lexis.Punct, pos+1+strings.Count(t[pos+1:pos+2], "/"))
			at := l.pos
			for ; at < len(t) && (isName(t[at]) || t[at] == '-' || t[at] == ':' || t[at] == '.'); at++ {
			}
			l.tag = t[l.pos:at]
			if t[pos+1] == '/' {
				l.tag = ""
			}
			l.add(lexis.Tag, at)
			l.mode = templTag
		case c == '{' && strings.TrimSpace(t[pos+1:end]) == "":
			l.add(lexis.Punct, pos+1)
		case c == '{':
			l.add(lexis.Punct, pos+1)
			l.other("go", templClosing(t, pos))
		case c == '@':
			at := pos + 1
			for ; at < len(t) && (isName(t[at]) || t[at] == '.'); at++ {
			}
			l.add(lexis.Function, at)
			if at < len(t) && t[at] == '(' {
				l.other("go", min(templClosing(t, at)+1, len(t)))
			}
		case c == '}':
			l.add(lexis.Punct, pos+1)
		default:
			at := pos + 1
			for ; at < end && strings.IndexByte("<{}@", t[at]) < 0; at++ {
			}
			l.add(lexis.Text, at)
		}
	case templTag:
		switch c := t[pos]; {
		case isSpace(c):
			at := pos
			for ; at < len(t) && isSpace(t[at]); at++ {
			}
			l.add(lexis.Text, at)
		case c == '>':
			l.add(lexis.Punct, pos+1)
			l.mode = templMarkup
			if l.tag == "script" || l.tag == "style" {
				at := len(t)
				if i := strings.Index(t[l.pos:], "</"+l.tag); i >= 0 {
					at = l.pos + i
				}
				l.other(map[string]string{"script": "js", "style": "css"}[l.tag], at)
			}
		case c == '/' && strings.HasPrefix(t[pos:], "/>"):
			l.add(lexis.Punct, pos+2)
			l.mode = templMarkup
		case c == '"':
			at := len(t)
			if i := strings.IndexByte(t[pos+1:], '"'); i >= 0 {
				at = pos + i + 2
			}
			l.add(lexis.String, at)
		case c == '{':
			l.add(lexis.Punct, pos+1)
			l.other("go", templClosing(t, pos))
		case isName(c) || c == '-' || c == ':' || c == '@':
			at := pos
			for ; at < len(t) && !isSpace(t[at]) && strings.IndexByte(`=>/{}"`, t[at]) < 0; at++ {
			}
			l.add(lexis.Type, at)
		default:
			l.add(symbol(c), pos+1)
		}
	}
}

func templLanguage(set *lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"templ"}, Files: []string{"*.templ"}, Lexer: func(text string) lexis.Lexer {
		l := &templLexer{set: set, text: text}
		l.more = l.step
		return l.next
	}}
}
