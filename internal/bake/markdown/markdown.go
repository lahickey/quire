package markdown

import (
	"bytes"
	"cmp"
	"fmt"
	"html"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/model"
)

const headerSep = "---"

func splitFrontmatter(raw []byte) (header, body []byte, err error) {
	lines := bytes.Split(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n")), []byte("\n"))
	i := 0
	for i < len(lines) && len(bytes.TrimSpace(lines[i])) == 0 {
		i++
	}
	if i >= len(lines) || string(bytes.TrimSpace(lines[i])) != headerSep {
		return nil, nil, fmt.Errorf("missing opening %q frontmatter fence", headerSep)
	}
	i++
	start := i
	for i < len(lines) {
		if string(bytes.TrimSpace(lines[i])) == headerSep {
			return bytes.Join(lines[start:i], []byte("\n")), bytes.Join(lines[i+1:], []byte("\n")), nil
		}
		i++
	}
	return nil, nil, fmt.Errorf("missing closing %q frontmatter fence", headerSep)
}

type embeds struct {
	list []journal.Embed
	at   map[journal.Embed]int
}

func (e *embeds) hole(em journal.Embed) string {
	i, ok := e.at[em]
	if !ok {
		i = len(e.list)
		e.list = append(e.list, em)
		e.at[em] = i
	}
	return journal.Hole(i)
}

func headingID(h *ast.Heading) string {
	v, ok := h.AttributeString("id")
	if !ok {
		return ""
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	case string:
		return t
	default:
		return ""
	}
}

func nodeText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := n.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(src))
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func collectHeadings(n ast.Node, src []byte) []journal.Heading {
	var out []journal.Heading
	_ = ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		out = append(out, journal.Heading{
			Level: h.Level,
			Text:  nodeText(h, src),
			ID:    headingID(h),
		})
		return ast.WalkSkipChildren, nil
	})
	return out
}

func remoteImage(dest string) bool {
	return strings.HasPrefix(dest, "https://") ||
		strings.HasPrefix(dest, "http://") ||
		strings.HasPrefix(dest, "//") ||
		strings.HasPrefix(dest, "data:")
}

func standaloneImage(p *ast.Paragraph, src []byte) (*ast.Image, bool) {
	var img *ast.Image
	for c := p.FirstChild(); c != nil; c = c.NextSibling() {
		switch n := c.(type) {
		case *ast.Image:
			if img != nil {
				return nil, false
			}
			img = n
		case *ast.Text:
			if len(bytes.TrimSpace(n.Segment.Value(src))) != 0 {
				return nil, false
			}
		default:
			return nil, false
		}
	}
	if img == nil || remoteImage(string(img.Destination)) {
		return nil, false
	}
	return img, true
}

var kindRawBlock = ast.NewNodeKind("RawBlock")

type rawBlock struct {
	ast.BaseBlock
	html string
}

func (n *rawBlock) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"html": n.html}, nil)
}

func (n *rawBlock) Kind() ast.NodeKind { return kindRawBlock }

func (n *rawBlock) IsRaw() bool { return true }

var kindRawInline = ast.NewNodeKind("RawInline")

type rawInline struct {
	ast.BaseInline
	html string
}

func (n *rawInline) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"html": n.html}, nil)
}

func (n *rawInline) Kind() ast.NodeKind { return kindRawInline }

type spanParser struct{}

var spanSyntax = regexp.MustCompile(`^\[([^\]\n]+)\]\{((?:\s*\.[A-Za-z_][\w-]*)+)\s*\}`)

func (spanParser) Trigger() []byte { return []byte{'['} }

func (spanParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, _ := block.PeekLine()
	m := spanSyntax.FindSubmatch(line)
	if m == nil {
		return nil
	}
	block.Advance(len(m[0]))
	classes := strings.ReplaceAll(strings.Join(strings.Fields(string(m[2])), " "), ".", "")
	return &rawInline{html: `<span class="` + classes + `">` + html.EscapeString(string(m[1])) + "</span>"}
}

