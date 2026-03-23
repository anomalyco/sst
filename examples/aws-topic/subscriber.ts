import type { SQSEvent } from "aws-lambda";

export const handler = async (event: SQSEvent) => {
  console.log(event);
  return "ok";
};
