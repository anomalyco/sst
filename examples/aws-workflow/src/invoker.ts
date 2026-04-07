import { Resource } from "sst";
import { workflow } from "sst/aws/workflow";

export const handler = async () => {
  const name = `workflow-example-${Date.now()}`;

  const response = await workflow.start(Resource.Workflow, {
    name,
    payload: {
      resolverUrl: Resource.Resolver.url,
    },
  });

  return {
    statusCode: 200,
    body: JSON.stringify(
      {
        message: "Workflow started. Check the workflow logs for the callback URL.",
        workflowExecutionArn: response.arn,
        workflowExecutionName: name,
      },
      null,
      2,
    ),
  };
};
