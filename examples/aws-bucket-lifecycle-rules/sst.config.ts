/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Bucket lifecycle policies
 *
 * Configure S3 bucket lifecycle policies to expire objects automatically.
 */
export default $config({
  app(input) {
    return {
      name: "aws-bucket-lifecycle-rules",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("MyBucket", {
      lifecycle: [
        {
          expiresIn: "60 days",
        },
        {
          id: "expire-tmp-files",
          prefix: "tmp/",
          expiresIn: "30 days",
        },
        {
          prefix: "data/",
          expiresAt: "2028-12-31",
        },
      ],
    });

    return {
      bucket: bucket.name,
    };
  },
});
