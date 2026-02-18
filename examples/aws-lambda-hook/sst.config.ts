/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda build hook
 * 
 * In this example we hook into the Lambda function build process with
 * `hook.postbuild`.
 *
 * This is useful for modifying the generated Lambda function code before it's
 * uploaded to AWS. It can also be used for uploading the generated sourcemaps
 * to a service like Sentry.
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-hook",
      removal: input?.stage === "production" ? "retain" : "remove",
      protect: ["production"].includes(input?.stage),
      home: "aws",
    };
  },
  async run() {
    new sst.aws.Function("MyFunction", {
      url: true,
      handler: "index.handler",
      hook: {
        async postbuild(dir) {
          console.log(`postbuild ------- ${dir}`);
        },
      },
    });
  },
});
