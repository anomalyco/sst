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
