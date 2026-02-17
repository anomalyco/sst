/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Router with WAF
 *
 * Enable WAF (Web Application Firewall) for a Router to protect against common
 * web exploits and bots.
 *
 * WAF includes rate limiting per IP, and AWS managed rules for core rule set,
 * known bad inputs, and SQL injection protection.
 *
 * You can also enable WAF logging to CloudWatch to monitor requests.
 */
export default $config({
  app(input) {
    return {
      name: "aws-router-waf",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const api = new sst.aws.Function("MyApi", {
      handler: "api.handler",
      url: true,
    });

    const router = new sst.aws.Router("MyRouter", {
      routes: {
        "/*": api.url,
      },
      waf: {
        rateLimitPerIp: 1000,
        managedRules: {
          coreRuleSet: true,
          knownBadInputs: true,
          sqlInjection: true,
        },
        logging: true,
      },
    });

    return {
      url: router.url,
    };
  },
});
