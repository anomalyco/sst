export const handler = awslambda.streamifyResponse(
  async (event, stream) => {
    stream = awslambda.HttpResponseStream.from(stream, {
      statusCode: 200,
      headers: {
        "Content-Type": "text/plain; charset=UTF-8",
        "X-Content-Type-Options": "nosniff",
      },
    });

    stream.write("Hello ");
    await new Promise((resolve) => setTimeout(resolve, 3000));
    stream.write("World");

    stream.end();
  },
);
