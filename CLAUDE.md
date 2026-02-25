SST is a framework for building full-stack apps on your own infrastructure. It uses Pulumi under the hood to deploy to AWS, Cloudflare, and other providers with a high-level component model, resource linking, and a live dev mode that runs Lambda functions locally.


## Commands

- **Setup**: `bun install && go mod tidy && cd platform && bun run build`
- **Run CLI locally**: `cd examples/<app> && go run ../../cmd/sst <command>`
- **Build CLI binary**: `go build ./cmd/sst` (useful when testing things outside of the repo)
- **Go tests**: `go test ./...`
- **Type check**: `cd platform && bun tsc --noEmit`
- **Build platform**: `cd platform && bun run build` (runs `scripts/build` — bundles workers, runtime, bridge binary, types)
- **Generate docs**: `cd www && bun run generate`
- **Run docs locally**: `cd www && bun run dev`
- **Deploy example**: `cd examples/<app> && go run ../../cmd/sst deploy`

## Codebase

- `cmd/sst/` — Go CLI entry, orchestrates everything. Commands as tree in `main.go`
- `cmd/sst/mosaic/` — live dev TUI, Lambda stubs forward invocations to local runtimes
- `pkg/server/` — JSON-RPC bridge, Go side (`rpc/rpc.ts` ↔ `pkg/server`)
- `pkg/runtime/` — runtimes each implement `Match()`, `Build()`, `Run()`, `ShouldRebuild()`
- `pkg/project/provider/` — pluggable state backend + encrypted secrets
- `pkg/bus/` — pub/sub connecting watcher, deployer, runtimes, UI
- `platform/` — TS Pulumi components by provider (`aws/`, `cloudflare/`, `vercel/`), embedded into Go binary via `//go:embed`
- `sdk/js/` — runtime SDK for reading linked resources
- `internal/` — shared Go utilities
- `examples/` — sample apps (useful for testing CLI locally)
- `www/` — docs site

## Notes

- This repo was renamed from `sst/sst` to `anomalyco/sst`
- When modifying SST components, verify changes by deploying a relevant example
- Always build the platform before running `go run ../../cmd/sst <command>`
- Docs are auto-generated from JSDoc comments in platform and extracted from the Go CLI
