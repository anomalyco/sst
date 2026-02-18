/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS OpenSearch
 *
 * In this example we create a new OpenSearch domain, link it to a function, and
 * then query it.
 *
 * Start by creating a new OpenSearch domain.
 *
 * ```ts title="sst.config.ts"
 * const search = new sst.aws.OpenSearch("MySearch");
 * ```
 *
 * Once linked to a function, we can connect to it.
 *
 * ```ts title="index.ts"
 * import { Resource } from "sst";
 * import { Client } from "@opensearch-project/opensearch";
 * 
 * const client = new Client({
 *   node: Resource.MySearch.url,
 *   auth: {
 *     username: Resource.MySearch.username,
 *     password: Resource.MySearch.password
 *   }
 * });
 * ```
 *
 * This is using the [OpenSearch JS SDK](https://docs.opensearch.org/docs/latest/clients/javascript/index) to connect to the OpenSearch domain..
 */
export default $config({
  app(input) {
    return {
      name: "aws-open-search",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const search = new sst.aws.OpenSearch("MySearch");
    const app = new sst.aws.Function("MyApp", {
      handler: "index.handler",
      url: true,
      link: [search],
    });

    return {
      app: app.url,
      url: search.url,
      username: search.username,
      password: search.password,
    };
  },
});
