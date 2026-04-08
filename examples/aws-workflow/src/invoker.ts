import { Resource } from "sst";
import { workflow } from "sst/aws/workflow";

export const handler = async () => {
  await workflow.start(Resource.Workflow, {
    name: `workflow-example-${Date.now()}`,
    payload: {
      resolverUrl: Resource.Resolver.url,
    },
  });

  return "Workflow started. Check the logs for the callback URL.";
};
