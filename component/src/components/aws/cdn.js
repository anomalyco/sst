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
var __makeTemplateObject = (this && this.__makeTemplateObject) || function (cooked, raw) {
    if (Object.defineProperty) { Object.defineProperty(cooked, "raw", { value: raw }); } else { cooked.raw = raw; }
    return cooked;
};
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Cdn = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var dns_validated_certificate_js_1 = require("./dns-validated-certificate.js");
var https_redirect_js_1 = require("./https-redirect.js");
var provider_js_1 = require("./helpers/provider.js");
var component_js_1 = require("../component.js");
var distribution_deployment_waiter_js_1 = require("./providers/distribution-deployment-waiter.js");
var dns_js_1 = require("./dns.js");
var aws_1 = require("@pulumi/aws");
var naming_js_1 = require("../naming.js");
/**
 * The `Cdn` component is internally used by other components to deploy a CDN to AWS. It uses [Amazon CloudFront](https://aws.amazon.com/cloudfront/) and [Amazon Route 53](https://aws.amazon.com/route53/) to manage custom domains.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * @example
 *
 * You'll find this component exposed in the `transform` of other components. And you can customize the args listed here. For example:
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   transform: {
 *     cdn: (args) => {
 *       args.wait = false;
 *     }
 *   }
 * });
 * ```
 */
