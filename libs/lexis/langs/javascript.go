package langs

import (
	"strings"

	"github.com/lahickey/quire/libs/lexis"
)

const (
	javascriptKeywords = "abstract as async await break case catch class const continue debugger default delete do else enum export extends false finally for from function get goto if implements import in instanceof interface let new null of package private protected public return set static super switch this throw true try typeof undefined using var void while with yield NaN Infinity"
	javascriptBuiltins = "Array Boolean Date Error Function Math Number Object Promise Proxy RegExp String decodeURI decodeURIComponent document encodeURI encodeURIComponent eval isFinite isNaN isSafeInteger parseFloat parseInt window"
	typescriptKeywords = " asserts constructor declare global infer is keyof module namespace readonly require type"
	typescriptTypes    = "any bigint boolean never number object string symbol unique unknown"
)

var (
	javascriptWords = words(map[lexis.Kind]string{lexis.Keyword: javascriptKeywords, lexis.Builtin: javascriptBuiltins})
	typescriptWords = words(map[lexis.Kind]string{lexis.Keyword: javascriptKeywords + typescriptKeywords, lexis.Builtin: javascriptBuiltins, lexis.Type: typescriptTypes})
	// javascriptLoose is the keywords that are names too: one is a name
	// when what follows could not follow the keyword.
	javascriptLoose = words(map[lexis.Kind]string{lexis.Keyword: "as from get set of type is module global declare namespace readonly require"})
	// javascriptTypers is the keywords a type's name follows.
	javascriptTypers = words(map[lexis.Kind]string{lexis.Keyword: "as class enum extends implements interface keyof new type"})
	// javascriptValues is the keywords that are values, so that a slash
	// after one divides.
	javascriptValues = words(map[lexis.Kind]string{lexis.Keyword: "this super true false null undefined NaN Infinity"})
)

// What a javascriptLexer is in the middle of: code, a template string, a
// JSX tag or its closing one, and what is between the two.
const (
	javascriptCode = iota
	javascriptTemplate
	javascriptTag
	javascriptClose
	javascriptChildren
)

// javascriptFrame is one of those, with how many braces are open in code
// and whether a tag's name has come.
type javascriptFrame struct {
	kind, depth int
	named       bool
}

// javascriptLexer is the lexer for JavaScript and TypeScript, which differ
// in their words. ts marks a capitalised name as a type where typed says
// one is due, after a colon or a word such as extends, and jsx takes a <
// where a value may start as a tag's. value says the last token ended a
// value, so a slash divides and a < compares, and dot that it was a dot.
type javascriptLexer struct {
	text              string
	pos               int
	words             map[string]lexis.Kind
	ts, jsx           bool
	value, dot, typed bool
	stack             []javascriptFrame
}

// javascriptRegex is where the regex whose slash is at from ends, or 0
// when it is not one.
func javascriptRegex(t string, from int) int {
	class := false
	for i := from + 1; i < len(t); i++ {
		switch c := t[i]; {
		case c == '\n':
			return 0
		case c == '\\':
			i++
		case c == '[':
			class = true
		case c == ']':
			class = false
		case c == '/' && !class:
			for i++; i < len(t) && isName(t[i]); i++ {
			}
			return i
		}
	}
	return 0
}

// javascriptGeneric says the name at from, after a <, is a type parameter
// and not a JSX tag's: a comma, an = or extends follows it, which nothing
// in a tag can.
func javascriptGeneric(t string, from int) bool {
	for ; from < len(t) && isName(t[from]); from++ {
	}
	rest := strings.TrimLeft(t[from:], " \t\n")
	after, extends := strings.CutPrefix(rest, "extends")
	return strings.HasPrefix(rest, ",") || strings.HasPrefix(rest, "=") || extends && (after == "" || !isName(after[0]))
}

func (l *javascriptLexer) at(i int) byte {
	if i < len(l.text) {
		return l.text[i]
	}
	return 0
}

func (l *javascriptLexer) push(kind int) {
	l.stack = append(l.stack, javascriptFrame{kind: kind})
}

func (l *javascriptLexer) pop() {
	l.stack = l.stack[:len(l.stack)-1]
}

