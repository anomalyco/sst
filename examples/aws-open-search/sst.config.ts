/// <reference path="./.sst/platform/config.d.ts" />

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