var Cdn = /** @class */ (function (_super) {
    __extends(Cdn, _super);
    function Cdn(name, args, opts) {
        var _this = _super.call(this, pulumiType, name, args, opts) || this;
        var parent = _this;
        if (args && "ref" in args) {
            var ref = reference();
            _this.distribution = (0, pulumi_1.output)(ref.distribution);
            _this._domainUrl = ref.distribution.aliases.apply(function (aliases) {
                return (aliases === null || aliases === void 0 ? void 0 : aliases.length) ? "https://".concat(aliases[0]) : undefined;
            });
            return _this;
        }
        var domain = normalizeDomain();
        var certificateArn = createSsl();
        var distribution = createDistribution();
        var waiter = createDistributionDeploymentWaiter();
        createDnsRecords();
        createRedirects();
        _this.distribution = waiter.isDone.apply(function () { return distribution; });
        _this._domainUrl = (domain === null || domain === void 0 ? void 0 : domain.name)
            ? (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["https://", ""], ["https://", ""])), domain.name) : (0, pulumi_1.output)(undefined);
        function reference() {
            var ref = args;
            var distribution = aws_1.cloudfront.Distribution.get("".concat(name, "Distribution"), ref.distributionID, undefined, { parent: parent });
            return { distribution: distribution };
        }
        function normalizeDomain() {
            if (!args.domain)
                return;
            // validate
            (0, pulumi_1.output)(args.domain).apply(function (domain) {
                var _a;
                if (typeof domain === "string")
                    return;
                if (!domain.name)
                    throw new Error("Missing \"name\" for domain.");
                if (domain.dns === false && !domain.cert)
                    throw new Error("Need to provide a validated certificate via \"cert\" when DNS is disabled");
                if (domain.dns === false && ((_a = domain.redirects) === null || _a === void 0 ? void 0 : _a.length))
                    throw new Error("Redirects are not supported when DNS is disabled");
            });
            // normalize
            return (0, pulumi_1.output)(args.domain).apply(function (domain) {
                var _a, _b, _c;
                var norm = typeof domain === "string" ? { name: domain } : domain;
                return {
                    name: norm.name,
                    aliases: (_a = norm.aliases) !== null && _a !== void 0 ? _a : [],
                    redirects: (_b = norm.redirects) !== null && _b !== void 0 ? _b : [],
                    dns: norm.dns === false ? undefined : (_c = norm.dns) !== null && _c !== void 0 ? _c : (0, dns_js_1.dns)(),
                    cert: norm.cert,
                };
            });
        }
        function createSsl() {
            if (!domain)
                return;
            return domain.apply(function (domain) {
                if (domain.cert)
                    return (0, pulumi_1.output)(domain.cert);
                // Certificates used for CloudFront distributions are required to be
                // created in the us-east-1 region
                return new dns_validated_certificate_js_1.DnsValidatedCertificate("".concat(name, "Ssl"), {
                    domainName: domain.name,
                    alternativeNames: domain.aliases,
                    dns: domain.dns,
                }, { parent: parent, provider: (0, provider_js_1.useProvider)("us-east-1") }).arn;
            });
        }
        function createDistribution() {
            var _a;
            var _b;
            return new ((_a = aws_1.cloudfront.Distribution).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.distribution, "".concat(name, "Distribution"), {
                comment: args.comment,
                enabled: true,
                origins: args.origins,
                originGroups: args.originGroups,
                defaultCacheBehavior: args.defaultCacheBehavior,
                orderedCacheBehaviors: args.orderedCacheBehaviors,
                defaultRootObject: args.defaultRootObject,
                customErrorResponses: args.customErrorResponses,
                restrictions: {
                    geoRestriction: {
                        restrictionType: "none",
                    },
                },
                aliases: domain
                    ? (0, pulumi_1.output)(domain).apply(function (domain) { return __spreadArray([
                        domain.name
                    ], domain.aliases, true); })
                    : [],
                viewerCertificate: certificateArn
                    ? {
                        acmCertificateArn: certificateArn,
                        sslSupportMethod: "sni-only",
                        minimumProtocolVersion: "TLSv1.2_2021",
                    }
                    : {
                        cloudfrontDefaultCertificate: true,
                    },
                waitForDeployment: false,
                tags: args.tags,
            }, { parent: parent }), false)))();
        }
        function createDistributionDeploymentWaiter() {
            return (0, pulumi_1.output)(args.wait).apply(function (wait) {
                return new distribution_deployment_waiter_js_1.DistributionDeploymentWaiter("".concat(name, "Waiter"), {
                    distributionId: distribution.id,
                    etag: distribution.etag,
                    wait: wait !== null && wait !== void 0 ? wait : true,
                }, { parent: parent, ignoreChanges: wait ? undefined : ["*"] });
            });
        }
        function createDnsRecords() {
            if (!domain)
                return;
            domain.apply(function (domain) {
                if (!domain.dns)
                    return;
                var existing = [];
                for (var _i = 0, _a = __spreadArray([
                    domain.name
                ], domain.aliases, true).entries(); _i < _a.length; _i++) {
                    var _b = _a[_i], i = _b[0], recordName = _b[1];
                    // Note: The way `dns` is implemented, the logical name for the DNS record is
                    // based on the sanitized version of the record name (ie. logicalName()). This
                    // means the logical name for `*.sst.sh` and `sst.sh` will trash b/c `*.` is
                    // stripped out.
                    // ```
                    // domain: {
                    //   name: "*.sst.sh",
                    //   aliases: ['sst.sh'],
                    // },
                    // ```
                    //
                    // Ideally, we don't sanitize the logical name. But that's a breaking change.
                    //
                    // As a workaround, starting v3.0.79, we prefix the logical name with a unique
                    // index for records with logical names that will trash.
                    var key = (0, naming_js_1.logicalName)(recordName);
                    var namePrefix = existing.includes(key) ? "".concat(name).concat(i) : name;
                    existing.push(key);
                    domain.dns.createAlias(namePrefix, {
                        name: recordName,
                        aliasName: distribution.domainName,
                        aliasZone: distribution.hostedZoneId,
                    }, { parent: parent });
                }
            });
        }
        function createRedirects() {
            if (!domain)
                return;
            (0, pulumi_1.output)(domain).apply(function (domain) {
                if (!domain.redirects.length)
                    return;
                new https_redirect_js_1.HttpsRedirect("".concat(name, "Redirect"), {
                    sourceDomains: domain.redirects,
                    targetDomain: domain.name,
                    cert: domain.cert,
                    dns: domain.dns,
                }, { parent: parent });
            });
        }
        return _this;
    }
    Object.defineProperty(Cdn.prototype, "url", {
        /**
         * The CloudFront URL of the distribution.
         */
        get: function () {
            return (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["https://", ""], ["https://", ""])), this.distribution.domainName);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Cdn.prototype, "domainUrl", {
        /**
         * If the custom domain is enabled, this is the URL of the distribution with the
         * custom domain.
         */
        get: function () {
            return this._domainUrl;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Cdn.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon CloudFront distribution.
                 */
                distribution: this.distribution,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Reference an existing CDN with the given distribution ID. This is useful when
     * you create a Router in one stage and want to share it in another. It avoids having to
     * create a new Router in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share Routers across stages.
     * :::
     *
     * @param name The name of the component.
     * @param distributionID The id of the existing CDN distribution.
     * @param opts? Resource options.
     */
    Cdn.get = function (name, distributionID, opts) {
        return new Cdn(name, {
            ref: true,
            distributionID: distributionID,
        }, opts);
    };
    return Cdn;
}(component_js_1.Component));
exports.Cdn = Cdn;
var pulumiType = "sst:aws:CDN";
// @ts-expect-error
Cdn.__pulumiType = pulumiType;
var templateObject_1, templateObject_2;
