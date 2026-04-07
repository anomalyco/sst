import { APIGatewayProxyEventV2 } from "aws-lambda";
import { workflow } from "sst/aws/workflow";

export const handler = async (event: APIGatewayProxyEventV2) => {
  const token = event.queryStringParameters?.token;

  if (!token) {
    return {
      statusCode: 400,
      body: JSON.stringify({
        message: "Missing token in query parameters",
      }),
    };
  }

  await workflow.succeed(token, {
    payload: {
      message: "Callback received",
    },
  });

  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Workflow callback sent." }),
  };
};
