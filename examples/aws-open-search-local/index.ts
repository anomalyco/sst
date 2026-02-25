import { Resource } from "sst";
import { Client } from "@opensearch-project/opensearch";

const client = new Client({
  node: Resource.MySearch.url,
  auth: {
    username: Resource.MySearch.username,
    password: Resource.MySearch.password,
  },
});

export async function handler() {
  // Add a document
  await client.index({
    index: "my-index",
    body: { message: "Hello world!" },
  });

  // Search for documents
  const result = await client.search({
    index: "my-index",
    body: { query: { match: { message: "world" } } },
  });

  return {
    statusCode: 200,
    body:
      `Querying ${Resource.MySearch.url}\n\n` +
      JSON.stringify(result.body.hits, null, 2),
  };
}
