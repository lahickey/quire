package langs

import (
	"strings"

	"github.com/lahickey/quire/libs/lexis"
)

// What a cssLexer is in: the start of a statement, a selector, a
// declaration's property, or its value, which an at-rule's prelude is too.
const (
	cssStart = iota
	cssSelector
	cssProperty
	cssValue
)

// cssLexer is the lexer for CSS, and with scss for SCSS and Less: their
// variables, line comments and #{} are taken too. attr says a selector is
// inside square brackets, url that a url's bracket is due, and str where
// the bare text inside one ends.
type cssLexer struct {
	text            string
	pos, mode, str  int
	scss, attr, url bool
}

// cssDeclaration says the statement at from is a declaration and not a
// rule: its end comes before any brace opens. It looks only so far, where a
// statement still inside brackets is a declaration with a long url.
func cssDeclaration(t string, from int) bool {
	depth := 0
	for i := from; i < min(len(t), from+4096); i++ {
		switch c := t[i]; {
		case c == '"' || c == '\'':
			i = quoted(t, i) - 1
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == '#' && i+1 < len(t) && t[i+1] == '{':
			end := strings.IndexByte(t[i:], '}')
			if end < 0 {
				return true
			}
			i += end
		case depth > 0:
		case c == '{':
			return false
		case c == ';' || c == '}':
			return true
		}
	}
	return depth > 0 || from+4096 >= len(t)
}

func cssName(c byte) bool { return isName(c) || c == '-' }

func (l *cssLexer) next() (lexis.Token, bool) {
	t, from := l.text, l.pos
	if from >= len(t) {
		return lexis.Token{}, false
	}
	c, end, kind := t[from], from+1, lexis.Text
	next := byte(0)
	if end < len(t) {
		next = t[end]
	}
	to := func(i int) int {
		if i < 0 {
			return len(t)
		}
		return from + i
	}
	switch {
	case from < l.str:
		kind, end = lexis.String, l.str
	case isSpace(c):
		for ; end < len(t) && isSpace(t[end]); end++ {
		}
	case c == '/' && next == '*':
		kind, end = lexis.Comment, min(to(strings.Index(t[from:], "*/"))+2, len(t))
	case c == '/' && next == '/' && l.scss:
		kind, end = lexis.Comment, to(strings.IndexByte(t[from:], '\n'))
	default:
		if l.mode == cssStart {
			switch {
			case c == '@':
				l.mode = cssValue
			case cssDeclaration(t, from):
				l.mode = cssProperty
			default:
				l.mode = cssSelector
			}
		}
		kind, end = l.token(c, next, from)
	}
	l.pos = end
	return lexis.Token{Text: t[from:end], Kind: kind}, true
}

// token is the token at from that is not space or a comment: its kind and
// where it ends.
func (l *cssLexer) token(c, next byte, from int) (lexis.Kind, int) {
	t, end := l.text, from+1
	names := func() int {
		for ; end < len(t) && cssName(t[end]); end++ {
		}
		return end
	}
	switch {
	case c == '"' || c == '\'':
		return lexis.String, quoted(t, from)
	case c == '{' || c == '}' || c == ';':
		l.mode, l.attr = cssStart, false
		return lexis.Punct, end
	case c == '(' && l.url:
		l.url = false
		if next != '"' && next != '\'' {
			if l.str = strings.IndexByte(t[end:], ')') + end; l.str < end {
				l.str = len(t)
			}
		}
		return lexis.Punct, end
	case c == '[' || c == ']':
		l.attr = c == '[' && l.mode == cssSelector
		return lexis.Punct, end
	case isPunct(c):
		return lexis.Punct, end
	case c == '#' && next == '{' && l.scss:
		if end = strings.IndexByte(t[from:], '}') + from + 1; end <= from {
			end = len(t)
		}
		return lexis.Builtin, end
	case c == '$' && cssName(next) && l.scss:
		return lexis.Builtin, names()
	case c == '@' && cssName(next):
		return lexis.Keyword, names()
	case c == '!' && cssName(next):
		return lexis.Preproc, names()
	case isDigit(c) || (c == '.' || c == '-' || c == '+') && (isDigit(next) || next == '.' && end+1 < len(t) && isDigit(t[end+1])):
		for ; end < len(t) && (isName(t[end]) || t[end] == '.' || t[end] == '%'); end++ {
		}
		return lexis.Number, end
	case c == '#' && cssName(next):
		if l.mode == cssSelector {
			return lexis.Builtin, names()
		}
		return lexis.Number, names()
	case (c == '.' || c == '%') && cssName(next) && l.mode == cssSelector:
		return lexis.Type, names()
	case c == ':' && l.mode == cssSelector && (cssName(next) || next == ':'):
		for ; end < len(t) && t[end] == ':'; end++ {
		}
		return lexis.Keyword, names()
	case c == ':':
		if l.mode == cssProperty {
			l.mode = cssValue
		}
		return lexis.Punct, end
	case cssName(c) && (c != '-' || cssName(next)):
		word := t[from:names()]
		switch {
		case l.mode == cssSelector && l.attr:
			return lexis.Type, end
		case l.mode == cssSelector:
			return lexis.Tag, end
		case strings.HasPrefix(word, "--"):
			return lexis.Builtin, end
		case l.mode == cssProperty:
			return lexis.Keyword, end
		case end < len(t) && t[end] == '(':
			l.url = word == "url"
			return lexis.Function, end
		}
		return lexis.Text, end
	case strings.IndexByte("+-*/>~=^$|&%<!?.", c) >= 0:
		return lexis.Operator, end
	}
	return lexis.Text, end
}

func cssWith(scss bool) func(string) lexis.Lexer {
	return func(text string) lexis.Lexer { return (&cssLexer{text: text, scss: scss}).next }
}

func cssLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"css"}, Files: []string{"*.css"}, Lexer: cssWith(false)}
}

func scssLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"scss", "less"}, Files: []string{"*.scss", "*.less"}, Lexer: cssWith(true)}
}
