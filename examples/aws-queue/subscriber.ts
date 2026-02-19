import type { SQSEvent, SQSHandler } from "aws-lambda";

export const handler: SQSHandler = async (event: SQSEvent) => {
  const batchItemFailures = []
  for (const record of event.Records){
    try {
      console.log(record.body)
      if (Math.random() < 0.1){
        throw new Error("An error occurred")
      }
    }
    catch (e) {
      batchItemFailures.push({ itemIdentifier: record.messageId });
    }
  }

  // Failed items will be made visible in the queue again
  return { batchItemFailures };
};
