import { Resource } from "sst/resource";

export default {
  async fetch(req: Request) {
    const url = new URL(req.url);

    const outcome = await Resource.MyRateLimit.limit({ key: url.pathname });
    if (!outcome.success) {
      return new Response(`Rate limit exceeded for ${url.pathname}`, { status: 429 });
    }

    return new Response("OK", { status: 200 });
  },
};
