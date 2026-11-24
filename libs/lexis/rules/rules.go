// Package rules is the engine for a language that is a file of rules, as
// Chroma's are. A rule is a pattern tried where the lexer stands, in a
// state; the first to match gives its text a kind and may push or pop
// states. Patterns are Go's regexp, which has no lookaround, so a rule that
// needs to see what follows without taking it says so in ahead, one that
// minds the character before it, when there is one, says which it may be in
// behind, and one that needs more than a pattern names a Matcher in func.
package rules

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"regexp/syntax"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/lahickey/quire/libs/lexis"
)

// A Matcher gives the bounds of its match at pos and of its groups, as
// regexp does, or nil.
type Matcher = func(text string, pos int) []int

type xmlConfig struct {
	Aliases         []string `xml:"alias"`
	Filenames       []string `xml:"filename"`
	CaseInsensitive bool     `xml:"case_insensitive"`
	DotAll          bool     `xml:"dot_all"`
	NotMultiline    bool     `xml:"not_multiline"`
	EnsureNL        bool     `xml:"ensure_nl"`
}

type xmlRules struct {
	Config xmlConfig `xml:"config"`
	States []struct {
		Name  string    `xml:"name,attr"`
		Rules []xmlRule `xml:"rule"`
	} `xml:"rules>state"`
}

type xmlRule struct {
	Pattern string `xml:"pattern,attr"`
	Ahead   string `xml:"ahead,attr"`
	Behind  string `xml:"behind,attr"`
	Func    string `xml:"func,attr"`
	Include *struct {
		State string `xml:"state,attr"`
	} `xml:"include"`
	Whole *struct {
		Name string `xml:"type,attr"`
	} `xml:"token"`
	Using *struct {
		Lexer string `xml:"lexer,attr"`
	} `xml:"using"`
	Groups *struct {
		Items []xmlItem `xml:",any"`
	} `xml:"bygroups"`
	Actions []xmlItem `xml:",any"`
}

// xmlItem is a group's token, using or usingself, or a rule's push, pop or
// combined, which pushes a state made of the rules of each it names.
type xmlItem struct {
	XMLName xml.Name
	Name    string `xml:"type,attr"`
	Lexer   string `xml:"lexer,attr"`
	States  string `xml:"states,attr"`
	State   string `xml:"state,attr"`
	Depth   int    `xml:"depth,attr"`
}

// emit is what a rule's whole match, or one of its groups, becomes: a token
// of kind, or with self the tokens of lexing it again from that state, or
// with using those of another language's lexer.
type emit struct {
	kind  lexis.Kind
	self  string
	using string
}

type rule struct {
	// at0 matches at the start of the text. at1 matches after one character
	// before it, for a pattern that looks at what is behind it, and is nil
	// for one that does not.
	at0, at1 *regexp.Regexp
	fn       Matcher
	// first is the bytes a match can start with, nil for any, and inLine
	// says it cannot take a newline, so the rest of the line is all it needs.
	first  *[256]bool
	inLine bool
	// ahead says the pattern's last group is what must follow, not taken.
	ahead  bool
	groups bool
	emits  []emit
	// then is the states to push after popping pop of them, "" for the
	// state the lexer is in.
	pop  int
	push []string
}

// grammar is a language's rules by state, and the set its rules look in
// for another language's lexer, nil when it was loaded without one. newline
// says its rules count on the text ending in one.
type grammar struct {
	states  map[string][]*rule
	set     *lexis.Languages
	newline bool
}

// other is the tokens of text in the language a rule names, as plain text
// when the grammar has no lexer for it.
func (g grammar) other(name, text string) lexis.Lexer {
	if g.set != nil {
		if next := g.set.Named(name, text); next != nil {
			return next
		}
	}
	done := false
	return func() (lexis.Token, bool) {
		done = !done
		return lexis.Token{Text: text}, done
	}
}

// twoStates finds a push with two state attributes, as Chroma writes one
// that pushes two: XML allows an attribute once, and only one is read.
var twoStates = regexp.MustCompile(`state="[^"]*"\s+state=`)

