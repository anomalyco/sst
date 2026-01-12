/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## CloudFront Web Application Firewall (WAF)
 *
 * Enable WAF for a CloudFront distribution created by `sst.aws.Nextjs`.
 *
 * The WAF is configured using AWS managed rules and is attached to the
 * CloudFront distribution at creation time.
 *
 */
export default $config({
    app(input) {
        return {
            name: "sst-v3",
            home: "aws",
            removal: input?.stage === "production" ? "retain" : "remove",
        };
    },
    async run() {
        const webAcl = new aws.wafv2.WebAcl("WebAcl", {
            scope: "CLOUDFRONT",
            defaultAction: { allow: {} },

            visibilityConfig: {
                cloudwatchMetricsEnabled: true,
                metricName: "web-acl",
                sampledRequestsEnabled: true,
            },

            rules: [
                {
                    name: "AWSManagedRules",
                    priority: 0,
                    overrideAction: { none: {} },
                    statement: {
                        managedRuleGroupStatement: {
                            vendorName: "AWS",
                            name: "AWSManagedRulesCommonRuleSet",
                        },
                    },
                    visibilityConfig: {
                        cloudwatchMetricsEnabled: true,
                        metricName: "managed-rules",
                        sampledRequestsEnabled: true,
                    },
                },
            ],
        })

        const site = new sst.aws.Nextjs("NextjsSite", {
            path: "web/",
            transform: {
                cdn: {
                    transform: {
                        distribution(args) {
                            args.webAclId = webAcl.arn
                        },
                    },
                },
            },
        })

        return {
            web: site.url,
        };
    },
});