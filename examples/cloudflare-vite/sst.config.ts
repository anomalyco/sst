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
    new sst.cloudflare.x.StaticSite("Vite2", {
      build: {
        command: "pnpm run build",
        output: "dist",
      },
      notFound: "spa",
    });
  },
});
