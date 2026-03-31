import { Resource } from "sst";

export default {
  async fetch(req: Request) {
    // curl -X POST -d "some-value" https://<url>
    if (req.method == "POST") {
      const key = crypto.randomUUID();
      const body = await req.text();
      await Resource.MyStorage.put(key, body);
      return new Response(key);
    }

    // curl https://<url>/<key>
    if (req.method == "GET") {
      const id = new URL(req.url).pathname.slice(1);
      if (!id) return new Response("use POST to write, GET /<key> to read");
      const result = await Resource.MyStorage.get(id);
      return new Response(result);
    }

    return new Response("not found", { status: 404 });
  },
};
