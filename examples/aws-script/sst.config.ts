/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Script
 *
 * Run different lambda functions for every lifecycle event of the resource (created, updated or deleted).
 */
export default $config({
  app(input) {
    return {
      name: "aws-script",
      home: "aws",
      removal: input.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    new sst.aws.Script("MyScript", {
      onCreate: "function.create",
      onUpdate: "function.update",
      onDelete: "function.remove",
      event: { foo: "bar" },
    });
  },
});
