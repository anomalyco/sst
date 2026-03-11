export const handler = async (event: any) => {
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ message: "Hello from WAF-protected Router!" }),
  };
};
