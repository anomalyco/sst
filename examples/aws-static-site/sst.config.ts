/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS static site
 *
 * Deploy a simple HTML file as a static site with S3 and CloudFront. The website is stored in
 * the `site/` directory.
 */
export default $config({
  app(input) {
    return {
      name: "aws-static-site",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    new sst.aws.StaticSite("MySite", {
      path: "site",
      errorPage: "404.html",
    });
  },
});
