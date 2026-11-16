package langs

import (
	"github.com/lahickey/quire/libs/lexis"
	"regexp"
	"strings"
)

var (
	markdownFence  = regexp.MustCompile("^ {0,3}(```+|~~~+) *([^ `\n]*)")
	markdownList   = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s`)
	markdownInline = regexp.MustCompile("(`+[^`\n]+`+)|(\\*\\*[^*\n]+\\*\\*|__[^_\n]+__)|(\\*[^*\\s][^*\n]*\\*|\\b_[^_\\s][^_\n]*_\\b)|(\\[[^\\]\n]*\\])(\\([^)\n]*\\))?|(<https?://[^>\\s]+>)")
)

// markdownLexer is the lexer for Markdown, a line at a time. Fenced code is given
// to the lexer for the language the fence names, when Lexis has it.
type markdownLexer struct {
	stepper
	// closing says the next line is the one that closes a block.
	closing bool
}

func (l *markdownLexer) put(kind lexis.Kind, text string) {
	if text != "" {
		l.queue = append(l.queue, lexis.Token{Text: text, Kind: kind})
	}
}

// inline queues a line's text, with its code, emphasis and links marked.
func (l *markdownLexer) inline(s string) {
	at := 0
	for _, m := range markdownInline.FindAllStringSubmatchIndex(s, -1) {
		l.put(lexis.Text, s[at:m[0]])
		at = m[1]
		switch {
		case m[2] >= 0:
			l.put(lexis.String, s[m[2]:m[3]])
		case m[4] >= 0:
			l.put(lexis.Strong, s[m[4]:m[5]])
		case m[6] >= 0:
			l.put(lexis.Emph, s[m[6]:m[7]])
		case m[8] >= 0:
			l.put(lexis.Tag, s[m[8]:m[9]])
			if m[10] >= 0 {
				l.put(lexis.String, s[m[10]:m[11]])
			}
		default:
			l.put(lexis.String, s[m[12]:m[13]])
		}
	}
	l.put(lexis.Text, s[at:])
}

// block gives the text up to the line that closes a block opened at the
// line before from, and where that line starts.
func (l *markdownLexer) block(from int, closes func(line string) bool) (string, int) {
	for at := from; at < len(l.text); {
		end := lineEnd(l.text, at)
		if closes(strings.TrimRight(l.text[at:end], "\r\n")) {
			return l.text[from:at], at
		}
		at = end
	}
	return l.text[from:], len(l.text)
}

// line takes the next line, or the whole of a block that starts on it.
func (l *markdownLexer) line() {
	t, from := l.text, l.pos
	end := lineEnd(t, from)
	line := strings.TrimRight(t[from:end], "\n")
	l.pos = end
	inside := func(name string, closes func(string) bool) {
		l.put(lexis.Keyword, t[from:end])
		var body string
		body, l.pos = l.block(end, closes)
		l.closing = l.pos < len(t)
		if l.sub = l.set.Named(name, body); l.sub == nil {
			l.put(lexis.String, body)
		}
	}
	hashes := len(line) - len(strings.TrimLeft(line, "#"))
	switch m := markdownFence.FindStringSubmatch(line); {
	case l.closing:
		l.closing = false
		l.put(lexis.Keyword, line)
	case from == 0 && line == "---":
		inside("yaml", func(s string) bool { return s == "---" })
		return
	case m != nil:
		inside(m[2], func(s string) bool { return strings.HasPrefix(strings.TrimLeft(s, " "), m[1]) })
		return
	case hashes == 1 && (line == "#" || line[1] == ' '):
		l.put(lexis.Heading, line)
	case hashes > 1 && hashes <= 6 && (len(line) == hashes || line[hashes] == ' '):
		l.put(lexis.Subheading, line)
	case strings.Trim(line, "-*_ ") == "" && len(strings.ReplaceAll(line, " ", "")) >= 3:
		l.put(lexis.Keyword, line)
	case strings.HasPrefix(strings.TrimLeft(line, " "), ">"):
		l.put(lexis.Emph, line)
	default:
		mark := markdownList.FindString(line)
		l.put(lexis.Keyword, mark)
		l.inline(line[len(mark):])
	}
	l.put(lexis.Text, t[from+len(line):end])
}

func markdownLanguage(set *lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"markdown", "md"}, Files: []string{"*.md", "*.markdown"}, Lexer: func(text string) lexis.Lexer {
		l := &markdownLexer{set: set, text: text}
		l.more = l.line
		return l.next
	}}
}
