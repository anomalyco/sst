/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda tRPC streaming
 *
 * An example on how to use tRPC with Lambda streaming.
 *
 * Uses `@trpc/server`'s `awsLambdaStreamingRequestHandler` adapter to handle
 * streaming responses through Lambda function URLs.
 *
 * The `trpc-server` function defines a tRPC router and streams responses.
 * The `trpc-client` function invokes the server using `httpBatchStreamLink`.
 *
 * Streaming is supported in both `sst dev` and `sst deploy`.
 *
 */
export default $config({
  app(input) {
    return {
      name: 'aws-lambda-trpc-stream',
      removal: input?.stage === 'production' ? 'retain' : 'remove',
      home: 'aws',
    };
  },
  async run() {
    const trpcServer = new sst.aws.Function('TrpcServer', {
      handler: 'trpc-server.handler',
      streaming: true,
      url: true,
      runtime: 'nodejs24.x',
    });

    new sst.x.DevCommand('Client', {
      dev: {
        autostart: false,
        command: $interpolate`npx tsx trpc-client.ts`,
        title: 'Run Client',
      },
      environment: {
        TRPC_SERVER_URL: trpcServer.url,
      },
    });

    return {
      serverUrl: trpcServer.url,
    };
  },
});
