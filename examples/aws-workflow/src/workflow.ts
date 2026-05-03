import { workflow } from "sst/aws/workflow";

interface Event {
  resolverUrl: string;
}

export const handler = workflow.handler<Event>(async (event, ctx) => {
  await ctx.step("start", async ({ logger }) => {
    logger.info("Workflow started.");
  });

  const result = await ctx.waitForCallback(
    "callback",
    async (token, { logger }) => {
      const resolverUrl = new URL(event.resolverUrl);
      resolverUrl.searchParams.set("token", token);

      logger.info(resolverUrl.toString());
    },
    {
      timeout: {
        minutes: 5,
      },
    },
  );

  await ctx.step("finish", async ({ logger }) => {
    logger.info(result);
  });
});