// KindOf is the kind Lexis gives one of Chroma's token types, by its name.
func KindOf(typ string) lexis.Kind {
	switch {
	case strings.HasPrefix(typ, "CommentPreproc"):
		return lexis.Preproc
	case strings.HasPrefix(typ, "Comment"):
		return lexis.Comment
	case typ == "KeywordType", typ == "NameClass", typ == "NameAttribute":
		return lexis.Type
	case strings.HasPrefix(typ, "Keyword"), typ == "OperatorWord", typ == "NameDecorator":
		return lexis.Keyword
	case typ == "Operator":
		return lexis.Operator
	case typ == "Punctuation":
		return lexis.Punct
	case typ == "LiteralStringEscape", typ == "NameBuiltin", typ == "NameBuiltinPseudo", strings.HasPrefix(typ, "NameVariable"):
		return lexis.Builtin
	case strings.HasPrefix(typ, "LiteralString"):
		return lexis.String
	case strings.HasPrefix(typ, "LiteralNumber"):
		return lexis.Number
	case typ == "NameFunction", typ == "NameFunctionMagic":
		return lexis.Function
	case typ == "NameTag":
		return lexis.Tag
	case typ == "GenericHeading":
		return lexis.Heading
	case typ == "GenericSubheading":
		return lexis.Subheading
	case typ == "GenericStrong":
		return lexis.Strong
	case typ == "GenericEmph":
		return lexis.Emph
	case typ == "GenericInserted":
		return lexis.Inserted
	case typ == "GenericDeleted":
		return lexis.Deleted
	}
	return lexis.Text
}

// wordList is a pattern that is a list of words to match whole: an optional
// \b, a group of alternatives, and \b.
var wordList = regexp.MustCompile(`^(\\b)?\((\?:)?((?:[^()\[\]\\]|\\.)+)\)\\b$`)

