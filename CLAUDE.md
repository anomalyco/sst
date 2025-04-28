# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, Lint, and Test Commands

- **Build**: Go - `go build`, TypeScript - `bun run build` (in respective directories)
- **Test**: Go - `go test ./pkg/...`, TypeScript - `cd platform && bun run test`
- **Single test**: Go - `go test ./pkg/path/file_test.go`, TS - `cd platform && bun run test path/to/test.test.ts`
- **Type check**: `cd platform && bun run dev` (watches for changes)
- **Generate docs**: `cd www && bun run generate`

## Code Style Guidelines

### Go
- Standard Go formatting
- Error handling with `if err != nil` pattern
- Packages use lowercase names
- CamelCase for functions, ALL_CAPS for constants
- Tests in separate `_test` packages

### TypeScript
- Strict typing with TypeScript
- camelCase for variables and functions, PascalCase for classes
- ES modules with explicit imports/exports
- Concise JSDoc comments for complex types
- Error handling with proper type guards

## Project Structure
- Monorepo with Go and TypeScript code
- Main directories: `cmd/`, `pkg/`, `platform/`, `sdk/`, `www/`
- Tests placed alongside implementation files
- Examples in the `examples/` directory

Run linting and type checking before submitting changes. Follow existing patterns when adding new code.