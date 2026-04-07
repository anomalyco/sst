import { Resource } from "sst";
import { bus } from "sst/aws/bus";

export async function handler() {
  const requestId = `workflow-request-${Date.now()}`;
  const requestedAt = new Date().toISOString();

  await bus.publish(Resource.Bus, "app.workflow.requested", {
    message: "Hello from the bus",
    requestId,
    requestedAt,
  });

  return {
    statusCode: 200,
    body: JSON.stringify(
      {
        bus: Resource.Bus.name,
        detailType: "app.workflow.requested",
        message: "Hello from the bus",
        requestId,
        requestedAt,
      },
      null,
      2,
    ),
  };
}
