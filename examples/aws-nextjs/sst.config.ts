/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-nextjs",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("MyBucket", {
      access: "public",
    });

    const secret = new sst.Secret("MySecret", "aws-nextjs-secret");

    new sst.aws.Nextjs("MyWeb", {
      link: [bucket, secret],
      environment: {
        API_URL: "https://example.com/aws-nextjs",
      },
    });
  }
});
