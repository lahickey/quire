package lexis_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/lahickey/quire/libs/lexis"
	"github.com/lahickey/quire/libs/lexis/langs"
	"github.com/lahickey/quire/libs/lexis/rules"
)

func all() *lexis.Languages {
	var set lexis.Languages
	langs.Add(&set)
	return &set
}

var languages = all()

// corpus is a file of real size in each language, in a folder named for
// the language: the file's name and its text.
var corpus = func() map[string][2]string {
	out := map[string][2]string{}
	paths, err := filepath.Glob("_corpus/*/*")
	if err != nil || len(paths) == 0 {
		panic("no corpus")
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			panic(err)
		}
		out[filepath.Base(filepath.Dir(p))] = [2]string{filepath.Base(p), string(b)}
	}
	return out
}()

func drain(next lexis.Lexer) []lexis.Token {
	var out []lexis.Token
	for tok, ok := next(); ok; tok, ok = next() {
		out = append(out, tok)
	}
	return out
}

// Every language Lexis comes with has a file in the corpus, found by the
// file's name and by the language's. Whatever the language, and wherever
// the text is cut off, the tokens are the text.
func TestTokensAreTheText(t *testing.T) {
	if len(corpus) != 34 {
		t.Errorf("the corpus has %d languages", len(corpus))
	}
	for name, f := range corpus {
		file, text := f[0], f[1]
		if languages.Named(name, "") == nil || languages.File(file, text) == nil {
			t.Errorf("%s: no lexer by name or for %s", name, file)
			continue
		}
		for cut := len(text); cut >= 0; cut -= 37 {
			var b strings.Builder
			for _, tok := range drain(languages.File(file, text[:cut])) {
				b.WriteString(tok.Text)
			}
			if b.String() != text[:cut] {
				t.Fatalf("%s cut at %d: the tokens are not the text", name, cut)
			}
		}
	}
}

