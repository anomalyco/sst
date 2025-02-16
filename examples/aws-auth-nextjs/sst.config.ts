/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-auth-nextjs",
      removal: input?.stage === "production" ? "retain" : "remove",
      protect: ["production"].includes(input?.stage),
      home: "aws",
    };
  },
  async run() {
    const auth = new sst.aws.Auth("MyAuth", {
      issuer: "auth/index.handler",
    });

    new sst.aws.Nextjs("MyWeb", {
      link: [auth],
    });
  },
});
