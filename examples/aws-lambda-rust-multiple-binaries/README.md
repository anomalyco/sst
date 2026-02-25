# SST Rust support

SST uses [cargo lambda](https://www.cargo-lambda.info/) to build rust binaries and deploy them on AWS Lambda. You MUST install [cargo](https://rustup.rs/) and [cargo lambda](https://www.cargo-lambda.info/guide/installation.html) and [zig](https://ziglang.org/download/) (a cargo-lambda dependency which is automatically installed with cargo-lambda in most installation methods) on your own before using sst rust lambdas.

## Setup with Github Actions

An example to deploy using github actions with `arm64` architecture, feel free to configure as needed

```yml
name: Deploy Prod

on:
  push:
    branches:
      - main

jobs:
  deploy-prod:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pguyot/arm-runner-action@v2

      - name: use Node.js
        uses: actions/setup-node@v3
        with:
          node-version: latest

      - name: use pnpm
        uses: pnpm/action-setup@v4
        with:
          version: latest

      - name: get pnpm store directory
        shell: bash
        run: |
          echo "STORE_PATH=$(pnpm store path --silent)" >> $GITHUB_ENV

      - name: setup pnpm cache
        uses: actions/cache@v3
        with:
          path: ${{ env.STORE_PATH }}
          key: ${{ runner.os }}-pnpm-store-${{ hashFiles('**/pnpm-lock.yaml') }}
          restore-keys: |
            ${{ runner.os }}-pnpm-store-

      - name: use Rust
        uses: actions-rs/toolchain@v1

      - name: use Rust cache
        uses: Swatinem/rust-cache@v2

      - name: use Zig
        uses: korandoru/setup-zig@v1
        with:
          zig-version: master

      - name: use Cargo Lambda
        uses: jaxxstorm/action-install-gh-release@v1.9.0
        with:
          repo: cargo-lambda/cargo-lambda
          platform: linux
          arch: aarch64 # | x86_64

      - name: pnpm install
        run: pnpm install --frozen-lockfile

      - name: sst install providers
        run: |
          set -euxo pipefail
          pnpm sst install

      - name: sst deploy
        run: |
          set -euxo pipefail
          pnpm sst deploy --stage prod

    env:
      STAGE: prod
      LOG_LEVEL: info
      AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
      AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
```
