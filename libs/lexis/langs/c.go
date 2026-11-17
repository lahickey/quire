package langs

import (
	"regexp"
	"strings"

	"github.com/lahickey/quire/libs/lexis"
)

const (
	cKeywords    = "alignas alignof auto break case const constexpr continue default do else enum extern false for goto if inline nullptr register restrict return sizeof static static_assert struct switch thread_local true typedef typeof union volatile while NULL _Alignas _Alignof _Atomic _Generic _Noreturn _Static_assert _Thread_local"
	cppKeywords  = " and and_eq bitand bitor catch class co_await co_return co_yield compl concept const_cast consteval constinit decltype delete dynamic_cast explicit export final friend import module mutable namespace new noexcept not not_eq operator or or_eq override private protected public reinterpret_cast requires static_cast template this throw try typeid typename using virtual xor xor_eq"
	objcKeywords = " self super nil Nil YES NO in nullable nonnull __weak __strong __block __bridge __unsafe_unretained __autoreleasing"
	objcAttrList = "assign atomic class copy getter nonatomic nonnull null_resettable nullable readonly readwrite retain setter strong unsafe_unretained weak"
	objcTypes    = " id instancetype BOOL SEL IMP Class NSInteger NSUInteger CGFloat"
	cTypes       = "bool char char8_t char16_t char32_t double float int long short signed unsigned void wchar_t size_t ssize_t ptrdiff_t intptr_t uintptr_t intmax_t uintmax_t int8_t int16_t int32_t int64_t uint8_t uint16_t uint32_t uint64_t FILE va_list _Bool _Complex"
)

// cEscape is an escape in a string, from its backslash.
var cEscape = regexp.MustCompile(`^(?s)\\(?:x[0-9a-fA-F]+|[0-7]{1,3}|u[0-9a-fA-F]{4}|U[0-9a-fA-F]{8}|.)`)

var (
	cWords      = words(map[lexis.Kind]string{lexis.Keyword: cKeywords, lexis.Type: cTypes})
	cppWords    = words(map[lexis.Kind]string{lexis.Keyword: cKeywords + cppKeywords, lexis.Type: cTypes})
	objcAttrs   = words(map[lexis.Kind]string{lexis.Keyword: objcAttrList})
	objcWords   = words(map[lexis.Kind]string{lexis.Keyword: cKeywords + objcKeywords, lexis.Type: cTypes + objcTypes})
	objcppWords = words(map[lexis.Kind]string{lexis.Keyword: cKeywords + cppKeywords + objcKeywords, lexis.Type: cTypes + objcTypes})
)

// cLexer is the lexer for C and the languages shaped like it, which differ
// in their words. A name is a function when a bracket follows it, and a
// name after an @ is a keyword, as Objective-C's are in any of them.
type cLexer struct {
	text  string
	pos   int
	words map[string]lexis.Kind
	// fresh says nothing but space has come on this line, where a # starts a
	// directive, and include that the directive is one naming a file.
	fresh, include bool
	// directive says the lexer is in a directive's line, which is all
	// Preproc but for its strings, numbers and comments, and joined says
	// the line goes on past its end, as one ending in a backslash does.
	directive, joined bool
	// str is where the string the lexer is inside ends, which it gives a
	// piece at a time so that its escapes are marked.
	str int
	// objc says the language has Objective-C's methods. method says the
	// lexer is in the declaration of one, depth how many brackets deep, and
	// named that its first name has come: that name and each one before a
	// colon are the method's. attrs is 1 after @property and 2 inside the
	// brackets that follow it, where its attributes are keywords.
	objc, method, named bool
	depth, attrs        int
}

// brackets follows the brackets in a symbol's text for what Objective-C
// has: a method's declaration ends at its body or a semicolon, and a
// property's attributes are in the brackets after it.
func (l *cLexer) brackets(text string) {
	for i := range len(text) {
		switch text[i] {
		case '(':
			l.depth, l.attrs = l.depth+1, l.attrs<<1&2
		case ')':
			l.depth, l.attrs = l.depth-1, 0
		case '{', ';':
			l.method, l.attrs = false, 0
		}
	}
}

// cRaw is where a raw string whose quote is at from ends: R"x( ... )x".
func cRaw(t string, from int) int {
	open := strings.IndexByte(t[from:], '(')
	if open < 0 || open > 17 {
		return quoted(t, from)
	}
	closing := ")" + t[from+1:from+open] + `"`
	if end := strings.Index(t[from+open:], closing); end >= 0 {
		return from + open + end + len(closing)
	}
	return len(t)
}

