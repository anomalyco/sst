import { Resource } from "sst";
import { DurableObject } from "cloudflare:workers";

export default {
  async fetch(request: Request) {
    const url = new URL(request.url);

    if (url.pathname === "/favicon.ico") {
      return new Response(null, { status: 204 });
    }

    const stub = Resource.Counter.getByName("global");
    return stub.fetch("https://counter/");
  },
};

export class Counter extends DurableObject {
  async fetch() {
    const current = (await this.ctx.storage.get<number>("count")) ?? 0;
    const count = current + 1;

    await this.ctx.storage.put("count", count);

    console.log("durable object hit", { count });

    return Response.json({ count });
  }
}
