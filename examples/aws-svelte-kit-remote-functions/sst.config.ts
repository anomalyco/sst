/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-svelte-kit-remote-functions",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    new sst.aws.SvelteKit("MyWeb", {
      dev: {
        command: "bun run dev",
      },
    });
  },
});
