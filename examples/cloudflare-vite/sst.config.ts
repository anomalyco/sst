/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare SPA with Vite
 *
 * Deploy a single-page app (SPA) with Vite to Cloudflare.
 */
export default $config({
  app(input) {
    return {
      name: "cloudflare-vite",
      home: "cloudflare",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    new sst.cloudflare.StaticSiteV2("Vite", {
      notFound: "single-page-application",
      dev: {
        title: "Example",
        command: "bun dev"
      },
      build: {
        command: "pnpm run build",
        output: "dist",
      },
    });
  },
});
