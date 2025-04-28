# SST Ion Development Guide

## Build Commands
- Build platform: `cd platform && npm run build`
- Build SDK: `cd sdk/js && npm run build`
- Run tests: `cd platform && npm test`
- Run single test: `cd platform && npx vitest run <test-file-path>`
- Go build: `go build -o bin/sst ./cmd/sst`
- Go test: `go test ./...`
- Go test single file: `go test ./cmd/sst/mosaic/ui/error_test.go`

## Code Style Guidelines
- TypeScript: Use ESNext modules with Bundler moduleResolution
- Go: Follow Go 1.23+ conventions with standard formatting
- Imports: Group imports by source (stdlib, external, internal)
- Naming: camelCase for JS/TS, snake_case for Go files, PascalCase for exported Go functions
- Types: Always use explicit types in TypeScript, avoid `any`
- Error handling: Use structured error types, propagate errors with context
- Formatting: Use Prettier for JS/TS, gofmt for Go
- Prefer async/await over Promises in JS/TS
- Use functional patterns where appropriate