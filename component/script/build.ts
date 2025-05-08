#!/bin/env bun

import { $ } from "bun";

await $`rm -rf ./dist`;
await $`mkdir -p ./dist/support`;

// simple copy
for (const item of [
  "python-runtime",
  "python-docker",
  "cf-static-site-router-worker",
  "cf-ssr-site-router-worker",
  "empty-function",
  "empty-site",
]) {
  await $`cp -a ./support/${item}/ ./dist/support/`;
}

// build typescript
await Promise.all(
  ["nodejs-runtime", "ssr-warmer", "vector-handler"].map(async (item) => {
    await Bun.build({
      entrypoints: [`./support/${item}/index.ts`],
      outdir: `./dist/support/${item}`,
      target: "node",
      format: "esm",
      banner: `
    import { createRequire as topLevelCreateRequire } from 'module';
    const require = topLevelCreateRequire(import.meta.url);
  `,
    });
  }),
);

await $`GOARCH=amd64 GOOS=linux go build -trimpath -mod=readonly -ldflags="-buildid=" -o ./dist/support/bridge/bootstrap ./support/bridge`;

await $`docker buildx create --name multi --driver docker-container --use`
  .cwd("./support/bridge-task")
  .nothrow();
await $`docker buildx build --platform linux/amd64,linux/arm64 -t ghcr.io/sst/sst/bridge-task:$(date +%Y%m%d%H%M%S) -t ghcr.io/sst/sst/bridge-task:latest ${
  process.env.DOCKER_PUSH ? "--push" : ""
} .`.cwd("./support/bridge-task");
