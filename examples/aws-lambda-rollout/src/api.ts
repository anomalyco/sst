export async function handler() {
  return {
    statusCode: 200,
    body: JSON.stringify({
      message: "Hello from the API",
      version: process.env.AWS_LAMBDA_FUNCTION_VERSION ?? "unknown",
    }),
  };
}
