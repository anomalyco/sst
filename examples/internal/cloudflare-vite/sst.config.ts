/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-vite",
      removal: input?.stage === "production" ? "retain" : "remove",
      providers: {
        cloudflare: {},
      },
      home: "cloudflare",
    };
  },
  async run() {
    new sst.cloudflare.x.StaticSite("Web", {
      build: {
        command: "pnpm run build",
        output: "dist",
      },
    });
  },
});
