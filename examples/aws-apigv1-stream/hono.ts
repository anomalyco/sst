import { Hono } from "hono";
import { streamText } from "hono/streaming";
import { streamHandle } from "hono/aws-lambda";

const app = new Hono().get("/hono", (c) => {
  return streamText(c, async (stream) => {
    await stream.writeln("Hello");
    await stream.sleep(3000);
    await stream.writeln("World");
  });
});

export const handler = streamHandle(app);
