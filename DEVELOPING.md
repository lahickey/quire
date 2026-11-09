# Developing Quire

## Tools

- Go 1.27 or later. templ runs as a Go tool from `go.mod`, so it needs no install.
- [just](https://just.systems) runs every task below.
- [golangci-lint](https://golangci-lint.run) and gopls, for `just check`.
- [wgo](https://github.com/bokwoon95/wgo), for `just dev`: `go install github.com/bokwoon95/wgo@latest`. Not `go tool wgo`, which double-fires SIGINT and orphans the server.
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck), for `just release`.
- curl and jq, for `just deps`; `fuser`, for `just free`.

## Running

`just example [name]` runs an example on :8091, and `just examples` lists them.
`just dev [name]` adds live reload and drafts; generated `*_templ.go` is not a watch trigger.
`just site` runs the documentation site.
`just free` frees :8091 after a bad exit.

## Testing

`just test` regenerates templ, runs vet, golangci-lint and gopls, then the tests.
`examples/examples_test.go` builds, bakes and fetches every example, both as it runs in dev and as it ships; `-short` skips it.
Generated `*_templ.go` is committed, so `go run` works without templ.

`examples/` is the check that the easy path stays easy: if an example grows boilerplate, that boilerplate belongs in Quire.

## Dependencies

`just deps` updates every dependency to its latest compatible version: Go modules and tools, the fonts, and Datastar within 1.x. Review the diff before committing.

## Releasing

`just release-check` names each converter (markdown, diagram, card, picture) whose code or dependencies changed since the table in `internal/bake/versions.go` was last set.
Bump the version of any whose output changed, which rebuilds exactly those artifacts in every site's next bake, and paste the fingerprints it prints.

`just release vX.Y.Z` runs every check and the tests, then tags `HEAD` on master. It prints the push rather than doing it.
