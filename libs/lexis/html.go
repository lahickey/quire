package lexis

import "html"

var kindNames = [...]string{
	Text:       "text",
	Comment:    "comment",
	Keyword:    "keyword",
	Type:       "type",
	Builtin:    "builtin",
	String:     "string",
	Number:     "number",
	Function:   "function",
	Tag:        "tag",
	Preproc:    "preproc",
	Heading:    "heading",
	Subheading: "subheading",
	Strong:     "strong",
	Emph:       "emph",
	Inserted:   "inserted",
	Deleted:    "deleted",
	Operator:   "operator",
	Punct:      "punct",
}

// String is the kind's name, in lower case.
func (k Kind) String() string { return kindNames[k] }

// AppendHTML appends next's tokens to dst as HTML and gives the result: the
// text escaped, and each token that is not Text in a span whose class is
// lx- and its kind's name, such as lx-keyword.
func AppendHTML(dst []byte, next Lexer) []byte {
	for t, ok := next(); ok; t, ok = next() {
		if t.Kind == Text {
			dst = append(dst, html.EscapeString(t.Text)...)
			continue
		}
		dst = append(dst, `<span class="lx-`...)
		dst = append(dst, kindNames[t.Kind]...)
		dst = append(dst, `">`...)
		dst = append(dst, html.EscapeString(t.Text)...)
		dst = append(dst, "</span>"...)
	}
	return dst
}
