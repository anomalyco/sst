/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Monorepo
 *
 * Structure your app as a monorepo by importing infrastructure definitions from
 * a separate file.
 */
export default $config({
  app(input) {
    return {
      name: "aws-monorepo",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const infra = await import("./infra");

    return {
      api: infra.api.url,
      astro: infra.astro.url,
    };
  },
});
