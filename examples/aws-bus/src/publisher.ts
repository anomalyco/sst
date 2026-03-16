import { bus } from "sst/aws/bus";
import { z } from "zod";
import { Resource } from "sst";
import { MyEvent } from "./events";

export async function handler() {
  await bus.publish(Resource.Bus, MyEvent, { foo: "hello" });

  return {
    statusCode: 200,
  };
}
