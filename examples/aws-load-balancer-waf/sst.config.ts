/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Load Balancer Web Application Firewall (WAF)
 *
 * Enable WAF for an AWS Load Balancer.
 *
 * The WAF is configured to enable a rate limit and enables AWS managed rules.
 *
 */
export default $config({
  app(input) {
    return {
      name: "aws-load-balancer-waf",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MyVpc");
    const cluster = new sst.aws.Cluster("MyCluster", { vpc });
    const service = cluster.addService("MyAppService", {
      image: {
        context: "./",
        dockerfile: "packages/server/Dockerfile",
      },
    });

    const rateLimitRule = {
      name: "RateLimitRule",
      statement: {
        rateBasedStatement: {
          limit: 200,
          aggregateKeyType: "IP",
        },
      },
      priority: 1,
      action: { block: {} },
      visibilityConfig: {
        cloudwatchMetricsEnabled: true,
        sampledRequestsEnabled: true,
        metricName: "MyAppRateLimitRule",
      },
    };

    const awsManagedRules = {
      name: "AWSManagedRules",
      statement: {
        managedRuleGroupStatement: {
          name: "AWSManagedRulesCommonRuleSet",
          vendorName: "AWS",
        },
      },
      priority: 2,
      overrideAction: {
        none: {},
      },
      visibilityConfig: {
        cloudwatchMetricsEnabled: true,
        sampledRequestsEnabled: true,
        metricName: "MyAppAWSManagedRules",
      },
    };

    const webAcl = new aws.wafv2.WebAcl("AppAlbWebAcl", {
      defaultAction: { allow: {} },
      scope: "REGIONAL",
      visibilityConfig: {
        cloudwatchMetricsEnabled: true,
        sampledRequestsEnabled: true,
        metricName: "AppAlbWebAcl",
      },
      rules: [rateLimitRule, awsManagedRules],
    });

    service.nodes.loadBalancer.arn.apply((arn) => {
      new aws.wafv2.WebAclAssociation("MyAppAlbWebAclAssociation", {
        resourceArn: arn,
        webAclArn: webAcl.arn,
      });
    });

    return {};
  },
});
