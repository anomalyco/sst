/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "{{.App}}",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "{{.Home}}",
    };
  },
  async run() {
    new sst.{{.Home}}.TanStackStart("MyWeb");
  },
});
