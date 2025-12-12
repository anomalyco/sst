import type { SQSEvent, SQSHandler } from "aws-lambda";

export const main = async (event) => {
  console.log(event);
  throw new Error("Manual error");
};

export const dlq: SQSHandler = async (event: SQSEvent) => {
  console.log(event);
  return;
};
