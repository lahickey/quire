# dir = this module's directory. `just …` from inside quire/.
dir := source_directory()

port := "8091"

_default:
    @just --list

# Regenerated *_templ.go is committed so `go run` works without the templ tool.
generate:
    @cd {{ dir }} && go tool templ generate -path .

# gopls check prints diagnostics and exits 0; treat any output as failure.
check:
    @cd {{ dir }} && go vet ./...
    @cd {{ dir }} && go vet -tags quire_nobake $(go list ./... | grep -v -e /examples -e /site)
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
    #!/usr/bin/env bash
    set -euo pipefail
    cd {{ dir }}
    v={{ quote(version) }}
    die() { echo "release: $*" >&2; exit 1; }
    [[ $v =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || die "usage: just release vMAJOR.MINOR.PATCH"
    git rev-parse -q --verify "refs/tags/$v" >/dev/null && die "$v is already a tag"
    last=$(git tag -l 'v*' --sort=-v:refname | head -n 1)
    [[ -z $last || $(printf '%s\n%s\n' "$last" "$v" | sort -V | tail -n 1) == "$v" ]] || die "$v does not come after $last"
    [[ $(git branch --show-current) == master ]] || die "not on master"
    [[ -z $(git status --porcelain) ]] || die "the tree has uncommitted changes"
    grep -Eq '^[[:space:]]*replace([[:space:]]|\()' go.mod && die "go.mod has a replace"
    go mod tidy -diff >/dev/null || die "go.mod or go.sum is not tidy: run go mod tidy"
    command -v govulncheck >/dev/null || die "govulncheck is not installed: go install golang.org/x/vuln/cmd/govulncheck@latest"
    just release-check
    govulncheck ./...
    just test
    [[ -z $(git status --porcelain) ]] || die "the tests changed the tree: commit what they generated"
    git tag -a "$v" -m "$v"
    echo "tagged $v; push it, then fetch it through the proxy so pkg.go.dev has it:"
    echo "  git push public master $v"
    echo "  GOPROXY=https://proxy.golang.org go list -m $(go list -m)@$v"

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

# Run Quire's documentation site on the example port.
[continue]
[no-exit-message]
site: generate
    cd {{ dir }}/site && go run . --addr :{{ port }}

# List what examples exist.
examples:
    @cd {{ dir }}/examples && for d in */main.go; do dirname "$d"; done

# Kill whatever is still listening on the example port after a bad exit.
free:
    -fuser -k {{ port }}/tcp

# Update every dependency to its latest compatible version: Go modules and tools, the fonts, Datastar 1.x. Review the diff.
deps:
    #!/usr/bin/env bash
    set -euo pipefail
    cd {{ dir }}
    go get -u ./...
    go get tool
    go mod tidy
    go run ./internal/tools/fonts
    v=$(curl -sSf 'https://data.jsdelivr.com/v1/packages/gh/starfederation/datastar/resolved?specifier=1' | jq -r .version)
    js=$(curl -sSf "https://cdn.jsdelivr.net/gh/starfederation/datastar@v$v/bundles/datastar.js")
    printf '/*! Datastar v%s | MIT License | Copyright (c) Star Federation | https://github.com/starfederation/datastar */\n%s\n' "$v" "$js" > internal/assets/datastar.js
