import { APIGatewayProxyEventV2 } from "aws-lambda";
import { workflow } from "sst/aws/workflow";

export const handler = async (event: APIGatewayProxyEventV2) => {
  const token = event.queryStringParameters?.token;

  if (!token) {
    return "Missing token in query parameters";
  }

  await workflow.succeed(token, {
    payload: {
      message: "Sent from the resolver."
    },
  });

  return "Workflow callback sent. Check the logs to see the workflow succeed." ;
};
