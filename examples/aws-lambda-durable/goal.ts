import { workflow } from "sst/aws/workflow";

export const handler = workflow.handler(async (event, ctx) => {
  const email = await ctx.step("generate", async () => {
    return "Generated with AI";
  });

  await ctx.waitForCallback("webhook", async (token, { logger }) => {
    logger.info(`Use ${token} to confirm the send.`);
  });

  await ctx.step(
    "send-email",
    async ({ logger }) => {
      logger.info(`Sending email: ${email}`);
    },
    {
      retryStrategy: (error, attempts) => attempts <= 2,
    },
  );
});

workflow.start(Resource.Durable, { name, payload });
workflow.succeed("token", { payload });
workflow.fail("token", { error });
workflow.heartbeat("token");
