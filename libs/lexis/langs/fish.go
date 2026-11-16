package langs

import (
	"strings"

	"github.com/lahickey/quire/libs/lexis"
)

var fishWords = words(map[lexis.Kind]string{
	lexis.Keyword: "and begin break case continue else end for function if in not or return switch while",
	lexis.Builtin: "abbr alias argparse bind builtin cd command complete contains count echo emit eval exec exit false fish_add_path functions math printf read set set_color source status string test true type",
})

// fishLexer is the lexer for fish: comments, strings, variables, its words,
// and the command each line, pipe or bracket starts with.
type fishLexer struct {
	text string
	pos  int
	// command says the next word is where a command's name goes.
	command bool
}

// fishInWord says c can be part of a word.
func fishInWord(c byte) bool { return !isSpace(c) && strings.IndexByte(`;|&()<>'"$#{}`, c) < 0 }

func (l *fishLexer) next() (lexis.Token, bool) {
	t, from := l.text, l.pos
	if from >= len(t) {
		return lexis.Token{}, false
	}
	at := func(i int) byte {
		if i < len(t) {
			return t[i]
		}
		return 0
	}
	c, end, kind := t[from], from+1, lexis.Text
	switch {
	case isSpace(c):
		for ; end < len(t) && isSpace(t[end]); end++ {
		}
		l.command = l.command || strings.IndexByte(t[from:end], '\n') >= 0
	case c == '#':
		kind, end = lexis.Comment, len(strings.TrimRight(t[:lineEnd(t, from)], "\r\n"))
	case c == '\'' || c == '"':
		kind, end = lexis.String, len(t)
		for i := from + 1; i < len(t); i++ {
			if t[i] == '\\' {
				i++
			} else if t[i] == c {
				end = i + 1
				break
			}
		}
	case c == '$' && at(end) == '{':
		kind, end = lexis.Builtin, len(t)
		if i := strings.IndexByte(t[from:], '}'); i >= 0 {
			end = from + i + 1
		}
	case c == '$' && (isName(at(end)) || strings.IndexByte("@*#?$!-", at(end)) >= 0 && at(end) != 0):
		kind = lexis.Builtin
		for end++; isName(t[end-1]) && isName(at(end)); end++ {
		}
	case fishInWord(c):
		for ; end < len(t) && (fishInWord(t[end]) || t[end] == '#'); end++ {
		}
		word := t[from:end]
		switch {
		case fishWords[word] != lexis.Text:
			kind = fishWords[word]
			l.command = kind == lexis.Keyword && !strings.Contains(" for in case select ", " "+word+" ")
		case strings.Trim(word, "0123456789") == "":
			kind = lexis.Number
		case l.command && !strings.Contains(word, "="):
			kind, l.command = lexis.Function, false
		}
	default:
		kind = symbol(c)
		l.command = l.command || strings.IndexByte(";|&(", c) >= 0
	}
	l.pos = end
	return lexis.Token{Text: t[from:end], Kind: kind}, true
}

func fishLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"fish"}, Files: []string{"*.fish"}, Lexer: func(text string) lexis.Lexer {
		return (&fishLexer{text: text, command: true}).next
	}}
}
