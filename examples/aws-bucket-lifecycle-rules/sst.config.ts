/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Bucket policy
 *
 * Create an S3 bucket and transform its bucket policy.
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
          expiresIn: "30 days",
        },
        {
          id: "expire-tmp-files",
          prefix: "tmp1/",
          expiresIn: "30 days",
        },
        {
          // id: "expire-tmp-files",
          prefix: "tmp2/",
          expiresAt: "2028-12-31",
        },
      ],
    });

    return {
      bucket: bucket.name,
    };
  },
});
