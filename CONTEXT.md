# SST Ion Development Context

## Build Commands
- **Build platform**: `cd platform && bun run build`
- **Build SDK**: `cd sdk/js && bun run build` or `cd sdk/js && tsc`
- **Build Go CLI**: `go build -o bin/sst ./cmd/sst`
- **Test Go**: `go test ./...`
- **Test Go single file**: `go test ./cmd/sst/mosaic/ui/error_test.go`
- **Test TypeScript**: `cd platform && bun run test` (uses vitest)
- **Test single TS file**: `cd platform && bun run test path/to/test.test.ts`
- **Type check**: `cd platform && bun run dev` (tsc --watch --noEmit)

## Code Style Guidelines

### Go
- Use Go 1.23+ conventions with standard formatting (`gofmt`)
- Error handling with `if err != nil` pattern
- Package names: lowercase, single words
- Functions: PascalCase for exported, camelCase for unexported
- Constants: ALL_CAPS or PascalCase for exported
- Group imports: stdlib, external, internal
- Tests alongside implementation files with `_test.go` suffix

### TypeScript
- Use ESNext modules with Bundler moduleResolution
- Strict typing, avoid `any`
- camelCase for variables/functions, PascalCase for classes/types
- ES modules with explicit imports/exports
- Prefer async/await over Promises
- Use Prettier for formatting
- Group imports by source (stdlib, external, internal)

### Project Structure
- Monorepo with Go CLI (`cmd/`, `pkg/`) and TypeScript platform (`platform/`, `sdk/js/`)
- Examples in `examples/` directory
- Tests placed alongside implementation files
- Uses Bun for TypeScript builds and package management