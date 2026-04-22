/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-solid",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    new sst.cloudflare.x.SolidStart("MyWeb", {
      environment: {
        MESSAGE: "Hello from SST on Cloudflare",
      },
      transform: {
        server: {
          placement: {
            region: "aws:eu-west-1",
          },
        },
      },
    });
  },
});
