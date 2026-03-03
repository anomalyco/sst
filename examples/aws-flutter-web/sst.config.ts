/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Flutter web
 *
 * Deploy a Flutter web app as a static site to S3 and CloudFront.
 */
export default $config({
  app(input) {
    return {
      name: "aws-flutter-web",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    new sst.aws.StaticSite("MySite", {
      build: {
        command: "flutter build web",
        output: "build/web",
      },
    });
  },
});
