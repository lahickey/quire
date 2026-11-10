# dir = this module's directory. `just …` from inside quire/.
dir := source_directory()

port := "8091"

_default:
    @just --list

# Format Go and templ, and turn each tab in Markdown into two spaces.
fmt:
    @cd {{ dir }} && gofmt -w .
    @cd {{ dir }} && go tool templ fmt .
    @cd {{ dir }} && sed -i 's/\t/  /g' $(git ls-files -co --exclude-standard '*.md')

# Use the committed hooks in .githooks. Run once per clone.
hooks:
    @cd {{ dir }} && git config core.hooksPath .githooks

# Regenerated *_templ.go is committed so `go run` works without the templ tool.
generate:
    @cd {{ dir }} && go tool templ generate -path .

# gopls check prints diagnostics and exits 0; treat any output as failure. A tab in Markdown fails too; `just fmt` expands it.
check:
    @cd {{ dir }} && ! git grep --untracked -nP '\t' -- '*.md'
    @cd {{ dir }} && go vet ./...
    @cd {{ dir }} && go vet -tags quire_nobake $(go list -tags quire_nobake ./... | grep -v -e /examples -e /site)
    @cd {{ dir }} && go vet -tags release ./internal/bake
    @cd {{ dir }} && golangci-lint run --show-stats=false ./...
    @cd {{ dir }} && sh -c 'out=$(gopls check $(find . -name "*.go" ! -path "./vendor/*")); if [ -n "$out" ]; then printf "%s\n" "$out"; exit 1; fi'

test: generate check
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

# The documentation site: `just site` runs it, `just site dev` adds live reload.
mod site

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
