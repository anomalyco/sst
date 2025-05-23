import * as sst from "sst-plugin";
import { FunctionArgs, FunctionArn } from "../function.js";
import { Queue } from "../queue.js";

export function isFunctionSubscriber(
  subscriber?: sst.Input<string | FunctionArgs | FunctionArn>,
) {
  if (!subscriber) return sst.output(false);

  return sst
    .output(subscriber)
    .apply(
      (subscriber) =>
        typeof subscriber === "string" ||
        typeof subscriber.handler === "string",
    );
}

export function isQueueSubscriber(subscriber?: sst.Input<string | Queue>) {
  if (!subscriber) return sst.output(false);

  return sst
    .output(subscriber)
    .apply(
      (subscriber) =>
        typeof subscriber === "string" || subscriber instanceof Queue,
    );
}
