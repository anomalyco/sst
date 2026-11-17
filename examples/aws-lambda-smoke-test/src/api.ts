export async function handler(event: any) {
  if (event.type === "health-check") {
    return {
      statusCode: 200,
      body: JSON.stringify({ status: "healthy" }),
    };
  }

  return {
    statusCode: 200,
    body: JSON.stringify({
      message: "Hello from the API",
      version: process.env.AWS_LAMBDA_FUNCTION_VERSION,
    }),
    headers: {
      "content-type": "application/json",
    },
  };
}
