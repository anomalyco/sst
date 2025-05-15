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
        myplugin: "workspace:*",
      },
    };
  },
  async run() {
    const MyResource = sst.resource({
      type: "MyResource",
      async create(inputs: { butt: number }) {
        return {
          id: "123",
          outputs: {
            hello: "world",
            updated: Date.now(),
          },
        };
      },
      async delete(id, inputs) {
        console.log("remove");
      },
      async update(id, state, news) {
        return {
          ...state.outputs,
          updated: Date.now(),
        };
      },
    });

    let first = false;
    process.on("beforeExit", () => {
      if (first) return;
      first = true;
    });
  },
});
