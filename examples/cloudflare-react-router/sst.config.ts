/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare React Router
 *
 * Deploy a [React Router](https://reactrouter.com) app to Cloudflare.
 */
export default $config({
  app(input) {
    return {
      name: "cloudflare-react-router",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const kv = new sst.cloudflare.Kv("MyKv");

    new sst.cloudflare.ReactRouter("MyWeb", {
      link: [kv],
    });
  },
});
