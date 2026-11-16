package langs

import (
	"github.com/lahickey/quire/libs/lexis"
	"go/scanner"
	"go/token"
	"strings"
)

var golangWords = words(map[lexis.Kind]string{
	lexis.Type:    "any bool byte comparable complex64 complex128 error float32 float64 int int8 int16 int32 int64 rune string uint uint8 uint16 uint32 uint64 uintptr",
	lexis.Builtin: "append cap clear close complex copy delete imag len make max min new panic print println real recover",
	lexis.Keyword: "true false nil iota",
})

// golangLexer is the lexer for Go, on Go's own scanner: a name is a function
// when a bracket follows it. The scanner drops carriage returns from what
// it gives, so text with any has no lexer.
func golangLexer(text string) lexis.Lexer {
	if strings.Contains(text, "\r") {
		return nil
	}
	file := token.NewFileSet().AddFile("", -1, len(text))
	var s scanner.Scanner
	s.Init(file, []byte(text), nil, scanner.ScanComments)
	type scanned struct {
		at  int
		tok token.Token
		lit string
	}
	scan := func() scanned {
		for {
			if pos, tok, lit := s.Scan(); tok != token.SEMICOLON || lit == ";" {
				return scanned{file.Offset(pos), tok, lit}
			}
		}
	}
	at, cur := 0, scan()
	return func() (lexis.Token, bool) {
		if at < cur.at {
			gap := text[at:cur.at]
			at = cur.at
			return lexis.Token{Text: gap, Kind: lexis.Text}, true
		}
		if cur.tok == token.EOF {
			return lexis.Token{}, false
		}
		was, n := cur, len(cur.lit)
		if n == 0 {
			n = len(cur.tok.String())
		}
		cur = scan()
		kind := lexis.Text
		switch {
		case was.tok == token.COMMENT:
			kind = lexis.Comment
		case was.tok.IsKeyword():
			kind = lexis.Keyword
		case was.tok == token.STRING || was.tok == token.CHAR:
			kind = lexis.String
		case was.tok == token.INT || was.tok == token.FLOAT || was.tok == token.IMAG:
			kind = lexis.Number
		case was.tok == token.PERIOD || was.tok == token.COLON:
			kind = lexis.Punct
		case was.tok.IsOperator():
			kind = symbol(text[at])
		case was.tok == token.IDENT:
			if named, ok := golangWords[was.lit]; ok {
				kind = named
			} else if cur.tok == token.LPAREN {
				kind = lexis.Function
			}
		}
		at += n
		return lexis.Token{Text: text[at-n : at], Kind: kind}, true
	}
}

func golangLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"go", "golang"}, Files: []string{"*.go"}, Lexer: golangLexer}
}
