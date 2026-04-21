/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-monorepo",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      providers: {
        cloudflare: "6.13.0",
      },
    };
  },
  async run() {
    const infra = await import("./infra");

    return {
      api: infra.api.url,
      astro: infra.astro.url,
      worker: infra.worker.url,
    };
  },
});
