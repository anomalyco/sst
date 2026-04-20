/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-workflow",
      home: "cloudflare",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const workflow = new sst.cloudflare.Workflow("SignupWorkflow", {
      handler: "workflow.ts",
      className: "SignupWorkflow",
    });

    const api = new sst.cloudflare.Worker("Api", {
      handler: "api.ts",
      link: [workflow],
      url: true,
    });

    return {
      url: api.url,
    };
  },
});
