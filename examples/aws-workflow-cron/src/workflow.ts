import { workflow } from "sst/aws/workflow";

export const handler = workflow.handler(async (event, ctx) => {
  await ctx.step("start", async ({ logger }) => {
    logger.info("Workflow invoked by cron.");
  });

  await ctx.wait("ten-seconds", { seconds: 10 });

  await ctx.step("finish", async ({ logger }) => {
    logger.info("Workflow finished.");
  });
});
