package langs

import (
	"strings"

	"github.com/lahickey/quire/libs/lexis"
)

// htmlLexer is the lexer for HTML, and with xml for XML, which is the same
// but for what is in a script or a style. A script's text goes to
// JavaScript's lexer, or JSON's when its type says so, and a style's to
// CSS's. inTag
// says the lexer is inside a tag, after its name, and value that an
// attribute's value is due. raw is the language of what follows the tag,
// "" for more HTML, and tag the element's name.
type htmlLexer struct {
	stepper
	xml, inTag, value bool
	tag, raw          string
}

// htmlClose is where the closing tag of the element called name starts,
// from from, or the end of the text.
func htmlClose(t string, from int, name string) int {
	for i := from; ; i += 2 {
		at := strings.Index(t[i:], "</")
		if at < 0 {
			return len(t)
		}
		if i += at; len(t) >= i+2+len(name) && strings.EqualFold(t[i+2:i+2+len(name)], name) {
			return i
		}
	}
}

func htmlName(c byte) bool { return isName(c) || c == '-' || c == ':' || c == '.' }

func (l *htmlLexer) step() {
	t, pos := l.text, l.pos
	c := t[pos]
	run := func(is func(byte) bool) int {
		end := pos + 1
		for ; end < len(t) && is(t[end]); end++ {
		}
		return end
	}
	past := func(s string) int {
		if i := strings.Index(t[pos:], s); i >= 0 {
			return pos + i + len(s)
		}
		return len(t)
	}
	switch {
	case l.inTag && isSpace(c):
		l.add(lexis.Text, run(isSpace))
	case l.inTag && (c == '>' || c == '/' && strings.HasPrefix(t[pos:], "/>")):
		l.inTag, l.value = false, false
		l.add(lexis.Punct, past(">"))
		if c == '>' && l.raw != "" {
			l.other(l.raw, htmlClose(t, l.pos, l.tag))
		}
		l.raw = ""
	case l.inTag && (c == '"' || c == '\'' || l.value):
		end := run(func(d byte) bool { return !isSpace(d) && d != '>' })
		if c == '"' || c == '\'' {
			if end = strings.IndexByte(t[pos+1:], c) + pos + 2; end < pos+2 {
				end = len(t)
			}
		}
		if l.tag == "script" && strings.Contains(t[pos:end], "json") {
			l.raw = "json"
		}
		l.value = false
		l.add(lexis.String, end)
	case l.inTag && c == '=':
		l.value = true
		l.add(lexis.Operator, pos+1)
	case l.inTag && htmlName(c):
		l.add(lexis.Type, run(htmlName))
	case l.inTag:
		l.add(lexis.Text, pos+1)
	case strings.HasPrefix(t[pos:], "<!--"):
		l.add(lexis.Comment, past("-->"))
	case strings.HasPrefix(t[pos:], "<![CDATA["):
		l.add(lexis.Preproc, past("]]>"))
	case strings.HasPrefix(t[pos:], "<?"):
		l.add(lexis.Preproc, past("?>"))
	case strings.HasPrefix(t[pos:], "<!"):
		end := past(">")
		if strings.Contains(t[pos:end], "[") {
			end = past("]>")
		}
		l.add(lexis.Preproc, end)
	case c == '<' && pos+1 < len(t) && (htmlName(t[pos+1]) || t[pos+1] == '/'):
		name := pos + 1
		if t[name] == '/' {
			name++
		}
		end := name
		for ; end < len(t) && htmlName(t[end]); end++ {
		}
		l.add(lexis.Punct, name)
		l.add(lexis.Tag, end)
		l.inTag, l.tag = true, strings.ToLower(t[name:end])
		if l.raw = ""; !l.xml && t[pos+1] != '/' {
			l.raw = map[string]string{"script": "js", "style": "css"}[l.tag]
		}
	case c == '&':
		if end := run(func(c byte) bool { return isName(c) || c == '#' }); end < len(t) && t[end] == ';' && end > pos+1 {
			l.add(lexis.Builtin, end+1)
			return
		}
		l.add(lexis.Text, pos+1)
	default:
		l.add(lexis.Text, run(func(c byte) bool { return c != '<' && c != '&' }))
	}
}

func htmlWith(set *lexis.Languages, xml bool) func(string) lexis.Lexer {
	return func(text string) lexis.Lexer {
		l := &htmlLexer{set: set, text: text, xml: xml}
		l.more = l.step
		return l.next
	}
}

func htmlLanguage(set *lexis.Languages) lexis.Language {
	return lexis.Language{Names: []string{"html"}, Files: []string{"*.html", "*.htm", "*.xhtml"}, Lexer: htmlWith(set, false)}
}

func xmlLanguage(set *lexis.Languages) lexis.Language {
	return lexis.Language{
		Names: []string{"xml"},
		Files: []string{
			"*.xml", "*.xsl", "*.rss", "*.xslt", "*.xsd", "*.wsdl", "*.wsf", "*.svg", "*.qrc", "*.csproj", "*.vcxproj", "*.fsproj",
			"*.plist", "*.xcprivacy", "*.storyboard", "*.xib", "*.entitlements", "*.xcscheme", "*.xcworkspacedata",
		},
		Lexer: htmlWith(set, true),
	}
}
