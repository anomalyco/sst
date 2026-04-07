import type { APIGatewayProxyEventV2 } from "aws-lambda";
import { Resource } from "sst";
import { workflow } from "sst/aws/workflow";

export const handler = async (event: APIGatewayProxyEventV2) => {
  const action = event.queryStringParameters?.action;
  const message = event.queryStringParameters?.message ?? "Hello from the invoker";
  const name =
    event.queryStringParameters?.name ?? `workflow-example-${Date.now()}`;

  const response = await workflow.start(Resource.Workflow, {
    name,
    payload: {
      ...(action === "succeed" || action === "fail" || action === "heartbeat"
        ? { action }
        : {}),
      message,
      resolverUrl: Resource.Resolver.url,
    },
  });

  return {
    statusCode: 200,
    body: JSON.stringify(
      {
        workflowExecutionArn: response.arn,
        executedVersion: response.version,
        name,
        resolverUrl: Resource.Resolver.url,
        statusCode: response.statusCode,
      },
      null,
      2,
    ),
  };
};