type rawHTMLRenderer struct{}

func renderRawHTML(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	switch n := node.(type) {
	case *rawBlock:
		_, _ = w.WriteString(n.html)
	case *rawInline:
		_, _ = w.WriteString(n.html)
	}
	return ast.WalkContinue, nil
}

func (rawHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindRawBlock, renderRawHTML)
	reg.Register(kindRawInline, renderRawHTML)
}

type imgTransform struct {
	src    []byte
	doc    string
	embeds *embeds
}

type imgTarget struct {
	node  ast.Node
	img   *ast.Image
	block bool
}

func (t *imgTransform) hole(img *ast.Image, block bool) string {
	alt := nodeText(img, t.src)
	return t.embeds.hole(journal.Embed{Picture: journal.Resolve(t.doc, string(img.Destination)), Alt: alt, Caption: cmp.Or(string(img.Title), alt), Figure: block})
}

func (t *imgTransform) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	var found []imgTarget
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if para, ok := n.(*ast.Paragraph); ok {
			img, yes := standaloneImage(para, t.src)
			if !yes {
				return ast.WalkContinue, nil
			}
			found = append(found, imgTarget{node: para, img: img, block: true})
			return ast.WalkSkipChildren, nil
		}
		img, ok := n.(*ast.Image)
		if !ok || remoteImage(string(img.Destination)) {
			return ast.WalkContinue, nil
		}
		found = append(found, imgTarget{node: img, img: img})
		return ast.WalkSkipChildren, nil
	})
	for _, f := range found {
		html := t.hole(f.img, f.block)
		var repl ast.Node = &rawInline{html: html}
		if f.block {
			repl = &rawBlock{html: html}
		}
		f.node.Parent().ReplaceChild(f.node.Parent(), f.node, repl)
	}
}

var calloutKinds = []string{"note", "tip", "important", "warning", "caution"}

func calloutMarker(p *ast.Paragraph, src []byte) (string, []ast.Node) {
	var b strings.Builder
	var marker []ast.Node
	for c := p.FirstChild(); c != nil; c = c.NextSibling() {
		t, ok := c.(*ast.Text)
		if !ok {
			return "", nil
		}
		b.Write(t.Segment.Value(src))
		marker = append(marker, t)
		if t.SoftLineBreak() || t.HardLineBreak() {
			break
		}
	}
	kind, ok := strings.CutPrefix(strings.TrimSpace(b.String()), "[!")
	kind, closed := strings.CutSuffix(strings.ToLower(kind), "]")
	if !ok || !closed || !slices.Contains(calloutKinds, kind) {
		return "", nil
	}
	return kind, marker
}

type calloutTransform struct{ src []byte }

func (t calloutTransform) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	var quotes []*ast.Blockquote
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if q, ok := n.(*ast.Blockquote); ok && entering {
			quotes = append(quotes, q)
		}
		return ast.WalkContinue, nil
	})
	for _, q := range quotes {
		p, ok := q.FirstChild().(*ast.Paragraph)
		if !ok {
			continue
		}
		kind, marker := calloutMarker(p, t.src)
		if kind == "" || (len(marker) == p.ChildCount() && q.ChildCount() == 1) {
			continue
		}
		for _, n := range marker {
			p.RemoveChild(p, n)
		}
		if p.ChildCount() == 0 {
			q.RemoveChild(q, p)
		}
		title := ast.NewParagraph()
		title.SetAttributeString("class", []byte("callout-title"))
		title.AppendChild(title, ast.NewString([]byte(strings.ToUpper(kind[:1])+kind[1:])))
		q.InsertBefore(q, q.FirstChild(), title)
		q.SetAttributeString("class", []byte("callout callout--"+kind))
	}
}

var converts atomic.Int64

func Converts() int64 { return converts.Load() }

type Widget = func(info, body string) (string, error)

