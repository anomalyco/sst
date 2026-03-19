/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Hono streaming
 *
 * An example on how to enable streaming for Lambda functions using Hono.
 *
 * ```ts title="sst.config.ts"
 * {
 *   streaming: true
 * }
 * ```
 *
 * ```ts title="index.ts"
 * export const handler = streamHandle(app);
 * ```
 *
 * To test this in your terminal, use the `curl` command with the `--no-buffer` option.
 *
 * ```bash "--no-buffer"
 * curl --no-buffer https://u3dyblk457ghskwbmzrbylpxoi0ayrbb.lambda-url.us-east-1.on.aws
 * ```
 *
 * Streaming is also supported through API Gateway REST API.
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
      streaming: true,
      timeout: "15 minutes",
      handler: "index.handler",
    });
    return {
      api: hono.url,
    };
  },
});
