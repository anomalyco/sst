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

- **Setup**: `bun run setup`
- **Build platform**: `bun run build:platform`
- **Build CLI**: `bun run build:cli`
- **Run CLI locally**: `cd examples/<app> && go run ../../cmd/sst <command>`
- **Go tests**: `bun run test:cli`
- **Typecheck**: `bun run typecheck`
- **Generate docs**: `bun run docs:generate`
- **Run docs locally**: `bun run docs:dev`

## Verification

1. Build the platform
2. `cd examples/<app> && bun install`
3. `go run ../../cmd/sst deploy`
4. Verify with `curl` or AWS CLI
5. Don't clean up unless told to
6. `sst dev --mode=basic` for dev mode
