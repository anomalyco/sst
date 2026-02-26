/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS API Gateway V1 streaming
 *
 * An example on how to enable streaming for API Gateway REST API routes.
 *
 * ```ts title="sst.config.ts"
 * api.route("GET /", "index.handler", {
 *   streaming: true,
 * });
 * ```
 *
 * The handler uses the native `awslambda.streamifyResponse` and
 * `awslambda.HttpResponseStream.from` to stream responses through API Gateway.
 *
 * ```ts title="index.ts"
 * export const handler = awslambda.streamifyResponse(
 *   async (_event: unknown, responseStream: any) => {
 *     responseStream = awslambda.HttpResponseStream.from(responseStream, {
 *       statusCode: 200,
 *       headers: { "Content-Type": "text/plain" },
 *     });
 *
 *     responseStream.write("Hello");
 *     await new Promise((resolve) => setTimeout(resolve, 3000));
 *     responseStream.write(" World");
 *     responseStream.end();
 *   },
 * );
 * ```
 *
 * :::note
 * Streaming is currently not supported in `sst dev`.
 * :::
 *
 * To test this in your terminal, use the `curl` command with the `--no-buffer` option.
 *
 * ```bash "--no-buffer"
 * curl --no-buffer https://xxxxxxxxxx.execute-api.us-east-1.amazonaws.com/prod
 * ```
 *
 */
export default $config({
  app(input) {
    return {
      name: "aws-apigv1-stream",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const api = new sst.aws.ApiGatewayV1("MyApi");
    api.route("GET /", "index.handler", {
      streaming: true,
    });
    api.route("GET /hono", "hono.handler", {
      streaming: true,
    });
    api.deploy();

    return {
      api: api.url,
    };
  },
});
