import { workflow } from "sst/aws/workflow";

export const handler = workflow.handler(async (_event, context) => {
  const startedAt = await context.step("start", async ({ logger }) => {
    const startedAt = new Date().toISOString();

    logger.info({
      message: "Workflow invoked by cron",
      startedAt,
    });

    return startedAt;
  });

  await context.wait("ten-seconds", { seconds: 10 })

  await context.step("finish", async ({ logger }) => {
    const finishedAt = new Date().toISOString();

    logger.info({
      finishedAt,
      message: "Workflow finished",
      startedAt,
    });
  });
});
