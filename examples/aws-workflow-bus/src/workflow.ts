import { workflow } from "sst/aws/workflow";

interface Event {
  "detail-type": string;
  detail: {
    properties: {
      message: string;
    };
  };
}

export const handler = workflow.handler<Event>(async (event, ctx) => {
  await ctx.step("start", async ({ logger }) => {
    logger.info("Workflow invoked by bus.");
  });

  await ctx.wait("ten-seconds", { seconds: 10 });

  await ctx.step("finish", async ({ logger }) => {
    logger.info("Workflow finished.");
  });
});
