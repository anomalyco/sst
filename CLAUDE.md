## Layout

- `platform/` — TS Pulumi components embedded via `//go:embed` (`platform/platform.go:16`)
- `examples/` — sample apps
- `cmd/sst/` — Go CLI entry, orchestrates everything
- `cmd/sst/mosaic/` — live dev TUI
- `pkg/server/` — JSON-RPC bridge for custom Pulumi dynamic providers (`rpc/rpc.ts` ↔ `pkg/server`)
- `pkg/bus/` — pub/sub event bus
- `sdk/js/` — runtime SDK for reading linked resources
- `www/` — docs site (auto-generated from JSDoc comments in platform and extracted from the Go CLI)

## Commands

- **Setup**: `bun install && go mod tidy`
- **Build platform**: `cd platform && bun run build`
- **Run CLI locally**: `cd ../examples/<app> && go run ../../cmd/sst <command>`
- **Go tests**: `go test ./...`
- **Generate docs**: `cd www && bun run generate`
- **Run docs locally**: `cd www && bun run dev`

## Verification

1. Build the platform
2. `cd examples/<app> && bun install`
3. `go run ../../cmd/sst deploy`
4. Verify with `curl` or AWS CLI
5. Don't clean up unless told to
