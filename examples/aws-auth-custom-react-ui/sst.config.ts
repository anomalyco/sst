/// <reference path="./.sst/platform/config.d.ts" />

/**
 */
export default $config({
  app(input) {
    return {
      name: "aws-auth-custom-react-ui",
      removal: input?.stage === "production" ? "retain" : "remove",
      protect: ["production"].includes(input?.stage),
      home: "aws",
    };
  },
  async run() {
    await import("./infra/auth");
    await import("./infra/api");
    await import("./infra/web");
  },
});
