import { env } from "cloudflare:workers";
import { Resource } from "sst/resource";

export default {
  async fetch(req: Request) {
    if (req.method == "PUT") {
      const key = crypto.randomUUID();
      await Resource.MyBucket.put(key, req.body, {
        httpMetadata: {
          contentType: req.headers.get("content-type"),
        },
      });
      return new Response(`Object created with key: ${key}`);
    }

    if (req.method == "GET") {
      const objects = await Resource.MyBucket.list();
      return Response.json({
        app: Resource.App,
        environment: {
          API_URL: (env as { API_URL?: string }).API_URL,
        },
        secret: Resource.MySecret.value,
        bucket: {
          objects: objects.objects.map((object) => object.key),
        },
      });
    }

    return new Response("Method not allowed", { status: 405 });
  },
};
