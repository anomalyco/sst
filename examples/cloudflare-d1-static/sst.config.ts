/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-d1-static",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const db = $app.stage !== "production"
      ? sst.cloudflare.D1.get("MyDatabase", "a928590c-34cc-49fd-b3cf-db643db54e9c")
      : new sst.cloudflare.D1("MyDatabase");

    const worker = new sst.cloudflare.Worker("Worker", {
      link: [db],
      url: true,
      handler: "index.ts",
    });

    return {
      dbId: db.databaseId,
      url: worker.url,
    };
  },
});
