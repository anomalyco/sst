# SST Development Context

## Build, Lint, and Test Commands
- **Build**: Go - `go build -o bin/sst ./cmd/sst`, TypeScript - `cd platform && bun run build`
- **Test**: Go - `go test ./...`, TypeScript - `cd platform && bun run test`
- **Single test**: Go - `go test ./cmd/sst/mosaic/ui/error_test.go`, TS - `cd platform && bun run test path/to/test.test.ts`
- **Type check**: `cd platform && bun run dev` (watches for changes)
- **Local development**: `cd examples/aws-api && go run ../../cmd/sst <command>`

## Code Style Guidelines

### Go (1.23+)
- Standard Go formatting with `gofmt`
- Error handling with `if err != nil` pattern
- Packages use lowercase names, CamelCase for exported functions
- Group imports: stdlib, external, internal
- Tests in `_test.go` files alongside implementation

### TypeScript
- ESNext modules with Bundler moduleResolution
- Strict typing, avoid `any`, use explicit types
- camelCase for variables/functions, PascalCase for classes
- Group imports by source (stdlib, external, internal)
- Prefer async/await over Promises
- Use Prettier for formatting (configured in package.json)

## Project Structure
- Monorepo: Go CLI in `cmd/`, TypeScript platform in `platform/`, examples in `examples/`
- Main entry: `cmd/sst/main.go`
- Platform components: `platform/src/components/`
- Tests alongside implementation files
- Uses Bun as package manager, Pulumi for infrastructure