package langs

import (
	"strings"

	"github.com/lahickey/quire/libs/lexis"
)

// jsonLexer is the lexer for JSON, and for what JSON5 and JSONC add to it:
// comments, single quotes and keys with no quotes. A string or a name is a
// key when a colon follows it.
type jsonLexer struct {
	text string
	pos  int
}

func jsonKey(t string, at int) bool {
	for ; at < len(t) && isSpace(t[at]); at++ {
	}
	return at < len(t) && t[at] == ':'
}

func (l *jsonLexer) next() (lexis.Token, bool) {
	t, from := l.text, l.pos
	if from >= len(t) {
		return lexis.Token{}, false
	}
	c, end, kind := t[from], from+1, lexis.Text
	after := byte(0)
	if end < len(t) {
		after = t[end]
	}
	switch {
	case isSpace(c):
		for ; end < len(t) && isSpace(t[end]); end++ {
		}
	case c == '"' || c == '\'':
		kind, end = lexis.String, quoted(t, from)
		if jsonKey(t, end) {
			kind = lexis.Tag
		}
	case isPunct(c) || c == ':':
		kind = lexis.Punct
	case c == '/' && after == '/':
		kind = lexis.Comment
		if end = strings.IndexByte(t[from:], '\n') + from; end < from {
			end = len(t)
		}
	case c == '/' && after == '*':
		kind = lexis.Comment
		if end = strings.Index(t[from+2:], "*/") + from + 4; end < from+4 {
			end = len(t)
		}
	case isDigit(c) || (c == '-' || c == '+' || c == '.') && isDigit(after):
		kind = lexis.Number
		for ; end < len(t) && (isName(t[end]) || t[end] == '.' || (t[end] == '+' || t[end] == '-') && (t[end-1] == 'e' || t[end-1] == 'E')); end++ {
		}
	case isName(c):
		for ; end < len(t) && isName(t[end]); end++ {
		}
		switch word := t[from:end]; {
		case word == "true" || word == "false" || word == "null":
			kind = lexis.Keyword
		case jsonKey(t, end):
			kind = lexis.Tag
		}
	}
	l.pos = end
	return lexis.Token{Text: t[from:end], Kind: kind}, true
}

func jsonLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{
		Names: []string{"json", "jsonc", "jsonl", "json5"},
		Files: []string{"*.json", "*.jsonl", "*.jsonc", "*.json5", "*.avsc", ".luaurc"},
		Lexer: func(text string) lexis.Lexer { return (&jsonLexer{text: text}).next },
	}
}
