/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS API Gateway V1 streaming
 *
 * An example on how to enable streaming for API Gateway REST API routes.
 *
 * ```ts title="sst.config.ts"
 * api.route("GET /", {
 *   handler: "index.handler",
 *   streaming: true,
 * });
 * ```
 *
 * The handler uses the native `awslambda.streamifyResponse` and
 * `awslambda.HttpResponseStream.from` to stream responses through API Gateway.
 *
 * ```ts title="index.ts"
 * export const handler = awslambda.streamifyResponse(
 *   async (event, stream) => {
 *     stream = awslambda.HttpResponseStream.from(stream, {
 *       statusCode: 200,
 *       headers: {
 *         "Content-Type": "text/plain; charset=UTF-8",
 *         "X-Content-Type-Options": "nosniff",
 *       },
 *     });
 *
 *     stream.write("Hello ");
 *     await new Promise((resolve) => setTimeout(resolve, 3000));
 *     stream.write("World");
 *
 *     stream.end();
 *   },
 * );
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
    api.route("GET /", {
      handler: "index.handler",
      streaming: true,
    });
    api.route("GET /hono", {
      handler: "hono.handler",
      streaming: true,
    });
    api.deploy();

    return {
      api: api.url,
    };
  },
});
