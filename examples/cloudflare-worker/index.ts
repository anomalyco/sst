import { Resource } from "sst";

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
      const first = await Resource.MyBucket.list().then(
        (res) =>
          res.objects.toSorted(
            (a, b) => a.uploaded.getTime() - b.uploaded.getTime(),
          )[0],
      );
      if (!first) {
        return new Response("No objects found");
      }
      const result = await Resource.MyBucket.get(first.key);
      return new Response(result.body, {
        headers: {
          "content-type": result.httpMetadata.contentType,
        },
      });
    }
  },
};
