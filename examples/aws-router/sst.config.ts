/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Router and function URL
 *
 * Creates a router that routes all requests to a function with a URL.
 */
export default $config({
  app(input) {
    return {
      name: "aws-router",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const api = new sst.aws.Function("MyApi", {
      handler: "api.handler",
      url: true,
    });
    const bucket = new sst.aws.Bucket("MyBucket", {
      access: "public",
    });
    const router = new sst.aws.Router("MyRouter", {
      // Use the AWS-managed CachingDisabled policy so this distribution is
      // compatible with the CloudFront Free Tier and safe for the API route.
      cachePolicy: sst.aws.cloudfront.cachePolicy.cachingDisabled,
      routes: {
        "/api/*": api.url,
        "/*": { bucket },
      },
    });

    return {
      router: router.url,
      bucket: bucket.domain,
    };
  },
});
