# 02 — CLI framework + `version`

## Description
Wire in Cobra so we have a real subcommand tree. Prove the wiring with a `version` subcommand — the smallest possible smoke test that both the framework and `main` are hooked up.

## Status
done

## Depends on
[01 — Project scaffold](01-project-scaffold.md)

## What / where to change

Add the dependency:
```
go get github.com/spf13/cobra@latest
```

Files to create/modify:

- `cmd/wut/root.go` — `NewRootCmd() *cobra.Command` returning a root command with `Use: "wut"`, short description matching the spec §1 one-liner, and no default action (print help on bare invocation).
- `cmd/wut/version.go` — `NewVersionCmd()` returning a `version` subcommand that prints a package-level `Version = "0.0.0-dev"` constant.
- `cmd/wut/main.go` — replace body with: build root, register `version`, `Execute()`. On error, exit 1.

The `Version` const should live at the top of `root.go` (or a new `version.go` constant file) so step 10+ can bump it from `goreleaser` metadata later.

## How to verify

```
go build ./cmd/wut
./wut                # prints help
./wut --help         # prints help
./wut version        # prints: 0.0.0-dev
./wut version; echo $?   # exit 0
./wut bogus          # exits non-zero with Cobra "unknown command" error
```

`go vet ./...` still clean.
