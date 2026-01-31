import { Resource } from "sst";

export default {
  async fetch(request, env) {
    await env.MyQueue.send({ hello: "world" });
    return new Response("Message sent!");
  },
};
