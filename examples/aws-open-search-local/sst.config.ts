/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS OpenSearch local
 *
 * In this example, we connect to a locally running OpenSearch process for dev. While
 * on deploy, we use AWS' OpenSearch Service.
 *
 * We use the [`docker run`](https://docs.docker.com/reference/cli/docker/container/run/)
 * CLI to start a local container with OpenSearch. You don't have to use Docker, you can use
 * any other way to run OpenSearch locally.
 *
 * ```bash
 * docker run \
 *   --rm \
 *   -p 9200:9200 \
 *   -v $(pwd)/.sst/storage/opensearch:/usr/share/opensearch/data \
 *   -e discovery.type=single-node \
 *   -e plugins.security.disabled=true \
 *   -e OPENSEARCH_INITIAL_ADMIN_PASSWORD=^Passw0rd^ \
 *   opensearchproject/opensearch:2.17.0
 * ```
 *
 * The data is saved to the `.sst/storage` directory. So if you restart the dev server, the
 * data will still be there.
 *
 * We then configure the `dev` property of the `OpenSearch` component with the settings for
 * the local OpenSearch instance.
 *
 * ```ts title="sst.config.ts"
 * dev: {
 *   url: "http://localhost:9200",
 *   username: "admin",
 *   password: "^Passw0rd^"
 * }
 * ```
 *
 * By providing the `dev` prop for OpenSearch, SST will use the local OpenSearch process and
 * not deploy a new OpenSearch domain when running `sst dev`.
 *
 * It also allows us to access the local process through a Resource `link` without having
 * to conditionally check if we are running locally.
 *
 * ```ts title="index.ts"
 * const client = new Client({
 *   node: Resource.MySearch.url,
 *   auth: {
 *     username: Resource.MySearch.username,
 *     password: Resource.MySearch.password,
 *   },
 * });
 * ```
 *
 * The above will work in both `sst dev` and `sst deploy`.
 */
export default $config({
  app(input) {
    return {
      name: "aws-open-search-local",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const search = new sst.aws.OpenSearch("MySearch", {
      dev: {
        url: "http://localhost:9200",
        username: "admin",
        password: "^Passw0rd^",
      },
    });

    new sst.aws.Function("MyApp", {
      handler: "index.handler",
      url: true,
      link: [search],
    });
  },
});
