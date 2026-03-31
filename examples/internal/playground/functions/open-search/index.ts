import { Resource } from "sst";
import { Client } from "@opensearch-project/opensearch";

const client = new Client({
  node: Resource.MyOpenSearch.url,
  auth: {
    username: Resource.MyOpenSearch.username,
    password: Resource.MyOpenSearch.password,
  },
});

export async function handler() {
  // Index a document
  await client.index({
    index: "my-index",
    id: "1",
    body: { user: "foo", message: "Hello World!" },
    refresh: true, // make it visible to search immediately
  });

  // Search for a document
  const result = await client.search({
    index: "my-index",
    body: {
      query: { match: { message: "World" } },
    },
  });

  return {
    statusCode: 200,
    body: JSON.stringify(result.body.hits, null, 2),
  };
}
