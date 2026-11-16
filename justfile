# dir = this module's directory. `just …` from inside quire/.
dir := source_directory()

port := "8091"

# The corpus is other people's kinds of file, for Lexis to read: no formatter or check of ours touches it.
corpus := "libs/lexis/_corpus"
# The languages page shows the corpus's files as they are, tabs and all.
shown := "site/content/docs/languages.md"
gofiles := 'find . -name "*.go" ! -path "./vendor/*" ! -path "./' + corpus + '/*"'

_default:
    @just --list

# Format Go and templ, and turn each tab in Markdown into two spaces.
fmt:
    @cd {{ dir }} && gofmt -w $({{ gofiles }})
    @cd {{ dir }} && go tool templ fmt .
    @cd {{ dir }} && sed -i --follow-symlinks 's/\t/  /g' $(git ls-files '*.md' ':!{{ corpus }}' ':!{{ shown }}')

# Use the committed hooks in .githooks. Run once per clone.
hooks:
    @cd {{ dir }} && git config core.hooksPath .githooks

# Regenerated *_templ.go is committed so `go run` works without the templ tool.
generate:
    @cd {{ dir }} && go tool templ generate -path .

# gopls check prints diagnostics and exits 0; treat any output as failure, hints included, as an editor shows them. A tab in Markdown fails too; `just fmt` expands it.
check:
    @cd {{ dir }} && ! git grep -nP '\t' -- '*.md' ':!{{ corpus }}' ':!{{ shown }}'
    @cd {{ dir }} && go vet ./...
    @cd {{ dir }} && go vet -tags quire_nobake $(go list -tags quire_nobake ./... | grep -v -e /examples -e /site)
    @cd {{ dir }} && go vet -tags release ./internal/bake
    @cd {{ dir }} && golangci-lint run --show-stats=false ./...
    @cd {{ dir }} && sh -c 'out=$(gopls check -severity=hint $({{ gofiles }})); if [ -n "$out" ]; then printf "%s\n" "$out"; exit 1; fi'

test: generate check release-check
    @cd {{ dir }} && go test ./...

# Before a release: fails on any converter whose code or dependencies changed since its version was set.
release-check:
    @cd {{ dir }} && go test -tags release -run Fingerprints -count=1 ./internal/bake

# Check everything, then tag HEAD on master. Prints the push; never pushes.
release version:
    @cd {{ dir }} && go run ./internal/tools/release {{ quote(version) }}

# Run an example: `just example` for 1-minimal, or `just example <name>`.
[continue]
[no-exit-message]
example name="1-minimal": generate
    cd {{ dir }}/examples/{{ name }} && go run . --addr :{{ port }}

# Same, with live reload. Needs `wgo` on PATH (not `go tool wgo` — that
# wrapper double-fires SIGINT and orphans the server). Install:
#   go install github.com/bokwoon95/wgo@latest
[continue]
[no-exit-message]
dev name="1-minimal": generate
    cd {{ dir }} && wgo \
    	-cd {{ dir }}/examples/{{ name }} \
    	-xfile '_templ\.go$' \
    	-file '\.go$' -file '\.templ$' -file '\.css$' -file '\.md$' -file '\.svg$' -file '\.js$' -file '\.yaml$' \
    	go tool templ generate -path {{ dir }} \
    	:: go run . --dev --drafts --addr :{{ port }}

# The documentation site: `just site` lists its recipes, `just site run` runs it, `just site dev` adds live reload.
mod site

# Lexis, the highlighter: `just lexis` lists its recipes.
mod lexis 'libs/lexis'

# List what examples exist.
examples:
    @cd {{ dir }}/examples && for d in */main.go; do dirname "$d"; done

# Kill whatever is still listening on the example port after a bad exit.
free:
    -fuser -k {{ port }}/tcp

# Update every dependency to its latest compatible version: Go modules and tools, the fonts, Datastar 1.x. Review the diff.
deps:
    @cd {{ dir }} && go get -u ./...
    @cd {{ dir }} && go get tool
    @cd {{ dir }} && go mod tidy
    @cd {{ dir }} && go run ./internal/tools/thirdparty
