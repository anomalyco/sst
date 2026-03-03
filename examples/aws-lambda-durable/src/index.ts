import { withDurableExecution, DurableContext } from "@aws/durable-execution-sdk-js";

export const handler = withDurableExecution(async (event: any, context: DurableContext) => {
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

  const step2 = await context.step("step2", async ({ logger }) => {
    logger.info("Executing step 2");
    return `${step1} World!`;
  });

  return { step1, step2, callbackResult };
});
