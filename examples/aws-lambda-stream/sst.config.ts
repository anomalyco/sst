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
 * While `sst dev` doesn't support streaming, you can use the
 * [`lambda-stream`](https://github.com/astuyve/lambda-stream) package to test locally.
 *
 * ```bash
 * npm install lambda-stream
 * ```
 *
 * Then, you can use the `streamifyResponse` function to wrap your handler:
 *
 * ```ts title="index.ts"
 * import { APIGatewayProxyEventV2 } from "aws-lambda";
 * import { streamifyResponse, ResponseStream } from "lambda-stream";
 *
 * export const handler = streamifyResponse(myHandler);
 *
 * async function myHandler(
 *   _event: APIGatewayProxyEventV2,
 *   responseStream: ResponseStream
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
 * When deployed, this will use the `awslambda.streamifyResponse`.
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
