# Distribution plan: `@guide/sst` fork

## Goal

Distribute this SST fork under the `@guide/sst` package name on JSR (TS pieces) and as the `sst` CLI binary on GitHub Releases, with versions suffixed `-guide.N` so they track upstream while carrying our own patches.

## Non-goals

- No PyPI package (Python SDK)
- No crates.io package (Rust SDK)
- No AUR package
- No Homebrew tap (for now)
- No Discord webhook
- No per-OS npm `optionalDependencies` matrix
- No `sst add` / install / setup wizard for npm (the curl installer is the only install path)

## What ships

| Artifact | Channel | Source |
|---|---|---|
| `sst` CLI binary (5 OS/arch) | GitHub Releases | `cmd/sst/` via goreleaser |
| `bridge-task` Docker image | `ghcr.io/guidefari/sst/bridge-task` | `platform/support/bridge-task/` via `platform/scripts/build` |
| `@guide/sst` (TS SDK) | JSR | `sdk/js/` |
| `@guide/platform` (TS Pulumi components) | JSR | `platform/` |

The bridge-task image is **required for `sst dev`** (mosaic mode). The CLI binary embeds the platform via `//go:embed` (`platform/platform.go:16`), so the platform is also implicitly shipped inside the CLI. The separate JSR publish of `@guide/platform` is for type discoverability and re-use in user code.

## Versioning

- Tags: `v3.5.10-guide.1`, `v3.5.10-guide.2`, `v3.5.11-guide.1`
- First sync onto a new upstream base → `-guide.1`
- Subsequent fork-only patches increment `.N`
- CLI version injected via `main.version` ldflag (already wired in `package.json:12`)
- JSR accepts the full string in `jsr.json` `version`
- Manual tagging for now; `scripts/release` not updated to handle `-guide.N` math

## Edits

### `.github/workflows/release.yml`

- **Keep**: checkout, bun, node, go setup, QEMU, docker-buildx, GHCR login (repointed), `go test`, platform build (with `DOCKER_PUSH: true`), goreleaser release, JS SDK release
- **Remove**:
  - `:72-74` — `astral-sh/setup-uv` step (Python)
  - `:95-106` — Rust SDK release
  - `:108-118` — Python SDK release
  - `:120-134` — Discord announcement
- **Repoint**: `goreleaser` env `GITHUB_TOKEN` → `SST_GITHUB_TOKEN` (existing) is fine; remove `AUR_KEY` env since AUR is gone
- **Resulting shape**: ~50 lines, down from 135

### `.goreleaser.yml`

- **Keep**: `builds` (with `CGO_ENABLED=0`, `goos`/`goarch` matrix, `main: ./cmd/sst`), `archives`, `checksum`, `changelog`
- **Remove**: `aurs`, `brews`, `nfpms` blocks
- **Update**: `project_name: sst` (stays — CLI name is `sst`)

### `install`

- `:42` — `sst/sst/releases/latest` → `guidefari/sst/releases/latest`
- `:43,50` — `api.github.com/repos/sst/sst/...` → `.../repos/guidefari/sst/...`
- `:3` — `APP=sst` stays
- Otherwise unchanged (PATH manipulation, version check, shell detection all stay)

### `sdk/js/package.json`

- `:3` — `"name": "@guide/sst"`
- `:8-9` — `repository.url` → `git+https://github.com/guidefari/sst.git`
- Keep `main`, `exports`, `bin`, `files` as-is

### `sdk/js/scripts/release.ts`

- **Remove**: the `cpus` map, `tmp` dir, `binaryPackages` loop (lines 23-62), per-OS `optionalDependencies` writes, `npm publish --tag ${tag}` for binary packages
- **Replace**: `npm publish` at the end with `npx jsr publish`
- **Add**: read `version` from `dist/metadata.json` (goreleaser-generated) and pass to `jsr publish --set-version`
- New shape (~10 lines):

  ```ts
  import metafile from "../../../dist/metadata.json";
  import pkg from "../package.json";

  const next = { ...pkg, version: metafile.version };
  await $`bun run build`;
  await $`npx jsr publish --set-version ${next.version}`;
  ```

### `sdk/js/jsr.json` (new)

```json
{
  "name": "@guide/sst",
  "version": "0.0.0",
  "exports": "./src/index.ts",
  "publish": { "include": ["src/**", "README.md", "package.json"] }
}
```

JSR reads `package.json#exports` if `jsr.json#exports` is omitted; the explicit `src/index.ts` is a fallback for consumers who want raw TS.

### `platform/package.json`

- `:3` — `"name": "@guide/platform"`

### `platform/jsr.json` (new, optional)

```json
{
  "name": "@guide/platform",
  "version": "0.0.0",
  "exports": "./src/index.ts",
  "publish": { "include": ["src/**", "README.md", "package.json"] }
}
```

Add a `publish:platform` step to the release workflow if you want this published.

### `platform/scripts/build`

- `:38-44` — repoint `ghcr.io/anomalyco/sst/...` → `ghcr.io/guidefari/sst/...`
- Keep everything else (the bridge-task build is still required for `sst dev`)

### `sdk/python/`, `sdk/rust/`

- **Delete both directories** — not published, not referenced after the workflow edits above

### `scripts/release` (deferred)

The bash version bumper reads the latest tag and increments the patch. It does not understand `-guide.N` yet. Two options:

1. **Manual tagging** (recommended for now): `git tag v3.5.10-guide.1 && git push --tags`
2. **Update `scripts/release`** to detect `guide.N` and either increment `.N` (if base version unchanged) or reset to `guide.1` (if base version changed)

Pick option 1 for the first cut. Option 2 is a 10-line sed.

## New files

- `sdk/js/jsr.json`
- `platform/jsr.json` (optional, recommended)

## Verification

1. Build locally: `bun run build:platform && bun run build:cli:semver`
2. Tag: `git tag v0.0.0-guide.1 && git push --tags`
3. Confirm GitHub Actions release workflow passes and produces:
   - 5 binary tarballs on the release page
   - `bridge-task` image on `ghcr.io/guidefari/sst/bridge-task:v0.0.0-guide.1`
   - JSR page shows `@guide/sst` at `0.0.0-guide.1`
4. Install in a test dir: `curl -fsSL https://raw.githubusercontent.com/guidefari/sst/dev/install | sh`
5. Smoke test: `sst version` prints `0.0.0-guide.1`; `sst dev` boots the bridge-task container
6. JSR consumer test: in a throwaway project, `npx jsr add @guide/sst` and `import { Resource } from "@guide/sst"` resolves

## Open questions for review

1. **Platform on JSR**: keep or skip? Pro = types discoverable, `import "@guide/platform"` works in user `sst.config.ts`. Con = extra publish step, larger surface.
2. **Manual vs automated `-guide.N` bumping**: start manual?
3. **CHANGELOG**: do we want a fork-specific changelog (commits past last upstream sync) in the release body, or rely on goreleaser's auto-generated one?
4. **`sst version` output**: should the CLI distinguish itself (e.g., `sst version` prints `sst 3.5.10-guide.1 (upstream: sst/sst@3.5.10)`) so users can see what they're on?
5. **Homebrew tap**: revisit when the first real fork-only patch lands.
6. **The local changes that prompted this** (`.gitignore` `.bookgen-notes`, `build:cli:semver` script): unrelated to distribution — keep as-is.

## Out of scope (defer)

- Auto-update subcommand in the CLI
- Homebrew tap
- AUR
- Discord announce
- Upstream-sync automation (the `sst/sst` watcher)
- `sst upgrade` command
