SST is a framework for building full-stack apps on your own infrastructure. It uses Pulumi under the hood to deploy to AWS, Cloudflare, and other providers.


## Commands

- **Setup**: `bun install && go mod tidy && cd platform && bun run build`
- **Run CLI locally**: `cd examples/<app> && go run ../../cmd/sst <command>`
- **Go tests**: `go test ./...`
- **Build platform**: `cd platform && bun run build`
- **Generate docs**: `cd www && bun run generate`
- **Run docs locally**: `cd www && bun run dev`

## Codebase

- `cmd/sst/` — Go CLI entry, orchestrates everything. Commands as tree in `main.go`
- `cmd/sst/mosaic/` — live dev TUI, Lambda stubs forward invocations to local runtimes
- `pkg/server/` — JSON-RPC bridge, Go side (`rpc/rpc.ts` ↔ `pkg/server`)
- `pkg/bus/` — pub/sub connecting watcher, deployer, runtimes, UI
- `platform/` — TS Pulumi components embedded into Go binary via `//go:embed`
- `sdk/js/` — runtime SDK for reading linked resources
- `internal/` — shared Go utilities
- `examples/` — sample apps (useful for testing CLI locally)
- `www/` — docs site

## Notes

- This repo was renamed from `sst/sst` to `anomalyco/sst`
- When modifying SST components, verify changes by deploying an existing relevant example
- Always build the platform before deploying an example
- Docs are auto-generated from JSDoc comments in platform and extracted from the Go CLI
