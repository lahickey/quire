// Package langs is the languages Lexis comes with. Some are lexers written
// by hand, a file each. The rest are a file of rules each, from Chroma,
// which took them from Pygments: see THIRD_PARTY_NOTICES.md. A file of
// rules says in its config what the language is called and which files it
// is for, and where a rule needs more than a pattern it names a function
// in matchers, written in a Go file beside it.
//
// Every name a language's file declares starts with the file's name, and
// the last is its Language. Names with no such start are shared, and here.
package langs

import (
	"embed"
	"strings"

	"github.com/lahickey/quire/libs/lexis"
	"github.com/lahickey/quire/libs/lexis/rules"
)

//go:embed *.xml
var files embed.FS

var matchers = map[string]rules.Matcher{
	"bashHeredoc": bashHeredoc,
	"luaComment":  func(t string, pos int) []int { return luaLong(t, pos, "--") },
	"luaString":   func(t string, pos int) []int { return luaLong(t, pos, "") },
	"rustRaw":     rustRaw,
	"yamlBlock":   yamlBlock,
}

// stepper is what the lexers that take their text a step at a time share.
// more takes the next step: it queues tokens, or gives a piece of the text
// to another language's lexer in sub, whose tokens come after those queued.
type stepper struct {
	text  string
	pos   int
	set   *lexis.Languages
	queue []lexis.Token
	head  int
	sub   lexis.Lexer
	more  func()
}

// words is a table of names and their kinds, from a list of names for each.
func words(lists map[lexis.Kind]string) map[string]lexis.Kind {
	out := map[string]lexis.Kind{}
	for kind, list := range lists {
		for w := range strings.FieldsSeq(list) {
			out[w] = kind
		}
	}
	return out
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

func isName(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c >= 0x80
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// isPunct says c is punctuation, as a bracket or a separator is, where any
// other symbol is an operator.
func isPunct(c byte) bool { return strings.IndexByte("()[]{},;", c) >= 0 }

// symbol is the kind of a symbol.
func symbol(c byte) lexis.Kind {
	if isPunct(c) {
		return lexis.Punct
	}
	return lexis.Operator
}

// lineEnd is where the line at from ends, after its newline when it has one.
func lineEnd(t string, from int) int {
	if i := strings.IndexByte(t[from:], '\n'); i >= 0 {
		return from + i + 1
	}
	return len(t)
}

// quoted is where a string or character opened at from ends: after its
// closing quote, or at the end of its line when it has none.
func quoted(t string, from int) int {
	q := t[from]
	for i := from + 1; i < len(t); i++ {
		switch t[i] {
		case '\\':
			i++
		case '\n':
			return i
		case q:
			return i + 1
		}
	}
	return len(t)
}

func (s *stepper) next() (lexis.Token, bool) {
	for {
		if s.head < len(s.queue) {
			s.head++
			return s.queue[s.head-1], true
		}
		if s.sub != nil {
			if t, ok := s.sub(); ok {
				return t, true
			}
			s.sub = nil
		}
		if s.pos >= len(s.text) {
			return lexis.Token{}, false
		}
		s.queue, s.head = s.queue[:0], 0
		s.more()
	}
}

// add queues the text up to end as kind, and moves on to there.
func (s *stepper) add(kind lexis.Kind, end int) {
	if end > s.pos {
		s.queue = append(s.queue, lexis.Token{Text: s.text[s.pos:end], Kind: kind})
		s.pos = end
	}
}

// other gives the text up to end to the lexer for the language called name,
// after what is queued, and moves on to there.
func (s *stepper) other(name string, end int) {
	if s.sub = s.set.Named(name, s.text[s.pos:end]); s.sub == nil {
		s.add(lexis.Text, end)
	}
	s.pos = end
}

// Add puts every language here in set, each file of rules read the first
// time its language is used. templ, Markdown, justfiles and some rules hold
// other languages inside them, and look in set for the lexers of those.
func Add(set *lexis.Languages) {
	entries, err := files.ReadDir(".")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		b, err := files.ReadFile(e.Name())
		if err != nil {
			panic(err)
		}
		set.Add(rules.Lazy(set, b, matchers))
	}
	for _, language := range []func(*lexis.Languages) lexis.Language{
		golangLanguage, cLanguage, cppLanguage, objcLanguage, objcppLanguage, templLanguage, markdownLanguage, fishLanguage, justLanguage, jsonLanguage, javascriptLanguage, typescriptLanguage, tsxLanguage, cssLanguage, scssLanguage, htmlLanguage, xmlLanguage, pythonLanguage,
	} {
		set.Add(language(set))
	}
}
