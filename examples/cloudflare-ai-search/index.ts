import { Resource } from "sst";

export default {
  async fetch(req: Request) {
    const url = new URL(req.url);
    const path = url.pathname;

    // POST /instances — create a new AI Search instance
    if (req.method === "POST" && path === "/instances") {
      const body = (await req.json()) as { id: string };
      if (!body.id) {
        return Response.json({ error: "Missing 'id' in body" }, { status: 400 });
      }
      const instance = await Resource.Search.create({ id: body.id });
      const info = await instance.info();
      return Response.json(info, { status: 201 });
    }

    // GET /instances — list all instances in the namespace
    if (req.method === "GET" && path === "/instances") {
      const result = await Resource.Search.list();
      return Response.json(result);
    }

    // DELETE /instances/:name — delete an instance
    const deleteMatch = path.match(/^\/instances\/([^/]+)$/);
    if (req.method === "DELETE" && deleteMatch) {
      await Resource.Search.delete(deleteMatch[1]);
      return new Response(null, { status: 204 });
    }

    // POST /instances/:name/items — upload a file to an instance
    const uploadMatch = path.match(/^\/instances\/([^/]+)\/items$/);
    if (req.method === "POST" && uploadMatch) {
      const instance = Resource.Search.get(uploadMatch[1]);
      const filename = url.searchParams.get("filename");
      if (!filename || !req.body) {
        return Response.json(
          { error: "Provide ?filename= and a request body" },
          { status: 400 },
        );
      }
      const item = await instance.items.uploadAndPoll(filename, req.body);
      return Response.json(item, { status: 201 });
    }

    // GET /instances/:name/search?q= — search an instance
    const searchMatch = path.match(/^\/instances\/([^/]+)\/search$/);
    if (req.method === "GET" && searchMatch) {
      const query = url.searchParams.get("q");
      if (!query) {
        return Response.json(
          { error: "Pass a ?q= query parameter" },
          { status: 400 },
        );
      }
      const instance = Resource.Search.get(searchMatch[1]);
      const results = await instance.search({ query });
      return Response.json(results);
    }

    return Response.json(
      {
        routes: [
          "GET    /instances                         — list instances",
          "POST   /instances           {id}          — create instance",
          "DELETE /instances/:name                    — delete instance",
          "POST   /instances/:name/items ?filename=  — upload file (body = content)",
          "GET    /instances/:name/search ?q=         — search instance",
        ],
      },
      { status: 404 },
    );
  },
};
