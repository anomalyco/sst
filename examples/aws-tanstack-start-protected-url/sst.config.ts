/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-tanstack-start-protected-url",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    new sst.aws.TanStackStart("MyWeb", {
      server: {
        runtime: "nodejs22.x",
        protectedUrl: true,
      },
    });
  },
});
