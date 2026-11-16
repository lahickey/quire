package markdown

import (
	"bytes"
	"cmp"
	"fmt"
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/libs/lexis"
	"github.com/lahickey/quire/libs/lexis/langs"
	"github.com/lahickey/quire/model"
)

const headerSep = "---"

var kindRawBlock = ast.NewNodeKind("RawBlock")

var kindRawInline = ast.NewNodeKind("RawInline")

var spanSyntax = regexp.MustCompile(`^\[([^\]\n]+)\]\{((?:\s*\.[A-Za-z_][\w-]*)+)\s*\}`)

var calloutKinds = []string{"note", "tip", "important", "warning", "caution"}

var converts atomic.Int64

var languages = sync.OnceValue(func() *lexis.Languages {
	var set lexis.Languages
	langs.Add(&set)
	return &set
})

type embeds struct {
	list []journal.Embed
	at   map[journal.Embed]int
}

type rawBlock struct {
	ast.BaseBlock
	html string
}

type rawInline struct {
	ast.BaseInline
	html string
}

type spanParser struct{}

type rawHTMLRenderer struct{}

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

type calloutTransform struct{ src []byte }

type linkTransform struct{}

type fenceTransform struct {
	src     []byte
	widgets map[string]model.Widget
	err     error
}

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

func remote(dest string) bool {
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
	if img == nil || remote(string(img.Destination)) {
		return nil, false
	}
	return img, true
}

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

func (t *imgTransform) hole(img *ast.Image, block bool) string {
	alt := nodeText(img, t.src)
	return t.embeds.hole(journal.Embed{Picture: journal.Resolve(t.doc, string(img.Destination)), Alt: alt, Caption: cmp.Or(string(img.Title), alt), Figure: block})
}

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

// paletteGrid draws a grid of colours: down the side and across the top are
// what it is labelled with, and each cell is the custom property named.
func paletteGrid(b *strings.Builder, name string, down, across []string, property func(down, across string) string) {
	if len(down) == 0 || len(across) == 0 {
		return
	}
	attr := ""
	if name != "" {
		attr = ` data-palette="` + name + `"`
	}
	fmt.Fprintf(b, `<div class="palette" style="grid-template-columns:max-content repeat(%d,1fr)"%s><span class="palette-label">%s</span>`, len(across), attr, name)
	for _, a := range across {
		fmt.Fprintf(b, `<span>%s</span>`, a)
	}
	for _, d := range down {
		fmt.Fprintf(b, `<span class="palette-label">%s</span>`, d)
		for _, a := range across {
			fmt.Fprintf(b, `<span style="background:var(--qr-%s)"></span>`, property(d, a))
		}
	}
	b.WriteString(`</div>`)
}

// palette draws the roles its body names: each range as a row of its twelve
// steps, and the rest side by side with a row for each of their steps.
func palette(name, body string) (string, error) {
	roles := (model.Palette{}).Roles()
	var names, ranges, others, twelve, steps []string
	for _, r := range roles {
		names = append(names, r.Name)
	}
	for word := range strings.FieldsSeq(body) {
		i := slices.Index(names, word)
		switch {
		case word == "all":
			for _, r := range roles {
				if r.Range {
					ranges = append(ranges, r.Name)
				} else {
					others = append(others, r.Name)
				}
			}
		case i < 0:
			return "", fmt.Errorf("palette: unknown role %q, want all or one of %s", word, strings.Join(names, " "))
		case roles[i].Range:
			ranges = append(ranges, word)
		default:
			others = append(others, word)
		}
	}
	for i := range 12 {
		twelve = append(twelve, strconv.Itoa(i+1))
	}
	for _, s := range model.Steps {
		steps = append(steps, s.Name)
	}
	var b strings.Builder
	name = html.EscapeString(name)
	paletteGrid(&b, name, ranges, twelve, func(role, step string) string { return role + "-" + step })
	paletteGrid(&b, name, steps, others, func(step, role string) string { return role + "-" + step })
	return b.String(), nil
}

func fences(names []string, em *embeds) map[string]model.Widget {
	out := map[string]model.Widget{"palette": palette}
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

func (n *rawBlock) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"html": n.html}, nil)
}

func (n *rawBlock) Kind() ast.NodeKind { return kindRawBlock }

func (n *rawBlock) IsRaw() bool { return true }

func (n *rawInline) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"html": n.html}, nil)
}

func (n *rawInline) Kind() ast.NodeKind { return kindRawInline }

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

func (rawHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindRawBlock, renderRawHTML)
	reg.Register(kindRawInline, renderRawHTML)
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
		if !ok || remote(string(img.Destination)) {
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

func (linkTransform) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if h, ok := n.(*ast.Heading); ok && entering && headingID(h) != "" {
			h.AppendChild(h, &rawInline{html: `<a class="heading-anchor" href="#` + headingID(h) + `" aria-label="Link to this section"></a>`})
		}
		l, ok := n.(*ast.Link)
		if !ok || !entering || remote(string(l.Destination)) || bytes.HasPrefix(l.Destination, []byte("/")) {
			return ast.WalkContinue, nil
		}
		file, frag, _ := bytes.Cut(l.Destination, []byte("#"))
		if name, ok := bytes.CutSuffix(file, []byte(".md")); ok {
			l.Destination = name
			if len(frag) > 0 {
				l.Destination = slices.Concat(name, []byte("#"), frag)
			}
		}
		return ast.WalkContinue, nil
	})
}

func Converts() int64 { return converts.Load() }

func (t *fenceTransform) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	var found []*ast.FencedCodeBlock
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if b, ok := n.(*ast.FencedCodeBlock); ok && entering {
			found = append(found, b)
		}
		return ast.WalkContinue, nil
	})
	for _, b := range found {
		name, info := fenceInfo(b, t.src)
		p := b.Parent()
		body := fenceText(b, t.src)
		if t.widgets[name] == nil {
			code := html.EscapeString(body)
			if next := languages().Named(name, body); next != nil {
				code = string(lexis.AppendHTML(nil, next))
			}
			p.ReplaceChild(p, b, &rawBlock{html: model.CodeOpen + "<pre><code>" + code + "</code></pre>" + model.CodeClose})
			continue
		}
		html, err := t.widgets[name](info, body)
		if err != nil {
			t.err = fmt.Errorf("%s: %w", name, err)
			return
		}
		p.ReplaceChild(p, b, &rawBlock{html: html})
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
		goldmark.WithExtensions(extension.Table),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithInlineParsers(util.Prioritized(spanParser{}, 150)),
			parser.WithASTTransformers(util.Prioritized(ft, 50), util.Prioritized(tf, 100), util.Prioritized(calloutTransform{srcb}, 200), util.Prioritized(linkTransform{}, 300)),
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
