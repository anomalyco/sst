import { workflow } from "sst/aws/workflow";

interface Event {
  action?: "succeed" | "fail" | "heartbeat";
  message?: string;
  resolverUrl?: string;
}

export const handler = workflow.handler<Event>(async (event, context) => {
  const step1 = await context.step("step1", async ({ logger }) => {
    const message = event.message ?? "Hello";
    logger.info(`Executing step 1: ${message}`);
    return message;
  });

  const callbackResult = await context.waitForCallback(
    "callback",
    async (callbackId, { logger }) => {
      logger.info({ callbackId });

      if (!event.resolverUrl) return;

      const resolverUrl = new URL(event.resolverUrl);
      resolverUrl.searchParams.set("callbackId", callbackId);
      resolverUrl.searchParams.set("action", event.action ?? "succeed");
      resolverUrl.searchParams.set("message", step1);

      const response = await fetch(resolverUrl, {
        method: "POST",
      });
      logger.info({
        resolverBody: await response.text(),
        resolverStatus: response.status,
      });
    },
    {
      timeout: {
        minutes: 5,
      },
    },
  );

  return context.step("complete", async ({ logger }) => {
    logger.info({ callbackResult, step1 });
    return {
      callbackResult,
      step1,
    };
  });
});
