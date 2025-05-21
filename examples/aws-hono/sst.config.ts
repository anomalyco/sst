/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-hono",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      version: "3.10.13",
      plugins: {
        example: "0.0.10",
      },
    };
  },
  async run() {
    const { MyResource } = await import("./resource");

    new MyResource("my-resource", {
      butt: 1,
    });

    let first = false;
    process.on("beforeExit", () => {
      if (first) return;
      first = true;
    });
  },
});
