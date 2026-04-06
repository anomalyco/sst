import {
  withDurableExecution,
  DurableContext,
} from "@aws/durable-execution-sdk-js";

export const handler = withDurableExecution(async (event, context) => {
  const step1 = await context.step("step1", async ({ logger }) => {
    logger.info("Executing step 1");
    return "Hello";
  });

  const callbackResult = await context.waitForCallback(
    "callback",
    async (callbackToken, { logger }) => {
      logger.info({ callbackToken });
    },
    {
      timeout: {
        minutes: 5,
      },
    },
  );
});