func TestKinds(t *testing.T) {
	for _, c := range []struct {
		name, text string
		kind       lexis.Kind
	}{
		{"bash", "#!/usr/bin/env bash\n", lexis.Preproc}, {"bash", "for", lexis.Keyword}, {"bash", "echo", lexis.Builtin}, {"bash", "$PROG", lexis.Builtin},
		{"bash", "'%s %s\\n'", lexis.String}, {"bash", "# Usage and logging.", lexis.Comment}, {"bash", "1024", lexis.Number}, {"bash", "<<EOF\nkeep $1\nEOF", lexis.String},
		{"bash", "$(", lexis.Keyword},
		{"fish", "function", lexis.Keyword}, {"fish", "set", lexis.Builtin}, {"fish", "$argv", lexis.Builtin}, {"fish", "end", lexis.Keyword},
		{"just", "# Vet and lint.", lexis.Comment}, {"just", "build", lexis.Function}, {"just", "set", lexis.Keyword}, {"just", `"bash"`, lexis.String},
		{"just", "{{ dist }}", lexis.Builtin}, {"just", "echo", lexis.Builtin}, {"just", "source_directory", lexis.Function}, {"just", "[private]", lexis.Type},
		{"go", "Post", lexis.Function}, {"go", "// Balance is what an account holds, and whether it has ever been posted to.", lexis.Comment}, {"go", "string", lexis.Type},
		{"go", "nil", lexis.Keyword}, {"go", "0.0125", lexis.Number}, {"go", ":=", lexis.Operator}, {"go", "{", lexis.Punct}, {"c", "->", lexis.Operator}, {"c", ";", lexis.Punct}, {"c", `\n`, lexis.Builtin}, {"c", `"%s: %s`, lexis.String}, {"cpp", "[[nodiscard]]", lexis.Type},
		{"c", "#include", lexis.Preproc}, {"c", "<stdio.h>", lexis.String}, {"c", "/* always a power of two */", lexis.Comment}, {"c", "fprintf", lexis.Function},
		{"c", "0x7F", lexis.Number}, {"c", "0.75e0", lexis.Number}, {"c", `'\\'`, lexis.String}, {"c", "const", lexis.Keyword}, {"c", "char", lexis.Type},
		{"cpp", "template", lexis.Keyword}, {"cpp", `R"(hand "left")"`, lexis.String}, {"cpp", "1'000'000ULL", lexis.Number}, {"cpp", "// namespace scene", lexis.Comment},
		{"objc", "@interface", lexis.Keyword}, {"objc", "#import", lexis.Preproc}, {"objc", "instancetype", lexis.Type}, {"objc", "nil", lexis.Keyword}, {"objc", `@"InventoryDidChange"`, lexis.String}, {"objc", "@0.75", lexis.Number}, {"objc", "@selector", lexis.Keyword}, {"objc", "didChangeItem", lexis.Function}, {"objc", "level", lexis.Function}, {"objc", "copy", lexis.Keyword}, {"objc", "weak", lexis.Keyword},
		{"objcpp", "@implementation", lexis.Keyword}, {"objcpp", "template", lexis.Keyword}, {"objcpp", "BOOL", lexis.Type},
		{"markdown", "# Running the ledger service", lexis.Heading}, {"markdown", "## Install", lexis.Subheading}, {"markdown", "`amd64`", lexis.String},
		{"markdown", "**what to type**", lexis.Strong}, {"markdown", "*why*", lexis.Emph}, {"markdown", "_Backups._", lexis.Emph}, {"markdown", "[design notes]", lexis.Tag}, {"markdown", "if", lexis.Keyword},
		{"markdown", "title", lexis.Tag},
		{"yaml", "name", lexis.Tag}, {"yaml", "# CI for the ledger service: test on every push, release on a tag.", lexis.Comment},
		{"yaml", "b", lexis.Tag}, {"yaml", "3", lexis.Number}, {"yaml", "e", lexis.Text},
		{"yaml", "\n    one long line\n    folded into a paragraph", lexis.String},
		{"toml", `"inventory"`, lexis.String}, {"toml", "3", lexis.Number},
		{"json", "0.2", lexis.Number}, {"json", `"name"`, lexis.Tag}, {"json", `"MIT"`, lexis.String}, {"json", "true", lexis.Keyword}, {"json", ":", lexis.Punct},
		{"templ", "templ", lexis.Keyword}, {"templ", "Layout", lexis.Function}, {"templ", "header", lexis.Tag}, {"templ", "class", lexis.Type},
		{"templ", `"logo"`, lexis.String}, {"templ", "<!-- the price of the whole line, not of one -->", lexis.Comment}, {"templ", "if", lexis.Keyword},
		{"templ", "@Header", lexis.Function}, {"templ", "meta", lexis.Tag}, {"templ", "const", lexis.Keyword}, {"templ", "var", lexis.Keyword},
		{"templ", "func", lexis.Keyword}, {"templ", "string", lexis.Type}, {"templ", "money", lexis.Function}, {"templ", "}", lexis.Punct},
		{"css", "/* Tokens */", lexis.Comment}, {"css", "color", lexis.Keyword}, {"css", "--ink", lexis.Builtin}, {"css", "#1b1d22", lexis.Number}, {"css", "html", lexis.Tag},
		{"css", "0.5rem", lexis.Number}, {"css", "@media", lexis.Keyword}, {"css", "clamp", lexis.Function}, {"css", "::before", lexis.Keyword},
		{"scss", "// theme.scss - the shop's theme: tokens, a card and its states.", lexis.Comment}, {"scss", "$gap", lexis.Builtin}, {"scss", "#{$name}", lexis.Builtin},
		{"scss", ".card", lexis.Type}, {"scss", "h2", lexis.Tag}, {"scss", "transform", lexis.Keyword}, {"scss", "!default", lexis.Preproc}, {"scss", ":hover", lexis.Keyword},
		{"html", "<!DOCTYPE html>", lexis.Preproc}, {"html", "&mdash;", lexis.Builtin}, {"html", `"@context"`, lexis.Tag}, {"html", "hidden", lexis.Type},
		{"html", "<!-- The header is repeated on every page; keep it in step with partials/header.html -->", lexis.Comment},
		{"xml", "<!-- One book per title; prices are in pence. -->", lexis.Comment}, {"xml", "catalogue", lexis.Tag}, {"xml", "season", lexis.Type},
		{"xml", `<?xml version="1.0" encoding="UTF-8" standalone="no"?>`, lexis.Preproc}, {"xml", "&publisher;", lexis.Builtin}, {"xml", `"spring"`, lexis.String},
		{"html", "body", lexis.Tag}, {"html", "class", lexis.Type}, {"html", `"logo"`, lexis.String}, {"html", "const", lexis.Keyword}, {"html", "margin", lexis.Keyword},
		{"python", "# One record for each line that is a log line.", lexis.Comment}, {"python", "def", lexis.Keyword}, {"python", "parse", lexis.Function},
		{"python", "0x4000", lexis.Number}, {"python", `"debug"`, lexis.String}, {"python", "Record", lexis.Type}, {"python", "@dataclass", lexis.Keyword}, {"python", "match", lexis.Keyword},
		{"python", "case", lexis.Keyword}, {"python", ":<7", lexis.String}, {"python", "upper", lexis.Function}, {"python", "__name__", lexis.Builtin}, {"python", `f"`, lexis.String},
		{"python", "self", lexis.Builtin}, {"python", "#!/usr/bin/env python3", lexis.Preproc},
		{"rust", "fn", lexis.Keyword}, {"rust", "main", lexis.Function}, {"rust", "\"bolt-m4: 40\nhinge: 12\nwasher\"#", lexis.String}, {"rust", "1.0e2", lexis.Number},
		{"rust", "u32", lexis.Type},
		{"typescript", "interface", lexis.Keyword}, {"typescript", `/:([A-Za-z_]\w*)(\?)?/g`, lexis.String}, {"java", "class", lexis.Keyword}, {"sql", "SELECT", lexis.Keyword},
		{"docker", "FROM", lexis.Keyword}, {"docker", "$HOME", lexis.Builtin}, {"diff", "+\t\"errors\"\n", lexis.Inserted}, {"diff", "-\tif e.Kind == Debit {\n", lexis.Deleted},
		{"ini", "[Unit]", lexis.Keyword}, {"csharp", "namespace", lexis.Keyword}, {"kotlin", "fun", lexis.Keyword}, {"zig", "fn", lexis.Keyword},
		{"lua", "-- @module packer", lexis.Comment}, {"lua", "--[[ A plugin spec is a table whose first element is the repository. ]]", lexis.Comment}, {"lua", "local", lexis.Keyword},
		{"lua", "[==[\n  plugins [[ready]]\n]==]", lexis.String}, {"lua", "1e3", lexis.Number}, {"lua", "function", lexis.Keyword},
		{"nix", "# A development shell and a package for a small Go service.", lexis.Comment}, {"nix", "let", lexis.Keyword}, {"nix", "in", lexis.Keyword}, {"nix", "8080", lexis.Number},
		{"nix", "inherit", lexis.Keyword},
		{"javascript", "formatMoney", lexis.Function}, {"javascript", "from", lexis.Keyword}, {"javascript", " × ${", lexis.String},
		{"jsx", "aside", lexis.Tag}, {"jsx", "className", lexis.Type}, {"jsx", `/^\s*$/`, lexis.String}, {"jsx", "/", lexis.Operator}, {"jsx", "Nothing here yet.", lexis.Text},
		{"jsx", "Remove ${", lexis.String}, {"jsx", "Cart.Footer", lexis.Tag},
		{"tsx", "thead", lexis.Tag}, {"tsx", "<", lexis.Operator}, {"tsx", "ReactNode", lexis.Type}, {"tsx", "keyof", lexis.Keyword}, {"tsx", "undefined", lexis.Keyword}, {"tsx", "object", lexis.Type}, {"typescript", "Guard", lexis.Type},
		{"javascript", "const", lexis.Keyword}, {"javascript", "=>", lexis.Punct}, {"javascript", "===", lexis.Operator}, {"javascript", `/^[A-Z]{2,4}-\d{3,}$/i`, lexis.String},
		{"javascript", "// cart.js - a shopping cart kept in localStorage, with a small event bus.", lexis.Comment},
	} {
		var kinds []lexis.Kind
		for _, tok := range drain(languages.Named(c.name, corpus[c.name][1])) {
			if tok.Text == c.text {
				kinds = append(kinds, tok.Kind)
			}
		}
		if !slices.Contains(kinds, c.kind) {
			t.Errorf("%s: %q is %v, want %v", c.name, c.text, kinds, c.kind)
		}
	}
}

