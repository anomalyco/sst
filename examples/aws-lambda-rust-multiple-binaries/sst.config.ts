/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lamda Rust multiple-binaries
 *
 * This example shows how to deploy multiple binary rust project to AWS Lambda.
 *
 * SST relies on the work of [cargo lambda](https://cargo-lambda) to build and deploy Rust Lambda functions.
 *
 * What is special about the following file is that we are defining multiple binaries using the `[[bin]]` section in the `Cargo.toml` file.
 *
 * ```toml title="Cargo.toml" {13,14,15,17,18,19}
 * [package]
 * name = "aws-lambda-rust-multi-bin"
 * version = "0.1.0"
 * edition = "2021"
 *
 * [dependencies]
 * lambda_runtime = "0.13.0"
 * serde = { version = "1.0.217", features = ["derive"] }
 * serde_json = "1.0.138"
 * tokio = { version = "1", features = ["macros"] }
 * # -- please note ommited dependencies --
 *
 * [[bin]]
 * name = "push"
 * path = "src/push.rs"
 *
 * [[bin]]
 * name = "pop"
 * path = "src/pop.rs"
 * ```
 *
 * We then utilise the . syntax to specify the handler binary
 *
 * ```ts title="sst.config.ts" {5,11}
 * new sst.aws.Function("push", {
 *   url: true,
 *   runtime: "rust",
 *   link: [bucket],
 *   handler: "./.push",
 * });
 * new sst.aws.Function("pop", {
 *   url: true,
 *   runtime: "rust",
 *   link: [bucket],
 *   handler: "./.pop",
 * });
 * ```
 */

export default $config({
  app(input) {
    return {
      name: "aws-rust-lambda",
      removal: input?.stage === "production" ? "retain" : "remove",
      protect: ["production"].includes(input?.stage),
      home: "aws",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("Bucket");
    const push = new sst.aws.Function("push", {
      runtime: "rust",
      handler: "./.push",
      url: true,
      architecture: "arm64",
      link: [bucket],
    });
    const pop = new sst.aws.Function("pop", {
      runtime: "rust",
      handler: "./.pop",
      url: true,
      architecture: "arm64",
      link: [bucket],
    });

    return { push_url: push.url, pop_url: pop.url };
  },
});
