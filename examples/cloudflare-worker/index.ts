import https from "node:https";
import { Resource } from "sst";

async function requestExampleDomain() {
  return new Promise<{ statusCode: number; body: string }>((resolve, reject) => {
    const req = https.request("https://example.com", { method: "GET" }, (res) => {
      const chunks: Uint8Array[] = [];

      res.on("data", (chunk) => {
        chunks.push(typeof chunk === "string" ? Buffer.from(chunk) : chunk);
      });
      res.on("end", () => {
        resolve({
          statusCode: res.statusCode ?? 0,
          body: Buffer.concat(chunks).toString("utf-8"),
        });
      });
      res.on("error", reject);
    });

    req.on("error", reject);
    req.end();
  });
}

export default {
  async fetch(req: Request) {
    const url = new URL(req.url);

    if (url.pathname === "/https") {
      const result = await requestExampleDomain();
      return Response.json({
        statusCode: result.statusCode,
        ok: result.body.includes("Example Domain"),
      });
    }

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
