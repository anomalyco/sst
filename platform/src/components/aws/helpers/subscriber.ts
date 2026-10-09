import { Input, output } from "@pulumi/pulumi";
import { Function, FunctionArgs, FunctionArn } from "../function.js";
import { Queue } from "../queue";

export function isFunctionSubscriber(
  subscriber?: Input<string | Function | FunctionArgs | FunctionArn>,
) {
  if (!subscriber) return output(false);

  return output(subscriber).apply(
    (subscriber) =>
      typeof subscriber === "string" ||
      subscriber instanceof Function ||
      typeof subscriber.handler === "string",
  );
}

export function isQueueSubscriber(subscriber?: Input<string | Queue>) {
  if (!subscriber) return output(false);

  return output(subscriber).apply(
    (subscriber) =>
      typeof subscriber === "string" || subscriber instanceof Queue,
  );
}
