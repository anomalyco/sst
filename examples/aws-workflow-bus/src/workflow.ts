import { workflow } from "sst/aws/workflow";

interface Event {
  id: string;
  time: string;
  "detail-type": string;
  detail: {
    properties: {
      message: string;
      requestId: string;
      requestedAt: string;
    };
  };
}

export const handler = workflow.handler<Event>(async (event, context) => {
  const request = await context.step("start", async ({ logger }) => {
    const startedAt = new Date().toISOString();

    logger.info({
      detailType: event["detail-type"],
      eventMessage: event.detail.properties.message,
      message: "Workflow invoked by bus",
      requestId: event.detail.properties.requestId,
      requestedAt: event.detail.properties.requestedAt,
      startedAt,
    });

    return {
      eventMessage: event.detail.properties.message,
      requestId: event.detail.properties.requestId,
      requestedAt: event.detail.properties.requestedAt,
      sourceEventId: event.id,
      sourceEventTime: event.time,
      startedAt,
    };
  });

  await context.wait("ten-seconds", { seconds: 10 });

  await context.step("finish", async ({ logger }) => {
    const finishedAt = new Date().toISOString();

    logger.info({
      finishedAt,
      message: "Workflow finished",
      ...request,
    });
  });
});
