/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda AI streaming
 *
 * An example on how to stream AI responses from a Lambda function using the
 * [AI SDK](https://ai-sdk.dev).
 *
 * Uses `streamText` from the AI SDK to stream a response
 * through a Lambda function URL.
 *
 * ```ts title="sst.config.ts"
 * {
 *   streaming: true
 * }
 * ```
 *
 * The handler uses `awslambda.streamifyResponse` to stream the AI response
 * back to the client as it's generated.
 *
 * ```ts title="index.ts"
 * export const handler = awslambda.streamifyResponse(
 *   async (_event, responseStream) => {
 *     const result = streamText({
 *       model: "amazon/nova-micro",
 *       prompt: "Write a poem about clouds that is twenty paragraphs long.",
 *     });
 *
 *     responseStream.setContentType("text/plain");
 *     for await (const chunk of result.textStream) {
 *       responseStream.write(chunk);
 *     }
 *     responseStream.end();
 *   },
 * );
 * ```
 *
 * Set the API key for the AI gateway.
 *
 * ```bash
 * sst secret set AiGatewayApiKey your-api-key-here
 * ```
 *
 * Use the "Run Client" dev command in the multiplexer to invoke the server and see
 * the streamed response.
 *
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-ai-stream",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const server = new sst.aws.Function("Server", {
      url: true,
      streaming: true,
      timeout: "15 minutes",
      handler: "index.handler",
      environment: {
        AI_GATEWAY_API_KEY: new sst.Secret("AiGatewayApiKey").value,
      },
    });

    new sst.x.DevCommand("Client", {
      dev: {
        autostart: false,
        command: $interpolate`curl --no-buffer ${server.url}`,
        title: "Run Client",
      },
    });

    return {
      url: server.url,
    };
  },
});
