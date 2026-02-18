/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "play-cf",
      home: "cloudflare",
    };
  },
  async run() {
    const ret: Record<string, any> = {};
    const secret = new sst.Secret("MySecret", "xyz123");
    const bucket = createBucket();
    const worker = createWorker();
    createAstro();
    createSolidStart();
    createStatic();
    return ret;

    function createBucket() {
      const bucket = new sst.cloudflare.Bucket("MyBucket");
      ret.bucket = bucket.name;
      return bucket;
    }

    function createWorker() {
      const worker = new sst.cloudflare.Worker("MyWorker", {
        handler: "workers/worker.ts",
        link: [bucket],
        url: true,
      });
      ret.worker = worker.url;
      return worker;
    }

    function createAstro() {
      new sst.cloudflare.x.Astro("MyAstro", {
        path: "../sites/astro5",
        link: [bucket],
        environment: {
          FOO: "hello",
        },
      });
    }

    function createSolidStart() {
      new sst.cloudflare.x.SolidStart("MySolidStart", {
        path: "sites/solid-start",
        link: [secret, bucket],
        environment: {
          FOO: "hello",
        },
      });
    }

    function createStatic() {
      new sst.cloudflare.x.StaticSite("MyAstroStatic", {
        errorPage: "404.html",
        path: "../sites/astro5-static",
        build: {
          command: "npm run build:cf",
          output: "dist",
        },
      });
    }
  },
});
