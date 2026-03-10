/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda streaming
 *
 * An example on how to enable streaming for Lambda functions.
 *
 * ```ts title="sst.config.ts"
 * {
 *   streaming: true
 * }
 * ```
 *
 * Use the `awslambda.streamifyResponse` function to wrap your handler. The `awslambda`
 * global is provided by the Lambda execution environment at runtime, and SST provides it
 * automatically during `sst dev` as well. For TypeScript types, importing from
 * `@types/aws-lambda` will augment the global namespace.
 *
 * ```ts title="index.ts"
 * import { APIGatewayProxyEventV2 } from "aws-lambda";
 *
 * export const handler = awslambda.streamifyResponse(myHandler);
 *
 * async function myHandler(
 *   _event: APIGatewayProxyEventV2,
 *   responseStream: awslambda.HttpResponseStream
 * ): Promise<void> {
 *   return new Promise((resolve, _reject) => {
 *     responseStream.setContentType('text/plain')
 *     responseStream.write('Hello')
 *     setTimeout(() => {
 *       responseStream.write(' World')
 *       responseStream.end()
 *       resolve()
 *     }, 3000)
 *   })
 * }
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
      name: "aws-lambda-stream",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const fn = new sst.aws.Function("MyFunction", {
      url: true,
      streaming: true,
      timeout: "15 minutes",
      handler: "index.handler",
    });

    return {
      url: fn.url,
    };
  },
});
