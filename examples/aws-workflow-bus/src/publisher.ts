import { Resource } from "sst";
import { bus } from "sst/aws/bus";

export async function handler() {
  await bus.publish(Resource.Bus, "app.workflow.requested", {
    message: "Hello from the bus",
    requestId: `workflow-request-${Date.now()}`,
    requestedAt: new Date().toISOString(),
  });

  return "Message published. Check the logs to see the workflow start.";
}
