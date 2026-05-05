import { Resource } from "sst";

export default {
  async fetch(req: Request) {
    const url = new URL(req.url);

    if (req.method === "POST" && url.pathname === "/orders") {
      const orderId = crypto.randomUUID();
      const instance = await Resource.OrderProcessor.create({
        params: { orderId },
      });
      return Response.json({
        orderId,
        instanceId: instance.id,
        status: await instance.status(),
      });
    }

    if (req.method === "GET" && url.pathname.startsWith("/orders/")) {
      const id = url.pathname.slice("/orders/".length);
      const instance = await Resource.OrderProcessor.get(id);
      return Response.json({
        instanceId: instance.id,
        status: await instance.status(),
      });
    }

    return new Response(
      "POST /orders to trigger a workflow, GET /orders/:id to check status",
      { status: 200 },
    );
  },
};
