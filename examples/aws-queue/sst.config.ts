/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Subscribe to queues
 *
 * Create an SQS queue, subscribe to it, and publish to it from a function.
 * 
 * ```ts title="sst.config.ts"
 * const queue = new sst.aws.Queue("MyQueue");
 * queue.subscribe("subscriber.handler");
 * 
 * const app = new sst.aws.Function("MyApp", {
 *   handler: "publisher.handler",
 *   link: [queue],
 *   url: true,
 * });
 * 
 * return {
 *   app: app.url,
 *   queue: queue.url,
 * };
 * ```
 *
 * The subscriber will read messages from the queue in batches. This array of messages exists on the `Records` property of the `SQSEvent`.
 *
 * ```ts title="subscriber.ts"
 * import type { SQSEvent, SQSHandler } from "aws-lambda";
 * 
 * export const handler: SQSHandler = async (event: SQSEvent) => {
 *   for (const record of event.Records){
 *     // Message bodies are always strings
 *     console.log(record.body) 
 *   }
 *   return;
 * };
 * ```
 *
 * By default, all messages in the batch become visible in the queue again if an error occurs. This can lead to unnecessary extra processing and messages being processed more than once. The solution is to enable [partial batch responsese](https://docs.aws.amazon.com/lambda/latest/dg/services-sqs-errorhandling.html#services-sqs-batchfailurereporting) and return which specific messages within the batch should be made visible again in the queue.
 *
 * Update the queue subscriber.
 *
 * ```ts title="sst.config.ts"
 * queue.subscribe("subscriber.handler", {
 *   batch: {
 *     partialResponses: true,
 *   }
 * });
 * ```
 *
 * Then update the handler to return the failed items.
 *
 * ```ts title="subscriber.ts"
 * import type { SQSEvent, SQSHandler } from "aws-lambda";
 * 
 * export const handler: SQSHandler = async (event: SQSEvent) => {
 *   const batchItemFailures = []
 *   for (const record of event.Records){
 *     try {
 *       console.log(record.body)
 *       if (Math.random() < 0.1){
 *         throw new Error("An error occurred")
 *       }
 *     }
 *     catch (e) {
 *       batchItemFailures.push({ itemIdentifier: record.messageId });
 *     }
 *   }
 * 
 *   // Failed items will be made visible in the queue again
 *   return { batchItemFailures };
 * };
 * ```
 *
 */
export default $config({
  app(input) {
    return {
      name: "aws-queue",
      home: "aws",
      removal: input.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const queue = new sst.aws.Queue("MyQueue");
    queue.subscribe("subscriber.handler", {
      batch: {
        partialResponses: true,
      }
    });

    const app = new sst.aws.Function("MyApp", {
      handler: "publisher.handler",
      link: [queue],
      url: true,
    });

    return {
      app: app.url,
      queue: queue.url,
    };
  },
});
