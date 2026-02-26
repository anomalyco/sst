declare const awslambda: any;

export const handler = awslambda.streamifyResponse(
  async (_event: any, responseStream: any, _context: any) => {
    responseStream = awslambda.HttpResponseStream.from(responseStream, {
      statusCode: 200,
      headers: { "content-type": "text/plain" },
    });
    responseStream.write("Hello");
    await new Promise((resolve) => setTimeout(resolve, 3000));
    responseStream.write(" World");
    responseStream.end();
  },
);
