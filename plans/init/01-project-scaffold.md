# 01 — Project scaffold

## Description
Bootstrap the Go module and the directory layout the rest of the steps assume. No behavior yet — just a runnable binary that prints its own name.

## Status
done

## Depends on
Nothing.

## What / where to change

Create at project root:

- `go.mod` — `go mod init wut` (module path `wut`; we'll rename if we publish).
- `.gitignore` — ignore the compiled binary, `dist/`, editor files.
- `cmd/wut/main.go` — `package main` with `func main()` that prints `wut` and exits 0. This is a placeholder; step 02 replaces the body with a Cobra executor.
- Empty package directories (each with a `doc.go` one-liner so Go treats them as packages):
  - `internal/detect/`
  - `internal/config/`
  - `internal/harness/`
  - `internal/shell/`

No dependencies added yet.

## How to verify

```
go build ./cmd/wut
./wut
```

Expected: prints `wut` and exits 0. `go vet ./...` clean. The four internal package dirs exist and each contains `doc.go`.