func isWord(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// words is a Matcher for a pattern that is a list of plain words, which is
// how a language's keywords are given and is slow as a regexp when the list
// is long, or nil when the pattern is anything else. It matches as the
// regexp would: the first word in the list that is there and ends at a
// word boundary. behind is what the character before may be, "" for any.
func words(pattern, behind string, fold bool) (Matcher, error) {
	m := wordList.FindStringSubmatch(pattern)
	if m == nil {
		return nil, nil
	}
	var byFirst [256][]string
	for alt := range strings.SplitSeq(m[3], "|") {
		re, err := syntax.Parse(alt, syntax.Perl)
		if err != nil || re.Op != syntax.OpLiteral || re.Flags&syntax.FoldCase != 0 || strings.HasSuffix(alt, `\`) {
			return nil, nil
		}
		word := string(re.Rune)
		c := word[0]
		byFirst[c] = append(byFirst[c], word)
		if fold && c|0x20 >= 'a' && c|0x20 <= 'z' {
			byFirst[c^0x20] = append(byFirst[c^0x20], word)
		}
	}
	var before *regexp.Regexp
	if behind != "" {
		var err error
		if before, err = regexp.Compile(`(?:` + behind + `)\z`); err != nil {
			return nil, err
		}
	}
	bounded, group := m[1] != "", m[2] == ""
	boundary := func(t string, at int) bool {
		return (at > 0 && isWord(t[at-1])) != (at < len(t) && isWord(t[at]))
	}
	return func(t string, pos int) []int {
		list := byFirst[t[pos]]
		if list == nil || bounded && !boundary(t, pos) {
			return nil
		}
		if before != nil && pos > 0 {
			_, n := utf8.DecodeLastRuneInString(t[:pos])
			if !before.MatchString(t[pos-n : pos]) {
				return nil
			}
		}
		for _, word := range list {
			end := pos + len(word)
			if end > len(t) || !boundary(t, end) {
				continue
			}
			if t[pos:end] == word || fold && strings.EqualFold(t[pos:end], word) {
				if group {
					return []int{pos, end, pos, end}
				}
				return []int{pos, end}
			}
		}
		return nil
	}, nil
}

// analyse is the bytes a match of pattern can start with, nil when it can be
// empty or start with any, and whether it cannot take a newline.
func analyse(pattern string) (first *[256]bool, inLine bool) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil, false
	}
	prog, err := syntax.Compile(re.Simplify())
	if err != nil {
		return nil, false
	}
	inLine = true
	for _, in := range prog.Inst {
		if in.Op == syntax.InstRuneAny || (in.Op == syntax.InstRune || in.Op == syntax.InstRune1) && in.MatchRune('\n') {
			inLine = false
		}
	}
	var set [256]bool
	seen := map[uint32]bool{}
	open := false
	var walk func(pc uint32)
	walk = func(pc uint32) {
		if seen[pc] || open {
			return
		}
		seen[pc] = true
		switch in := prog.Inst[pc]; in.Op {
		case syntax.InstAlt, syntax.InstAltMatch:
			walk(in.Out)
			walk(in.Arg)
		case syntax.InstCapture, syntax.InstNop, syntax.InstEmptyWidth:
			walk(in.Out)
		case syntax.InstRune, syntax.InstRune1:
			fold := syntax.Flags(in.Arg)&syntax.FoldCase != 0
			add := func(lo, hi rune) {
				for c := lo; c <= min(hi, 0xff); c++ {
					set[c] = true
					if fold && (c|0x20 >= 'a' && c|0x20 <= 'z') {
						set[c^0x20] = true
					}
				}
				if hi >= utf8.RuneSelf {
					for c := 0x80; c <= 0xff; c++ {
						set[c] = true
					}
				}
			}
			if len(in.Rune) == 1 {
				add(in.Rune[0], in.Rune[0])
			}
			for i := 0; i+1 < len(in.Rune); i += 2 {
				add(in.Rune[i], in.Rune[i+1])
			}
		default:
			open = true
		}
	}
	walk(uint32(prog.Start))
	if open {
		return nil, inLine
	}
	return &set, inLine
}

// looksBehind says a match of pattern can depend on the character before
// it: a word boundary or the start of a line or of the text, reached before
// anything is taken.
func looksBehind(pattern string) bool {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return true
	}
	prog, err := syntax.Compile(re.Simplify())
	if err != nil {
		return true
	}
	const behind = syntax.EmptyBeginLine | syntax.EmptyBeginText | syntax.EmptyWordBoundary | syntax.EmptyNoWordBoundary
	seen := map[uint32]bool{}
	var walk func(pc uint32) bool
	walk = func(pc uint32) bool {
		if seen[pc] {
			return false
		}
		seen[pc] = true
		switch in := prog.Inst[pc]; in.Op {
		case syntax.InstAlt, syntax.InstAltMatch:
			return walk(in.Out) || walk(in.Arg)
		case syntax.InstCapture, syntax.InstNop:
			return walk(in.Out)
		case syntax.InstEmptyWidth:
			return syntax.EmptyOp(in.Arg)&behind != 0 || walk(in.Out)
		}
		return false
	}
	return walk(uint32(prog.Start))
}

// parse reads a language's rules and its config from its file's contents.
func parse(b []byte, set *lexis.Languages, funcs map[string]Matcher) (grammar, xmlConfig, error) {
	if twoStates.Match(b) {
		return grammar{}, xmlConfig{}, fmt.Errorf("a push names two states; give each a push of its own")
	}
	var x xmlRules
	var err error
	if err = xml.Unmarshal(b, &x); err != nil {
		return grammar{}, xmlConfig{}, err
	}
	flags := "m"
	if x.Config.NotMultiline {
		flags = ""
	}
	if x.Config.DotAll {
		flags += "s"
	}
	if x.Config.CaseInsensitive {
		flags += "i"
	}
	flagged := func(pattern string) string {
		if flags == "" {
			return "(?:" + pattern + ")"
		}
		return "(?" + flags + ":" + pattern + ")"
	}
	raw := map[string][]xmlRule{}
	for _, s := range x.States {
		raw[s.Name] = s.Rules
	}
	var failed error
	combined := map[string]bool{}
	made := map[*xmlRule]*rule{}
	var expand func(state string) []*rule
	expand = func(state string) []*rule {
		var out []*rule
		for i := range raw[state] {
			xr := &raw[state][i]
			if xr.Include != nil {
				out = append(out, expand(xr.Include.State)...)
				continue
			}
			if r := made[xr]; r != nil {
				out = append(out, r)
				continue
			}
			r := &rule{fn: funcs[xr.Func], ahead: xr.Ahead != ""}
			if xr.Whole != nil {
				r.emits = []emit{{kind: KindOf(xr.Whole.Name)}}
			}
			if xr.Using != nil {
				r.emits = []emit{{using: xr.Using.Lexer}}
			}
			if xr.Groups != nil {
				r.groups = true
				for _, g := range xr.Groups.Items {
					r.emits = append(r.emits, emit{kind: KindOf(g.Name), self: g.State, using: g.Lexer})
				}
			}
			for _, a := range xr.Actions {
				switch a.XMLName.Local {
				case "pop":
					r.pop += a.Depth
				case "push":
					r.push = append(r.push, a.State)
				case "combined":
					r.push = append(r.push, a.States)
					combined[a.States] = true
				}
			}
			if xr.Func != "" && r.fn == nil {
				failed = fmt.Errorf("state %s: no matcher called %s", state, xr.Func)
				continue
			}
			if r.fn == nil && xr.Ahead == "" {
				r.fn, err = words(xr.Pattern, xr.Behind, x.Config.CaseInsensitive)
				if err != nil {
					failed = fmt.Errorf("state %s: %w", state, err)
					continue
				}
			}
			if r.fn == nil {
				pattern := flagged(xr.Pattern)
				r.first, r.inLine = analyse(pattern)
				if r.ahead {
					pattern += "(" + flagged(xr.Ahead) + ")"
					r.inLine = false
				}
				if r.at0, err = regexp.Compile(`\A` + pattern); err != nil {
					failed = fmt.Errorf("state %s: %w", state, err)
					continue
				}
				behind := "(?s:.)"
				if xr.Behind != "" {
					behind = flagged(xr.Behind)
				}
				if xr.Behind != "" || looksBehind(pattern) {
					if r.at1, err = regexp.Compile(`\A` + behind + pattern); err != nil {
						failed = fmt.Errorf("state %s: %w", state, err)
						continue
					}
				}
			}
			made[xr] = r
			out = append(out, r)
		}
		return out
	}
	g := grammar{states: map[string][]*rule{}, set: set, newline: x.Config.EnsureNL}
	for state := range raw {
		g.states[state] = expand(state)
	}
	for states := range combined {
		for state := range strings.FieldsSeq(states) {
			g.states[states] = append(g.states[states], g.states[state]...)
		}
	}
	return g, x.Config, failed
}

// lexer is how to make a Lexer that runs g over a text.
func (g grammar) lexer(text string) lexis.Lexer {
	l := &ruleLexer{g: g, text: text, limit: len(text), stack: []string{"root"}}
	if g.newline && !strings.HasSuffix(text, "\n") {
		l.text += "\n"
	}
	return l.next
}

func (c xmlConfig) language(lexer func(text string) lexis.Lexer) lexis.Language {
	return lexis.Language{Names: c.Aliases, Files: c.Filenames, Lexer: lexer}
}

// Load makes a language from the contents of a file of rules. Its names
// are the aliases in the file's config and its files the filenames there,
// when it has them. Where a rule gives its text to another language, that
// language is looked for by name in set, which may be nil, and where one
// names a func, in funcs. It fails on a pattern Go's regexp cannot compile,
// naming the state it is in.
func Load(set *lexis.Languages, b []byte, funcs map[string]Matcher) (lexis.Language, error) {
	g, config, err := parse(b, set, funcs)
	if err != nil {
		return lexis.Language{}, err
	}
	return config.language(g.lexer), nil
}

// Lazy is Load for rules built into a program: it reads the config now and
// the rules the first time the language is used, and panics where Load
// would fail.
func Lazy(set *lexis.Languages, b []byte, funcs map[string]Matcher) lexis.Language {
	var config xmlConfig
	d := xml.NewDecoder(bytes.NewReader(b))
	for {
		tok, err := d.Token()
		if err != nil {
			panic(err)
		}
		if start, ok := tok.(xml.StartElement); ok && start.Name.Local == "config" {
			if err := d.DecodeElement(&config, &start); err != nil {
				panic(err)
			}
			break
		}
	}
	loaded := sync.OnceValue(func() grammar {
		g, _, err := parse(b, set, funcs)
		if err != nil {
			panic(fmt.Errorf("%v: %w", config.Aliases, err))
		}
		return g
	})
	return config.language(func(text string) lexis.Lexer { return loaded().lexer(text) })
}

// match is where r matches at pos, as regexp gives a match and its groups.
func (r *rule) match(text string, pos, lineEnd int) []int {
	if r.fn != nil {
		return r.fn(text, pos)
	}
	if r.first != nil && !r.first[text[pos]] {
		return nil
	}
	re, from := r.at0, pos
	if pos > 0 && r.at1 != nil {
		_, n := utf8.DecodeLastRuneInString(text[:pos])
		re, from = r.at1, pos-n
	}
	window := text[from:]
	if r.inLine {
		window = text[from:lineEnd]
	}
	var loc []int
	if r.groups || r.ahead {
		loc = re.FindStringSubmatchIndex(window)
	} else {
		loc = re.FindStringIndex(window)
	}
	if loc == nil {
		return nil
	}
	for i := range loc {
		if loc[i] >= 0 {
			loc[i] += from
		}
	}
	loc[0] = pos
	if r.ahead {
		loc[1] = loc[len(loc)-2]
	}
	return loc
}

// ruleLexer runs a language's rules over text, which ends in a newline of
// its own when the rules count on one and the text given had none: limit is
// where the given text ends.
type ruleLexer struct {
	g                   grammar
	text                string
	pos, lineEnd, limit int
	stack               []string
	queue               []lexis.Token
	head                int
}

func (l *ruleLexer) next() (lexis.Token, bool) {
	for {
		for l.head == len(l.queue) {
			if l.pos >= l.limit {
				return lexis.Token{}, false
			}
			l.queue, l.head = l.queue[:0], 0
			l.step()
		}
		t := l.queue[l.head]
		l.head++
		if t.Text != "" {
			return t, true
		}
	}
}

// add queues the text from from to to, as far as the given text goes.
func (l *ruleLexer) add(kind lexis.Kind, from, to int) {
	if to = min(to, l.limit); from < to {
		l.queue = append(l.queue, lexis.Token{Text: l.text[from:to], Kind: kind})
	}
}

// emit queues the text from from to to as e says.
func (l *ruleLexer) emit(e emit, from, to int) {
	to = min(to, l.limit)
	var sub lexis.Lexer
	switch {
	case from >= to:
		return
	case e.using != "":
		sub = l.g.other(e.using, l.text[from:to])
	case e.self != "":
		again := &ruleLexer{g: l.g, text: l.text[from:to], limit: to - from, stack: []string{e.self}}
		sub = again.next
	default:
		l.add(e.kind, from, to)
		return
	}
	for tok, ok := sub(); ok; tok, ok = sub() {
		l.queue = append(l.queue, tok)
	}
}

// step takes one match from where the lexer stands, or a character when no
// rule matches there.
func (l *ruleLexer) step() {
	t, pos := l.text, l.pos
	if pos >= l.lineEnd {
		l.lineEnd = len(t)
		if i := strings.IndexByte(t[pos:], '\n'); i >= 0 {
			l.lineEnd = pos + i + 1
		}
	}
	var hit *rule
	var loc []int
	for _, r := range l.g.states[l.stack[len(l.stack)-1]] {
		if loc = r.match(t, pos, l.lineEnd); loc != nil {
			hit = r
			break
		}
	}
	if hit == nil {
		if t[pos] == '\n' && len(l.stack) > 1 {
			l.stack = l.stack[:1]
			return
		}
		_, n := utf8.DecodeRuneInString(t[pos:])
		l.add(lexis.Text, pos, pos+n)
		l.pos += n
		return
	}
	end := loc[1]
	if !hit.groups {
		whole := emit{}
		if len(hit.emits) > 0 {
			whole = hit.emits[0]
		}
		l.emit(whole, pos, end)
	} else {
		at := pos
		for i, e := range hit.emits {
			if 2*i+3 >= len(loc) {
				break
			}
			a, b := loc[2*i+2], loc[2*i+3]
			if a < at || b > end {
				continue
			}
			l.add(lexis.Text, at, a)
			at = b
			l.emit(e, a, b)
		}
		l.add(lexis.Text, at, end)
	}
	state := l.stack[len(l.stack)-1]
	l.stack = l.stack[:max(len(l.stack)-hit.pop, 1)]
	for _, s := range hit.push {
		if s == "" {
			s = state
		}
		l.stack = append(l.stack, s)
	}
	if end == pos && hit.pop == 0 && len(hit.push) == 0 {
		_, n := utf8.DecodeRuneInString(t[pos:])
		l.add(lexis.Text, pos, pos+n)
		end = pos + n
	}
	l.pos = end
}