func (l *cLexer) next() (lexis.Token, bool) {
	t, from := l.text, l.pos
	if from >= len(t) {
		return lexis.Token{}, false
	}
	if from < l.str {
		kind, end := lexis.String, l.str
		if t[from] == '\\' {
			kind, end = lexis.Builtin, min(from+len(cEscape.FindString(t[from:])), l.str)
		} else if i := strings.IndexByte(t[from:l.str], '\\'); i >= 0 {
			end = from + i
		}
		l.pos = end
		return lexis.Token{Text: t[from:end], Kind: kind}, true
	}
	at := func(i int) byte {
		if i < len(t) {
			return t[i]
		}
		return 0
	}
	c, end, kind := t[from], from+1, lexis.Text
	fresh, escapes := false, false
	switch {
	case isSpace(c):
		fresh = l.fresh
		for ; end < len(t) && isSpace(t[end]); end++ {
		}
		if strings.IndexByte(t[from:end], '\n') >= 0 {
			fresh, l.include = true, false
			l.directive = l.directive && l.joined
		}
	case c == '/' && at(end) == '/':
		kind = lexis.Comment
		if end = strings.IndexByte(t[from:], '\n') + from; end < from {
			end = len(t)
		}
	case c == '/' && at(end) == '*':
		kind = lexis.Comment
		if end = strings.Index(t[from+2:], "*/") + from + 4; end < from+4 {
			end = len(t)
		}
	case c == '#' && l.fresh:
		kind = lexis.Preproc
		for ; at(end) == ' ' || at(end) == '\t'; end++ {
		}
		word := end
		for ; isName(at(end)); end++ {
		}
		l.include, l.directive = t[word:end] == "include" || t[word:end] == "import", true
	case c == '<' && l.include:
		kind = lexis.String
		for ; end < len(t) && t[end] != '\n' && t[end-1] != '>'; end++ {
		}
		l.include = false
	case c == '"' || c == '\'':
		kind, end, escapes = lexis.String, quoted(t, from), c == '"'
	case c == '@' && at(end) == '"':
		kind, end, escapes = lexis.String, quoted(t, end), true
	case c == '[' && at(end) == '[' && strings.Contains(t[from:lineEnd(t, from)], "]]"):
		kind, end = lexis.Type, from+strings.Index(t[from:], "]]")+2
	case isDigit(c) || (c == '.' || c == '@') && isDigit(at(end)):
		kind = lexis.Number
		hex := at(end) == 'x' || at(end) == 'X'
	number:
		for ; ; end++ {
			switch d, last := at(end), t[end-1]; {
			case isName(d) || d == '.':
			case d == '\'' && isName(at(end+1)):
			case (d == '+' || d == '-') && (last == 'p' || last == 'P' || !hex && (last == 'e' || last == 'E')):
			default:
				break number
			}
		}
	case c == '@' && at(end) >= 'a' && at(end) <= 'z':
		kind = lexis.Keyword
		for ; isName(at(end)); end++ {
		}
		if t[from:end] == "@property" {
			l.attrs = 1
		}
	case isName(c):
		for ; isName(at(end)); end++ {
		}
		word := t[from:end]
		after := end
		for ; at(after) == ' ' || at(after) == '\t'; after++ {
		}
		switch q := at(end); {
		case (q == '"' || q == '\'') && len(word) <= 3 && strings.Trim(word, "LuU8R") == "":
			kind = lexis.String
			if q == '"' && strings.HasSuffix(word, "R") {
				end = cRaw(t, end)
			} else {
				end, escapes = quoted(t, end), q == '"'
			}
		case l.words[word] != lexis.Text:
			kind = l.words[word]
		case l.attrs == 2 && objcAttrs[word] != lexis.Text:
			kind = objcAttrs[word]
		case at(after) == '(' || l.method && l.depth == 0 && (!l.named || at(after) == ':'):
			kind = lexis.Function
		}
		if l.method || l.attrs != 0 {
			l.named, l.attrs = l.named || l.depth == 0, l.attrs&2
		}
	default:
		punct := isPunct(c) || c == '.'
		kind = lexis.Operator
		if punct {
			kind = lexis.Punct
		}
		for ; end < len(t); end++ {
			d := t[end]
			if (isPunct(d) || d == '.') != punct || isSpace(d) || isName(d) || d == '@' || d == '"' || d == '\'' || d == '/' && (at(end+1) == '/' || at(end+1) == '*') || d == '<' && l.include {
				break
			}
		}
		if l.objc && l.fresh && (c == '-' || c == '+') && strings.HasPrefix(strings.TrimLeft(t[end:], " \t"), "(") {
			l.method, l.named, l.depth = true, false, 0
		}
		if l.method || l.attrs != 0 {
			l.brackets(t[from:end])
		}
	}
	if l.directive && !isSpace(c) && kind != lexis.String && kind != lexis.Number && kind != lexis.Comment {
		kind = lexis.Preproc
	}
	l.joined = t[end-1] == '\\'
	if escapes {
		if i := strings.IndexByte(t[from:end], '\\'); i >= 0 {
			l.str, end = end, from+i
		}
	}
	l.pos, l.fresh = end, fresh
	return lexis.Token{Text: t[from:end], Kind: kind}, true
}

func cLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"c"}, Files: []string{"*.c"}, Lexer: func(text string) lexis.Lexer {
		return (&cLexer{text: text, words: cWords, fresh: true}).next
	}}
}

func cppLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{
		Names: []string{"cpp", "c++", "cc", "h", "hpp"},
		Files: []string{"*.h", "*.hh", "*.hpp", "*.hxx", "*.h++", "*.cc", "*.cpp", "*.cxx", "*.c++", "*.C", "*.H", "*.ipp", "*.inl", "*.tpp"},
		Lexer: func(text string) lexis.Lexer {
			return (&cLexer{text: text, words: cppWords, fresh: true}).next
		},
	}
}

func objcLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"objc", "objective-c"}, Files: []string{"*.m"}, Lexer: func(text string) lexis.Lexer {
		return (&cLexer{text: text, words: objcWords, fresh: true, objc: true}).next
	}}
}

func objcppLanguage(*lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"objcpp", "objc++", "objective-c++"}, Files: []string{"*.mm"}, Lexer: func(text string) lexis.Lexer {
		return (&cLexer{text: text, words: objcppWords, fresh: true, objc: true}).next
	}}
}