func palette(_, body string) (string, error) {
	var roles []string
	for _, r := range (model.Palette{}).Roles() {
		roles = append(roles, r.Name)
	}
	var b strings.Builder
	b.WriteString(`<div class="palette"><span></span>`)
	for i := range 12 {
		fmt.Fprintf(&b, `<span>%d</span>`, i+1)
	}
	for role := range strings.FieldsSeq(body) {
		if !slices.Contains(roles, role) {
			return "", fmt.Errorf("palette: unknown role %q, want one of %s", role, strings.Join(roles, " "))
		}
		fmt.Fprintf(&b, `<span>%s</span>`, role)
		for i := range 12 {
			fmt.Fprintf(&b, `<span style="background:var(--qr-%s-%d)"></span>`, role, i+1)
		}
	}
	b.WriteString(`</div>`)
	return b.String(), nil
}

func fences(names []string, em *embeds) map[string]Widget {
	out := map[string]Widget{"palette": palette}
	for _, name := range append([]string{"mermaid"}, names...) {
		out[name] = func(info, body string) (string, error) {
			return em.hole(journal.Embed{Fence: name, Info: info, Body: body}), nil
		}
	}
	return out
}

func fenceText(b *ast.FencedCodeBlock, src []byte) string {
	var s strings.Builder
	for i := range b.Lines().Len() {
		line := b.Lines().At(i)
		s.Write(line.Value(src))
	}
	return s.String()
}

func fenceInfo(b *ast.FencedCodeBlock, src []byte) (name, info string) {
	if b.Info == nil {
		return "", ""
	}
	name, info, _ = strings.Cut(strings.TrimSpace(string(b.Info.Segment.Value(src))), " ")
	return name, strings.TrimSpace(info)
}

type fenceTransform struct {
	src     []byte
	widgets map[string]Widget
	err     error
}

func (t *fenceTransform) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	var found []*ast.FencedCodeBlock
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if b, ok := n.(*ast.FencedCodeBlock); ok && entering && t.widgets[string(b.Language(t.src))] != nil {
			found = append(found, b)
		}
		return ast.WalkContinue, nil
	})
	for _, b := range found {
		name, info := fenceInfo(b, t.src)
		html, err := t.widgets[name](info, fenceText(b, t.src))
		if err != nil {
			t.err = fmt.Errorf("%s: %w", name, err)
			return
		}
		b.Parent().ReplaceChild(b.Parent(), b, &rawBlock{html: html})
	}
}

func Render(src, doc string, fenceNames []string) (string, []journal.Heading, []journal.Embed, error) {
	if src == "" {
		return "", nil, nil, nil
	}
	srcb := []byte(src)
	em := &embeds{at: map[journal.Embed]int{}}
	tf := &imgTransform{src: srcb, doc: doc, embeds: em}
	ft := &fenceTransform{src: srcb, widgets: fences(fenceNames, em)}
	md := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("github-dark"),
				highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithInlineParsers(util.Prioritized(spanParser{}, 150)),
			parser.WithASTTransformers(util.Prioritized(ft, 50), util.Prioritized(tf, 100), util.Prioritized(calloutTransform{srcb}, 200)),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(),
			renderer.WithNodeRenderers(util.Prioritized(rawHTMLRenderer{}, 0)),
		),
	)
	node := md.Parser().Parse(text.NewReader(srcb))
	if ft.err != nil {
		return "", nil, nil, ft.err
	}
	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, srcb, node); err != nil {
		return "", nil, nil, err
	}
	return buf.String(), collectHeadings(node, srcb), em.list, nil
}

func Convert(name string, raw []byte, fenceNames []string) (journal.Document, error) {
	converts.Add(1)
	header, body, err := splitFrontmatter(raw)
	if err != nil {
		return journal.Document{}, err
	}
	html, headings, embeds, err := Render(string(body), name, fenceNames)
	if err != nil {
		return journal.Document{}, fmt.Errorf("markdown: %w", err)
	}
	return journal.Document{Path: name, Front: string(header), HTML: html, Words: len(strings.Fields(string(body))), Headings: headings, Embeds: embeds}, nil
}