// template is where the piece of a template string at from ends: after its
// closing quote, or after the ${ that opens code inside it.
func (l *javascriptLexer) template(from int) int {
	t := l.text
	for i := from; i < len(t); i++ {
		switch {
		case t[i] == '\\':
			i++
		case t[i] == '`':
			l.pop()
			l.value = true
			return i + 1
		case t[i] == '$' && l.at(i+1) == '{':
			l.push(javascriptCode)
			l.value = false
			return i + 2
		}
	}
	return len(t)
}

// children is the token at from between a JSX tag and its closing one.
func (l *javascriptLexer) children(f *javascriptFrame, from int) (int, lexis.Kind) {
	t := l.text
	switch c := t[from]; {
	case c == '{':
		l.push(javascriptCode)
		l.value = false
		return from + 1, lexis.Punct
	case c == '<' && l.at(from+1) == '/':
		f.kind, f.named = javascriptClose, false
		return from + 2, lexis.Punct
	case c == '<':
		l.push(javascriptTag)
		return from + 1, lexis.Punct
	}
	end := from + 1
	for ; end < len(t) && t[end] != '{' && t[end] != '<'; end++ {
	}
	return end, lexis.Text
}

// tag is the token at from inside a JSX tag, after its <.
func (l *javascriptLexer) tag(f *javascriptFrame, from int) (int, lexis.Kind) {
	t, end := l.text, from+1
	switch c := t[from]; {
	case isSpace(c):
		for ; end < len(t) && isSpace(t[end]); end++ {
		}
		return end, lexis.Text
	case c == '>' && f.kind == javascriptTag:
		f.kind = javascriptChildren
		return end, lexis.Punct
	case c == '>' || c == '/' && l.at(end) == '>':
		l.pop()
		l.value = true
		return from + strings.IndexByte(t[from:], '>') + 1, lexis.Punct
	case c == '"' || c == '\'':
		if i := strings.IndexByte(t[end:], c); i >= 0 {
			return end + i + 1, lexis.String
		}
		return len(t), lexis.String
	case c == '{':
		l.push(javascriptCode)
		l.value = false
		return end, lexis.Punct
	case c == '=':
		return end, lexis.Operator
	case isName(c) || c == '$':
		for ; end < len(t) && (isName(t[end]) || strings.IndexByte("$-.:", t[end]) >= 0); end++ {
		}
		kind := lexis.Type
		if !f.named {
			kind, f.named = lexis.Tag, true
		}
		return end, kind
	}
	return end, lexis.Text
}

// symbol is the symbol at from in code.
func (l *javascriptLexer) symbol(f *javascriptFrame, from int) (int, lexis.Kind) {
	t, end := l.text, from+1
	c, was := t[from], l.value
	l.value = c == ')' || c == ']'
	switch {
	case c == '{':
		f.depth++
	case c == '}' && f.depth == 0 && len(l.stack) > 1:
		l.pop()
		if l.stack[len(l.stack)-1].kind == javascriptTemplate {
			return end, lexis.String
		}
	case c == '}':
		f.depth, l.value = max(f.depth-1, 0), true
	case c == '.' && strings.HasPrefix(t[end:], ".."):
		end += 2
	case c == '.':
		l.dot = true
	case c == '=' && l.at(end) == '>':
		end++
	case isPunct(c):
	case strings.IndexByte("+-*/%=!<>&|^~?:", c) >= 0:
		for ; end < len(t) && strings.IndexByte("+-*%=!>&|^~?:", t[end]) >= 0; end++ {
		}
		op := t[from:end]
		l.value, l.typed = was && (op == "++" || op == "--"), op == ":" || op == "?:"
		return end, lexis.Operator
	default:
		return end, lexis.Text
	}
	return end, lexis.Punct
}

