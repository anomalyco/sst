/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Hono streaming
 *
 * An example on how to enable streaming for Lambda functions using Hono.
 *
 * While `sst dev` doesn't support streaming, we can conditionally enable it on deploy.
 *
 * ```ts title="sst.config.ts"
 * {
 *   streaming: $dev ? false : true
 * }
 * ```
 *
 * ```ts title="index.ts"
 * export const handler = process.env.SST_LIVE ? handle(app) : streamHandle(app);
 * ```
 *
 * This will return the standard handler for `sst dev`.
 *
 * :::note
 * Streaming is currently not supported in `sst dev`.
 * :::
 *
 * To test this in your terminal, use the `curl` command with the `--no-buffer` option.
 *
 * ```bash "--no-buffer"
 * curl --no-buffer https://u3dyblk457ghskwbmzrbylpxoi0ayrbb.lambda-url.us-east-1.on.aws
 * ```
 *
 * Here we are using a Function URL directly because API Gateway doesn't support streaming.
 *
 */
export default $config({
  app(input) {
    return {
      name: "aws-hono-stream",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const hono = new sst.aws.Function("Hono", {
      url: true,
      streaming: $dev ? false : true,
      timeout: "15 minutes",
      handler: "index.handler",
    });
    return {
      api: hono.url,
    };
  },
});
