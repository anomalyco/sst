import { Resource } from "sst";

export default {
  async fetch() {
    await Resource.MyQueue.send({ hello: "world" });
    return new Response("Message sent!");
  },
};