// code is the token at from in code.
func (l *javascriptLexer) code(f *javascriptFrame, from int) (int, lexis.Kind) {
	t, end := l.text, from+1
	c, next := t[from], l.at(from+1)
	switch {
	case isSpace(c):
		for ; end < len(t) && isSpace(t[end]); end++ {
		}
		return end, lexis.Text
	case c == '/' && next == '/', c == '#' && next == '!' && from == 0:
		if end = strings.IndexByte(t[from:], '\n') + from; end < from {
			end = len(t)
		}
		if c == '#' {
			return end, lexis.Preproc
		}
		return end, lexis.Comment
	case c == '/' && next == '*':
		if end = strings.Index(t[from+2:], "*/") + from + 4; end < from+4 {
			end = len(t)
		}
		return end, lexis.Comment
	}
	value, dot, typed := l.value, l.dot, l.typed
	l.value, l.dot, l.typed = true, false, false
	switch {
	case c == '"' || c == '\'':
		return quoted(t, from), lexis.String
	case c == '`':
		l.push(javascriptTemplate)
		return end, lexis.String
	case isDigit(c) || c == '.' && isDigit(next):
		hex := next == 'x' || next == 'X'
		for ; end < len(t); end++ {
			d, last := t[end], t[end-1]
			sign := (d == '+' || d == '-') && !hex && (last == 'e' || last == 'E')
			if !isName(d) && (d != '.' || !isDigit(l.at(end+1))) && !sign {
				break
			}
		}
		return end, lexis.Number
	case isName(c) || c == '$' || (c == '#' || c == '@') && isName(next):
		for ; end < len(t) && (isName(t[end]) || t[end] == '$'); end++ {
		}
		if c == '@' {
			return end, lexis.Keyword
		}
		word, after := t[from:end], end
		for ; l.at(after) == ' ' || l.at(after) == '\t'; after++ {
		}
		kind := l.words[word]
		if dot || javascriptLoose[word] != lexis.Text && strings.IndexByte("(=:,);.", l.at(after)) >= 0 {
			kind = lexis.Text
		}
		switch {
		case kind != lexis.Text:
			l.value = kind != lexis.Keyword || javascriptValues[word] != lexis.Text
			l.typed = javascriptTypers[word] != lexis.Text
		case l.at(after) == '(':
			kind = lexis.Function
		case l.ts && typed && c >= 'A' && c <= 'Z':
			kind = lexis.Type
		}
		return end, kind
	case c == '/' && !value:
		if end := javascriptRegex(t, from); end > 0 {
			return end, lexis.String
		}
	case c == '<' && l.jsx && !value && (next == '>' || next == '_' || next|0x20 >= 'a' && next|0x20 <= 'z') && (!l.ts || !javascriptGeneric(t, end)):
		l.push(javascriptTag)
		return end, lexis.Punct
	}
	l.value = value
	return l.symbol(f, from)
}

func (l *javascriptLexer) next() (lexis.Token, bool) {
	from := l.pos
	if from >= len(l.text) {
		return lexis.Token{}, false
	}
	f := &l.stack[len(l.stack)-1]
	end, kind := 0, lexis.String
	switch f.kind {
	case javascriptCode:
		end, kind = l.code(f, from)
	case javascriptTemplate:
		end = l.template(from)
	case javascriptChildren:
		end, kind = l.children(f, from)
	default:
		end, kind = l.tag(f, from)
	}
	l.pos = end
	return lexis.Token{Text: l.text[from:end], Kind: kind}, true
}

func javascriptWith(words map[string]lexis.Kind, ts, jsx bool) func(string) lexis.Lexer {
	return func(text string) lexis.Lexer {
		return (&javascriptLexer{text: text, words: words, ts: ts, jsx: jsx, stack: make([]javascriptFrame, 1, 8)}).next
	}
}

func javascriptLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{
		Names: []string{"js", "javascript", "jsx"},
		Files: []string{"*.js", "*.jsm", "*.mjs", "*.cjs", "*.jsx"},
		Lexer: javascriptWith(javascriptWords, false, true),
	}
}

func typescriptLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{
		Names: []string{"ts", "typescript"},
		Files: []string{"*.ts", "*.mts", "*.cts"},
		Lexer: javascriptWith(typescriptWords, true, false),
	}
}

func tsxLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"tsx"}, Files: []string{"*.tsx"}, Lexer: javascriptWith(typescriptWords, true, true)}
}
