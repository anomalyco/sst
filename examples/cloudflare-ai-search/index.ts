import { Resource } from "sst";

export default {
  async fetch(req: Request) {
    const url = new URL(req.url);
    const query = url.searchParams.get("q");
    const instance = url.searchParams.get("instance");

    if (!query) {
      return new Response("Pass a ?q= query parameter to search", {
        status: 400,
      });
    }

    if (!instance) {
      return new Response(
        "Pass an ?instance= parameter for the instance name",
        { status: 400 },
      );
    }

    const search = Resource.Search.get(instance);
    const results = await search.search({
      query,
    });

    return Response.json(results);
  },
};
