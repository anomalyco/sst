/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-kv-static",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const storage = $app.stage !== "production"
      ? sst.cloudflare.Kv.get("MyStorage", {
          namespaceId: "0a41762f971d489bba975c5be92858e2",
        })
      : new sst.cloudflare.Kv("MyStorage");

    const worker = new sst.cloudflare.Worker("Worker", {
      url: true,
      link: [storage],
      handler: "index.ts",
    });

    return {
      kvId: storage.id,
      url: worker.url,
    };
  },
});
