export const handler = awslambda.streamifyResponse(
  async (_event: unknown, responseStream: any) => {
    responseStream = awslambda.HttpResponseStream.from(responseStream, {
      statusCode: 200,
      headers: { "Content-Type": "text/plain" },
    });

    responseStream.write("Hello");
    await new Promise((resolve) => setTimeout(resolve, 3000));
    responseStream.write(" World");
    responseStream.end();
  },
);
