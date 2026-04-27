import { Resource } from "sst";

export default {
  async fetch(req: Request) {
    const url = new URL(req.url);

    // Upload a document: PUT /items?filename=guide.md
    if (req.method === "PUT" && url.pathname === "/items") {
      const filename = url.searchParams.get("filename");
      if (!filename || !req.body)
        return Response.json(
          { error: "Provide ?filename= and a request body" },
          { status: 400 },
        );
      const item = await Resource.Search.items.uploadAndPoll(
        filename,
        req.body,
      );
      return Response.json(item, { status: 201 });
    }

    // Search: GET /search?q=how+does+caching+work
    if (req.method === "GET" && url.pathname === "/search") {
      const query = url.searchParams.get("q") ?? "";
      const results = await Resource.Search.search({ query });
      return Response.json(results);
    }

    // Search with filters: GET /search?q=deploy&filter_field=category&filter_value=docs
    // Demonstrates metadata filtering with Vectorize filter syntax.
    if (req.method === "GET" && url.pathname === "/search/filtered") {
      const query = url.searchParams.get("q") ?? "";
      const field = url.searchParams.get("filter_field") ?? "";
      const value = url.searchParams.get("filter_value") ?? "";
      const results = await Resource.Search.search({
        query,
        ai_search_options: {
          retrieval: {
            filters: { [field]: value },
          },
        },
      });
      return Response.json(results);
    }

    // Chat completions: POST /chat  { "question": "What is Cloudflare?" }
    // Returns an AI-generated answer grounded in your indexed content.
    if (req.method === "POST" && url.pathname === "/chat") {
      const body = (await req.json()) as { question: string };
      const answer = await Resource.Search.chatCompletions({
        messages: [{ role: "user", content: body.question }],
      });
      return Response.json(answer);
    }

    // Streaming chat: POST /chat/stream  { "question": "What is Cloudflare?" }
    if (req.method === "POST" && url.pathname === "/chat/stream") {
      const body = (await req.json()) as { question: string };
      const stream = await Resource.Search.chatCompletions({
        messages: [{ role: "user", content: body.question }],
        stream: true,
      });
      return new Response(stream, {
        headers: { "content-type": "text/event-stream" },
      });
    }

    return Response.json(
      {
        routes: [
          "PUT  /items?filename=        — upload a document (body = content)",
          "GET  /search?q=              — search indexed content",
          "GET  /search/filtered?q=&... — search with metadata filters",
          "POST /chat {question}        — AI answer grounded in your content",
          "POST /chat/stream {question} — streaming AI answer",
        ],
      },
      { status: 404 },
    );
  },
};