// A caller's own language is found before one Lexis has for the same file,
// and rules a caller loads make a language like any other.
func TestACallersOwnLanguage(t *testing.T) {
	set := all()
	set.Add(lexis.Language{Files: []string{"*.go"}, Lexer: func(text string) lexis.Lexer {
		done := false
		return func() (lexis.Token, bool) {
			done = !done
			return lexis.Token{Text: text, Kind: lexis.Emph}, done
		}
	}})
	if got := drain(set.File("a.go", "package p")); len(got) != 1 || got[0].Kind != lexis.Emph {
		t.Errorf("a.go: %v", got)
	}
	ini, err := rules.Load(set, []byte(`<lexer><config><alias>conf</alias><filename>*.ini</filename></config><rules><state name="root">
		<rule pattern=";[^\n]*"><token type="Comment"/></rule>
		<rule pattern="(\w+)(=)"><bygroups><token type="NameTag"/><token type="Text"/></bygroups></rule>
		<rule pattern="[^\n;=]+|\n"><token type="Text"/></rule>
	</state></rules></lexer>`), nil)
	if err != nil {
		t.Fatal(err)
	}
	set.Add(ini)
	if set.Named("Conf", "") == nil {
		t.Error("conf: no lexer")
	}
	if got := drain(set.File("a.ini", "; c\nkey=v\n")); len(got) != 6 || got[0].Kind != lexis.Comment || got[2].Kind != lexis.Tag {
		t.Errorf("a.ini: %v", got)
	}
	for _, bad := range []string{`<rule pattern="a(?=b)"><token type="Text"/></rule>`, `<rule pattern="a"><push state="x" state="y"/></rule>`, `<rule func="none"/>`} {
		if _, err := rules.Load(nil, []byte(`<lexer><rules><state name="root">`+bad+`</state></rules></lexer>`), nil); err == nil {
			t.Errorf("%s loaded", bad)
		}
	}
}
