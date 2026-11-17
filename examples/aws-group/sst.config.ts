/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Group
 *
 * Use `sst.x.Group` to target or exclude a set of components together.
 *
 * ```bash
 * sst deploy --target Api
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "aws-group",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("MyBucket");
    const fn = new sst.aws.Function("MyFunction", {
      handler: "index.handler",
      url: true,
    });

    const api = new sst.x.Group("Api");
    api.add(bucket, fn);

    return {
      url: fn.url,
      bucket: bucket.name,
    };
  },
});
