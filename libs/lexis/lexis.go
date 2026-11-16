// Package lexis splits source text into tokens and says what kind each is,
// for a caller to colour. It is made to be quick enough to run as someone
// types: a lexer gives its tokens one at a time, so a caller takes only as
// many as it has room to show.
//
// This package is the whole of what a caller uses: tokens, a lexer, and a
// set of languages to find a file's lexer in. It has no languages of its
// own. Those Lexis comes with are in langs, some written by hand and some a
// file of rules each, which the engine in rules runs; a caller adds them to
// its set, and any of its own.
package lexis

import (
	"path"
	"slices"
	"strings"
)

// Kind is what a token is.
type Kind uint8

const (
	Text Kind = iota
	Comment
	Keyword
	Type
	Builtin
	String
	Number
	Function
	Tag
	Preproc
	Heading
	Subheading
	Strong
	Emph
	Inserted
	Deleted
	Operator
	// Punct is punctuation: brackets and what separates things.
	Punct
)

// Token is a piece of the text and what kind of thing it is.
type Token struct {
	Text string
	Kind Kind
}

// A Lexer gives the next token of its text, and false once there are no
// more. The tokens' text, joined, is the text it was made over.
type Lexer func() (Token, bool)

// Language is a way to make a Lexer over a text, the names it goes by, such
// as "go" where a Markdown fence says what its code is, and the files it is
// for, as path.Match takes them, such as "*.go" or "justfile".
type Language struct {
	Names []string
	Files []string
	Lexer func(text string) Lexer
}

// Languages is a set of languages. The zero value is an empty set.
type Languages struct {
	all []Language
}

// Add puts languages in the set. One added later is found before those
// added earlier, so that a caller can put its own in place of one Lexis has.
func (l *Languages) Add(langs ...Language) {
	l.all = append(l.all, langs...)
}

// Named is a Lexer over text for the language called name, whatever its
// case, or nil when the set has none.
func (l *Languages) Named(name, text string) Lexer {
	for _, lang := range slices.Backward(l.all) {
		if slices.ContainsFunc(lang.Names, func(n string) bool { return strings.EqualFold(n, name) }) {
			return lang.Lexer(text)
		}
	}
	return nil
}

// File is a Lexer over text for the file called name, chosen by the name
// alone, or nil when the set has no language for it.
func (l *Languages) File(name, text string) Lexer {
	base := path.Base(strings.ReplaceAll(name, `\`, "/"))
	for _, lang := range slices.Backward(l.all) {
		for _, f := range lang.Files {
			if ok, _ := path.Match(f, base); ok {
				return lang.Lexer(text)
			}
		}
	}
	return nil
}
