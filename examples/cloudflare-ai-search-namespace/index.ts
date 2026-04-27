import { Resource } from "sst";

export default {
  async fetch(req: Request) {
    const url = new URL(req.url);
    const path = url.pathname;

    // List all instances in the namespace.
    if (req.method === "GET" && path === "/instances") {
      const result = await Resource.Search.list();
      return Response.json(result);
    }

    // Create an instance: POST /instances { "id": "my-docs" }
    if (req.method === "POST" && path === "/instances") {
      const body = (await req.json()) as { id: string };
      if (!body.id)
        return Response.json({ error: "Missing 'id'" }, { status: 400 });
      const instance = await Resource.Search.create({ id: body.id });
      const info = await instance.info();
      return Response.json(info, { status: 201 });
    }

    // Delete an instance: DELETE /instances/my-docs
    const deleteMatch = path.match(/^\/instances\/([^/]+)$/);
    if (req.method === "DELETE" && deleteMatch) {
      await Resource.Search.delete(deleteMatch[1]);
      return new Response(null, { status: 204 });
    }

    // Upload a document to a specific instance:
    //   PUT /instances/my-docs/items?filename=guide.md
    const uploadMatch = path.match(/^\/instances\/([^/]+)\/items$/);
    if (req.method === "PUT" && uploadMatch) {
      const instance = Resource.Search.get(uploadMatch[1]);
      const filename = url.searchParams.get("filename");
      if (!filename || !req.body)
        return Response.json(
          { error: "Provide ?filename= and a request body" },
          { status: 400 },
        );
      const item = await instance.items.uploadAndPoll(filename, req.body);
      return Response.json(item, { status: 201 });
    }

    // Search a specific instance:
    //   GET /instances/my-docs/search?q=caching
    const searchMatch = path.match(/^\/instances\/([^/]+)\/search$/);
    if (req.method === "GET" && searchMatch) {
      const instance = Resource.Search.get(searchMatch[1]);
      const query = url.searchParams.get("q") ?? "";
      const results = await instance.search({ query });
      return Response.json(results);
    }

    // Search across multiple instances at once:
    //   GET /search?q=caching&instances=docs,blog
    if (req.method === "GET" && path === "/search") {
      const query = url.searchParams.get("q") ?? "";
      const ids = (url.searchParams.get("instances") ?? "").split(",");
      if (!ids.length)
        return Response.json(
          { error: "Provide ?instances=id1,id2" },
          { status: 400 },
        );
      const results = await Resource.Search.search({
        query,
        ai_search_options: { instance_ids: ids },
      });
      return Response.json(results);
    }

    return Response.json(
      {
        routes: [
          "GET    /instances                     — list instances",
          "POST   /instances {id}                — create instance",
          "DELETE /instances/:name               — delete instance",
          "PUT    /instances/:name/items?filename — upload document",
          "GET    /instances/:name/search?q=     — search one instance",
          "GET    /search?q=&instances=a,b       — search across instances",
        ],
      },
      { status: 404 },
    );
  },
};
