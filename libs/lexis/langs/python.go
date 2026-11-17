package langs

import (
	"strings"

	"github.com/lahickey/quire/libs/lexis"
)

var (
	pythonWords = words(map[lexis.Kind]string{
		lexis.Keyword: "False None True and as assert async await break class continue def del elif else except finally for from global if import in is lambda nonlocal not or pass raise return try while with yield",
		lexis.Builtin: "abs all any ascii bin bool breakpoint bytearray bytes callable chr classmethod compile complex delattr dict dir divmod enumerate eval exec filter float format frozenset getattr globals hasattr hash help hex id input int isinstance issubclass iter len list locals map max memoryview min next object oct open ord pow print property range repr reversed round set setattr slice sorted staticmethod str sum super tuple type vars zip __import__ self cls Ellipsis NotImplemented",
	})
	// pythonSoft is the keywords that are names anywhere but at the start
	// of a statement.
	pythonSoft = words(map[lexis.Kind]string{lexis.Keyword: "match case type"})
)

// pythonFrame is what a pythonLexer is in the middle of: an f-string with
// the quote that closes it, or with no quote the code in one's braces, as
// the whole file is, and how many brackets are open in it.
type pythonFrame struct {
	quote string
	depth int
}

// pythonLexer is the lexer for Python. fresh says nothing but space has
// come on this line, dot that the last token was a dot, and def what the
// next name is: a function's after def, a class's after class.
type pythonLexer struct {
	text       string
	pos        int
	fresh, dot bool
	def        lexis.Kind
	stack      []pythonFrame
}

// pythonQuote is the quote that opens the string at from, after its
// prefix's letters, with whether it is an f-string, or "" when what is at
// from is not a string.
func pythonQuote(t string, from int) (quote string, f bool) {
	at := from
	for ; at < len(t) && at < from+2 && strings.IndexByte("rRbBuUfF", t[at]) >= 0; at++ {
		f = f || t[at]|0x20 == 'f'
	}
	if at >= len(t) || t[at] != '"' && t[at] != '\'' {
		return "", false
	}
	if q := t[at : at+1]; strings.HasPrefix(t[at:], q+q+q) {
		return t[from : at+3], f
	}
	return t[from : at+1], f
}

// pythonString is where the string whose body starts at from ends, after
// the quote that closes it, or at the end of its line when it is on one
// line and has none.
func pythonString(t string, from int, quote string) int {
	for i := from; i < len(t); i++ {
		switch {
		case t[i] == '\\':
			i++
		case t[i] == '\n' && len(quote) == 1:
			return i
		case strings.HasPrefix(t[i:], quote):
			return i + len(quote)
		}
	}
	return len(t)
}

func (l *pythonLexer) at(i int) byte {
	if i < len(l.text) {
		return l.text[i]
	}
	return 0
}

// fstring is where the piece of an f-string at from ends: after the quote
// that closes it, or after a brace that opens code inside it.
func (l *pythonLexer) fstring(quote string, from int) int {
	t := l.text
	for i := from; i < len(t); i++ {
		switch c := t[i]; {
		case c == '\\':
			i++
		case c == '{' && l.at(i+1) == '{':
			i++
		case c == '{':
			l.stack = append(l.stack, pythonFrame{})
			return i + 1
		case c == '\n' && len(quote) == 1:
			l.stack = l.stack[:len(l.stack)-1]
			return i
		case strings.HasPrefix(t[i:], quote):
			l.stack = l.stack[:len(l.stack)-1]
			return i + len(quote)
		}
	}
	return len(t)
}

// symbol is the symbol at from in code.
func (l *pythonLexer) symbol(f *pythonFrame, from int) (int, lexis.Kind) {
	t, end := l.text, from+1
	inside := len(l.stack) > 1 && f.depth == 0
	switch c := t[from]; {
	case c == '}' && inside:
		l.stack = l.stack[:len(l.stack)-1]
		return end, lexis.String
	case inside && (c == ':' || c == '!' && l.at(end) != '='):
		for depth := 0; end < len(t) && (t[end] != '}' || depth > 0) && t[end] != '\n'; end++ {
			switch t[end] {
			case '{':
				depth++
			case '}':
				depth--
			}
		}
		return end, lexis.String
	case c == '(' || c == '[' || c == '{':
		f.depth++
	case c == ')' || c == ']' || c == '}':
		f.depth = max(f.depth-1, 0)
	case c == '.':
		l.dot = l.at(end) != '.'
	case c == ',' || c == ';':
	case c == ':' && l.at(end) != '=':
	case strings.IndexByte("+-*/%=<>&|^~!:@", c) >= 0:
		for ; end < len(t) && strings.IndexByte("+-*/%=<>&|^~!", t[end]) >= 0; end++ {
		}
		return end, lexis.Operator
	default:
		return end, lexis.Text
	}
	return end, lexis.Punct
}

