"use strict";
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (Object.prototype.hasOwnProperty.call(b, p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        if (typeof b !== "function" && b !== null)
            throw new TypeError("Class extends value " + String(b) + " is not a constructor or null");
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.HttpsRedirect = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var dns_validated_certificate_js_1 = require("./dns-validated-certificate.js");
var bucket_js_1 = require("./bucket.js");
var component_js_1 = require("../component.js");
var provider_js_1 = require("./helpers/provider.js");
var aws_1 = require("@pulumi/aws");
/**
 * Allows creating a domainA -> domainB redirect using CloudFront and S3.
 * You can specify multiple domains to be redirected.
 */
var HttpsRedirect = /** @class */ (function (_super) {
    __extends(HttpsRedirect, _super);
    function HttpsRedirect(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var certificateArn = createSsl();
        var bucket = new bucket_js_1.Bucket("".concat(name, "Bucket"), {}, { parent: parent });
        var bucketWebsite = new aws_1.s3.BucketWebsiteConfigurationV2("".concat(name, "BucketWebsite"), {
            bucket: bucket.name,
            redirectAllRequestsTo: {
                hostName: args.targetDomain,
                protocol: "https",
            },
        }, { parent: parent });
        var distribution = new aws_1.cloudfront.Distribution("".concat(name, "Distribution"), {
            enabled: true,
            waitForDeployment: false,
            aliases: args.sourceDomains,
            restrictions: {
                geoRestriction: {
                    restrictionType: "none",
                },
            },
            comment: (0, pulumi_1.all)([args.targetDomain, args.sourceDomains]).apply(function (_a) {
                var targetDomain = _a[0], sourceDomains = _a[1];
                var comment = "Redirect to ".concat(targetDomain, " from ").concat(sourceDomains.join(", "));
                return comment.length > 128
                    ? comment.slice(0, 125) + "..."
                    : comment;
            }),
            priceClass: "PriceClass_All",
            viewerCertificate: {
                acmCertificateArn: certificateArn,
                sslSupportMethod: "sni-only",
            },
            defaultCacheBehavior: {
                allowedMethods: ["GET", "HEAD", "OPTIONS"],
                targetOriginId: "s3Origin",
                viewerProtocolPolicy: "redirect-to-https",
                cachedMethods: ["GET", "HEAD"],
                forwardedValues: {
                    cookies: { forward: "none" },
                    queryString: false,
                },
            },
            origins: [
                {
                    originId: "s3Origin",
                    domainName: bucketWebsite.websiteEndpoint,
                    customOriginConfig: {
                        httpPort: 80,
                        httpsPort: 443,
                        originProtocolPolicy: "http-only",
                        originSslProtocols: ["TLSv1.2"],
                    },
                },
            ],
        }, { parent: parent });
        (0, pulumi_1.all)([args.dns, args.sourceDomains]).apply(function (_a) {
            var dns = _a[0], sourceDomains = _a[1];
            for (var _i = 0, sourceDomains_1 = sourceDomains; _i < sourceDomains_1.length; _i++) {
                var recordName = sourceDomains_1[_i];
                dns.createAlias(name, {
                    name: recordName,
                    aliasName: distribution.domainName,
                    aliasZone: distribution.hostedZoneId,
                }, { parent: parent });
            }
        });
        function createSsl() {
            if (args.cert)
                return args.cert;
            return new dns_validated_certificate_js_1.DnsValidatedCertificate("".concat(name, "Ssl"), {
                domainName: (0, pulumi_1.output)(args.sourceDomains).apply(function (domains) { return domains[0]; }),
                alternativeNames: (0, pulumi_1.output)(args.sourceDomains).apply(function (domains) {
                    return domains.slice(1);
                }),
                dns: args.dns,
            }, { parent: parent, provider: (0, provider_js_1.useProvider)("us-east-1") }).arn;
        }
        return _this;
    }
    return HttpsRedirect;
}(component_js_1.Component));
exports.HttpsRedirect = HttpsRedirect;
var __pulumiType = "sst:aws:HttpsRedirect";
// @ts-expect-error
HttpsRedirect.__pulumiType = __pulumiType;
