# AGENTS.md

## specific instructions

### Project overview

tRPC-Agent-Go is a Go multi-module monorepo (library/framework) for building AI agent systems. It is **not** a standalone application — there is no single `main.go` to run. The root module path is `trpc.group/trpc-go/trpc-agent-go`.

### Remote

- `origin` → `https://github.com/cyl6/trpc-agent-go.git` 
- Current branch: `main`

### Common commands

| Task | Command | Notes |
|------|---------|-------|
| Build | `go build ./...` | Root module only |
| Unit tests | `go test ./...` | Root module; all tests use mocks, no API keys needed |
| E2E tests | `cd test && go test ./...` | Separate module in `test/` |
| Lint | `golangci-lint run --timeout=10m` | Config in `.golangci.yml` |
| gofmt check | `gofmt -r 'interface{} -> any' -l .` | CI enforces `any` over `interface{}` |
| goimports check | `goimports -l .` | |
| All sub-module tests (CI-style) | `bash .github/scripts/run-go-tests.sh` | Runs tests across ~80 modules excluding examples/docs/test |
| Check example builds | `bash .github/scripts/check-examples.sh` | |
| Trace eval example (no API key) | `cd examples/evaluation/trace && go run .` | Trace mode skips LLM calls; verified runnable without `OPENAI_API_KEY` |

### Non-obvious caveats

- **`/home/cyl/.local/bin` and `$(go env GOPATH)/bin` must be on PATH** for `golangci-lint` and `goimports` to be found. The install block in "Local environment setup" already exports both. If a tool is missing in a new shell, run `source ~/.bashrc`.
- **No external API keys needed for tests.** The entire test suite uses mocks. API keys (e.g. `OPENAI_API_KEY`) are only needed to run the examples under `examples/`.
- **Multi-module monorepo:** There are ~80 `go.mod` files. Running `go test ./...` from the repo root only tests the root module. To test all modules, use the CI script `.github/scripts/run-go-tests.sh`.
- **SQLite CGO dependency:** The root module depends on `github.com/mattn/go-sqlite3`, which requires CGO. Ensure `CGO_ENABLED=1` (the default) and a C compiler is available.
- **License headers required on all `.go` files.** CI checks that every Go file has the Tencent Apache 2.0 header. See `CONTRIBUTING.md` for the template.
