import { workflow } from "sst/aws/workflow";

interface Event {
  resolverUrl: string;
}

export const handler = workflow.handler<Event>(async (event, context) => {
  await context.step("start", async ({ logger }) => {
    logger.info({ message: "Workflow started" });
  });

  const callbackResult = await context.waitForCallback(
    "callback",
    async (token, { logger }) => {
      const resolverUrl = new URL(event.resolverUrl);
      resolverUrl.searchParams.set("token", token);

      logger.info({
        message: "Open this URL to resume the workflow",
        token,
        callbackUrl: resolverUrl.toString(),
      });
    },
    {
      timeout: {
        minutes: 5,
      },
    },
  );

  return {
    callbackResult,
  };
});