// code is the token at from in code.
func (l *pythonLexer) code(f *pythonFrame, from int) (int, lexis.Kind) {
	t, end := l.text, from+1
	c, next := t[from], l.at(from+1)
	switch {
	case isSpace(c):
		for ; end < len(t) && isSpace(t[end]); end++ {
		}
		l.fresh = l.fresh || strings.IndexByte(t[from:end], '\n') >= 0
		return end, lexis.Text
	case c == '#':
		if end = strings.IndexByte(t[from:], '\n') + from; end < from {
			end = len(t)
		}
		if from == 0 && next == '!' {
			return end, lexis.Preproc
		}
		return end, lexis.Comment
	}
	fresh, dot, def := l.fresh, l.dot, l.def
	l.fresh, l.dot, l.def = false, false, lexis.Text
	quote, fstring := pythonQuote(t, from)
	switch {
	case fstring:
		l.stack = append(l.stack, pythonFrame{quote: strings.TrimLeft(quote, "rRbBuUfF")})
		return from + len(quote), lexis.String
	case quote != "":
		return pythonString(t, from+len(quote), strings.TrimLeft(quote, "rRbBuUfF")), lexis.String
	case isDigit(c) || c == '.' && isDigit(next):
		hex := next == 'x' || next == 'X'
		for ; end < len(t); end++ {
			d, last := t[end], t[end-1]
			sign := (d == '+' || d == '-') && !hex && (last == 'e' || last == 'E')
			if !isName(d) && (d != '.' || isName(l.at(end+1)) && !isDigit(l.at(end+1)) || l.at(end+1) == '.') && !sign {
				break
			}
		}
		return end, lexis.Number
	case c == '@' && fresh && isName(next):
		for ; end < len(t) && (isName(t[end]) || t[end] == '.'); end++ {
		}
		return end, lexis.Keyword
	case isName(c):
		for ; end < len(t) && isName(t[end]); end++ {
		}
		word, after := t[from:end], end
		for ; l.at(after) == ' ' || l.at(after) == '\t'; after++ {
		}
		kind := pythonWords[word]
		switch {
		case def != lexis.Text:
			return end, def
		case len(word) > 4 && strings.HasPrefix(word, "__") && strings.HasSuffix(word, "__"):
			kind = lexis.Builtin
		case dot:
			kind = lexis.Text
		case pythonSoft[word] != lexis.Text && fresh && l.at(end) != '(' && strings.IndexByte("=.,):", l.at(after)) < 0:
			kind = lexis.Keyword
		}
		switch {
		case word == "def" && kind == lexis.Keyword:
			l.def = lexis.Function
		case word == "class" && kind == lexis.Keyword:
			l.def = lexis.Type
		case kind == lexis.Text && l.at(after) == '(':
			kind = lexis.Function
		}
		return end, kind
	}
	return l.symbol(f, from)
}

func (l *pythonLexer) next() (lexis.Token, bool) {
	from := l.pos
	if from >= len(l.text) {
		return lexis.Token{}, false
	}
	f := &l.stack[len(l.stack)-1]
	end, kind := 0, lexis.String
	if f.quote != "" {
		end = l.fstring(f.quote, from)
	} else {
		end, kind = l.code(f, from)
	}
	l.pos = end
	return lexis.Token{Text: l.text[from:end], Kind: kind}, true
}

func pythonLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{
		Names: []string{"python", "py", "sage", "python3", "py3", "starlark"},
		Files: []string{
			"*.py", "*.pyi", "*.pyw", "*.jy", "*.sage", "*.sc", "SConstruct", "SConscript", "*.bzl", "BUCK", "BUILD", "BUILD.bazel",
			"WORKSPACE", "WORKSPACE.bzlmod", "WORKSPACE.bazel", "MODULE.bazel", "REPO.bazel", "*.star", "*.tac",
		},
		Lexer: func(text string) lexis.Lexer {
			return (&pythonLexer{text: text, fresh: true, stack: make([]pythonFrame, 1, 8)}).next
		},
	}
}
