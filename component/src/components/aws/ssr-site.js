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
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype);
    return g.next = verb(0), g["throw"] = verb(1), g["return"] = verb(2), typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
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
exports.SsrSite = void 0;
var path_1 = require("path");
var fs_1 = require("fs");
var glob_1 = require("glob");
var crypto_1 = require("crypto");
var pulumi_1 = require("@pulumi/pulumi");
var cdn_js_1 = require("./cdn.js");
var function_js_1 = require("./function.js");
var bucket_js_1 = require("./bucket.js");
var bucket_files_js_1 = require("./providers/bucket-files.js");
var naming_js_1 = require("../naming.js");
var component_js_1 = require("../component.js");
var error_js_1 = require("../error.js");
var cron_js_1 = require("./cron.js");
var base_site_js_1 = require("../base/base-site.js");
var base_ssr_site_js_1 = require("../base/base-ssr-site.js");
var aws_1 = require("@pulumi/aws");
var kv_keys_js_1 = require("./providers/kv-keys.js");
var provider_js_1 = require("./helpers/provider.js");
var link_js_1 = require("../link.js");
var linkable_js_1 = require("./linkable.js");
var router_js_1 = require("./router.js");
var distribution_invalidation_js_1 = require("./providers/distribution-invalidation.js");
var duration_js_1 = require("../duration.js");
var kv_routes_update_js_1 = require("./providers/kv-routes-update.js");
var supportedRegions = {
    "af-south-1": { lat: -33.9249, lon: 18.4241 }, // Cape Town, South Africa
    "ap-east-1": { lat: 22.3193, lon: 114.1694 }, // Hong Kong
    "ap-northeast-1": { lat: 35.6895, lon: 139.6917 }, // Tokyo, Japan
    "ap-northeast-2": { lat: 37.5665, lon: 126.978 }, // Seoul, South Korea
    "ap-northeast-3": { lat: 34.6937, lon: 135.5023 }, // Osaka, Japan
    "ap-southeast-1": { lat: 1.3521, lon: 103.8198 }, // Singapore
    "ap-southeast-2": { lat: -33.8688, lon: 151.2093 }, // Sydney, Australia
    "ap-southeast-3": { lat: -6.2088, lon: 106.8456 }, // Jakarta, Indonesia
    "ap-southeast-4": { lat: -37.8136, lon: 144.9631 }, // Melbourne, Australia
    "ap-southeast-5": { lat: 3.139, lon: 101.6869 }, // Kuala Lumpur, Malaysia
    "ap-southeast-7": { lat: 13.7563, lon: 100.5018 }, // Bangkok, Thailand
    "ap-south-1": { lat: 19.076, lon: 72.8777 }, // Mumbai, India
    "ap-south-2": { lat: 17.385, lon: 78.4867 }, // Hyderabad, India
    "ca-central-1": { lat: 45.5017, lon: -73.5673 }, // Montreal, Canada
    "ca-west-1": { lat: 51.0447, lon: -114.0719 }, // Calgary, Canada
    "cn-north-1": { lat: 39.9042, lon: 116.4074 }, // Beijing, China
    "cn-northwest-1": { lat: 38.4872, lon: 106.2309 }, // Yinchuan, Ningxia
    "eu-central-1": { lat: 50.1109, lon: 8.6821 }, // Frankfurt, Germany
    "eu-central-2": { lat: 47.3769, lon: 8.5417 }, // Zurich, Switzerland
    "eu-north-1": { lat: 59.3293, lon: 18.0686 }, // Stockholm, Sweden
    "eu-south-1": { lat: 45.4642, lon: 9.19 }, // Milan, Italy
    "eu-south-2": { lat: 40.4168, lon: -3.7038 }, // Madrid, Spain
    "eu-west-1": { lat: 53.3498, lon: -6.2603 }, // Dublin, Ireland
    "eu-west-2": { lat: 51.5074, lon: -0.1278 }, // London, UK
    "eu-west-3": { lat: 48.8566, lon: 2.3522 }, // Paris, France
    "il-central-1": { lat: 32.0853, lon: 34.7818 }, // Tel Aviv, Israel
    "me-central-1": { lat: 25.2048, lon: 55.2708 }, // Dubai, UAE
    "me-south-1": { lat: 26.0667, lon: 50.5577 }, // Manama, Bahrain
    "mx-central-1": { lat: 19.4326, lon: -99.1332 }, // Mexico City, Mexico
    "sa-east-1": { lat: -23.5505, lon: -46.6333 }, // São Paulo, Brazil
    "us-east-1": { lat: 39.0438, lon: -77.4874 }, // Ashburn, VA
    "us-east-2": { lat: 39.9612, lon: -82.9988 }, // Columbus, OH
    "us-gov-east-1": { lat: 38.9696, lon: -77.3861 }, // Herndon, VA
    "us-gov-west-1": { lat: 34.0522, lon: -118.2437 }, // Los Angeles, CA
    "us-west-1": { lat: 37.7749, lon: -122.4194 }, // San Francisco, CA
    "us-west-2": { lat: 45.5122, lon: -122.6587 }, // Portland, OR
};
var SsrSite = /** @class */ (function (_super) {
    __extends(SsrSite, _super);
    function SsrSite(type, name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, type, name, args, opts) || this;
        var self = _this;
        validateDeprecatedProps();
        var regions = normalizeRegions();
        var route = normalizeRoute();
        var edge = normalizeEdge();
        var serverTimeout = normalizeServerTimeout();
        var buildCommand = _this.normalizeBuildCommand(args);
        var sitePath = regions.apply(function () { return normalizeSitePath(); });
        var dev = normalizeDev();
        var purge = (0, pulumi_1.output)(args.assets).apply(function (assets) { var _a; return (_a = assets === null || assets === void 0 ? void 0 : assets.purge) !== null && _a !== void 0 ? _a : false; });
        if (dev.enabled) {
            var server_1 = createDevServer();
            _this.devUrl = dev.url;
            _this.registerOutputs({
                _metadata: {
                    mode: "placeholder",
                    path: sitePath,
                    server: server_1.arn,
                },
                _dev: __assign(__assign({}, dev.outputs), { aws: { role: server_1.nodes.role.arn } }),
            });
            return _this;
        }
        var outputPath = (0, base_ssr_site_js_1.buildApp)(self, name, args, sitePath, buildCommand !== null && buildCommand !== void 0 ? buildCommand : undefined);
        var bucket = createS3Bucket();
        var plan = validatePlan(_this.buildPlan(outputPath, name, args, { bucket: bucket }));
        var timeout = (0, pulumi_1.all)([serverTimeout, plan.server]).apply(function (_a) {
            var _b;
            var argsTimeout = _a[0], plan = _a[1];
            return (_b = argsTimeout !== null && argsTimeout !== void 0 ? argsTimeout : plan === null || plan === void 0 ? void 0 : plan.timeout) !== null && _b !== void 0 ? _b : "20 seconds";
        });
        var servers = createServers();
        var imageOptimizer = createImageOptimizer();
        var assetsUploaded = uploadAssets();
        var kvNamespace = buildKvNamespace();
        var distribution;
        var distributionId;
        var kvStoreArn;
        var invalidationDependsOn = [];
        var prodUrl;
        if (route) {
            kvStoreArn = route.routerKvStoreArn;
            distributionId = route.routerDistributionId;
            invalidationDependsOn = [updateRouterKvRoutes()];
            prodUrl = route.routerUrl;
        }
        else {
            kvStoreArn = createRequestKvStore();
            distribution = createDistribution();
            distributionId = distribution.nodes.distribution.id;
            prodUrl = distribution.domainUrl.apply(function (domainUrl) {
                return (0, pulumi_1.output)(domainUrl !== null && domainUrl !== void 0 ? domainUrl : distribution.url);
            });
        }
        function createCachePolicy() {
            return new aws_1.cloudfront.CachePolicy("".concat(name, "ServerCachePolicy"), {
                comment: "SST server response cache policy",
                defaultTtl: 0,
                maxTtl: 31536000, // 1 year
                minTtl: 0,
                parametersInCacheKeyAndForwardedToOrigin: {
                    cookiesConfig: {
                        cookieBehavior: "none",
                    },
                    headersConfig: {
                        headerBehavior: "whitelist",
                        headers: {
                            items: ["x-open-next-cache-key"],
                        },
                    },
                    queryStringsConfig: {
                        queryStringBehavior: "all",
                    },
                    enableAcceptEncodingBrotli: true,
                    enableAcceptEncodingGzip: true,
                },
            }, { parent: self });
        }
        function createRequestKvStore() {
            return edge.apply(function (edge) {
                var viewerRequest = edge === null || edge === void 0 ? void 0 : edge.viewerRequest;
                if (viewerRequest === null || viewerRequest === void 0 ? void 0 : viewerRequest.kvStore)
                    return (0, pulumi_1.output)(viewerRequest === null || viewerRequest === void 0 ? void 0 : viewerRequest.kvStore);
                return new aws_1.cloudfront.KeyValueStore("".concat(name, "KvStore"), {}, { parent: self }).arn;
            });
        }
        function createRequestFunction() {
            return edge.apply(function (edge) {
                var _a, _b;
                var userInjection = (_b = (_a = edge === null || edge === void 0 ? void 0 : edge.viewerRequest) === null || _a === void 0 ? void 0 : _a.injection) !== null && _b !== void 0 ? _b : "";
                var blockCloudfrontUrlInjection = args.domain
                    ? router_js_1.CF_BLOCK_CLOUDFRONT_URL_INJECTION
                    : "";
                return new aws_1.cloudfront.Function("".concat(name, "CloudfrontFunctionRequest"), {
                    runtime: "cloudfront-js-2.0",
                    keyValueStoreAssociations: kvStoreArn ? [kvStoreArn] : [],
                    code: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["\nimport cf from \"cloudfront\";\nasync function handler(event) {\n  ", "\n  ", "\n  ", "\n\n  const kvNamespace = \"", "\";\n\n  // Load metadata\n  let metadata;\n  try {\n    const v = await cf.kvs().get(kvNamespace + \":metadata\");\n    metadata = JSON.parse(v);\n  } catch (e) {}\n\n  await routeSite(kvNamespace, metadata);\n  return event.request;\n}"], ["\nimport cf from \"cloudfront\";\nasync function handler(event) {\n  ", "\n  ", "\n  ", "\n\n  const kvNamespace = \"", "\";\n\n  // Load metadata\n  let metadata;\n  try {\n    const v = await cf.kvs().get(kvNamespace + \":metadata\");\n    metadata = JSON.parse(v);\n  } catch (e) {}\n\n  await routeSite(kvNamespace, metadata);\n  return event.request;\n}"])), userInjection, blockCloudfrontUrlInjection, router_js_1.CF_ROUTER_INJECTION, kvNamespace),
                }, { parent: self });
            });
        }
        function createResponseFunction() {
            return edge.apply(function (edge) {
                var userConfig = edge === null || edge === void 0 ? void 0 : edge.viewerResponse;
                var userInjection = userConfig === null || userConfig === void 0 ? void 0 : userConfig.injection;
                var kvStoreArn = userConfig === null || userConfig === void 0 ? void 0 : userConfig.kvStore;
                if (!userInjection)
                    return;
                return new aws_1.cloudfront.Function("".concat(name, "CloudfrontFunctionResponse"), {
                    runtime: "cloudfront-js-2.0",
                    keyValueStoreAssociations: kvStoreArn ? [kvStoreArn] : [],
                    code: "\nimport cf from \"cloudfront\";\nasync function handler(event) {\n  ".concat(userInjection, "\n  return event.response;\n}"),
                }, { parent: self });
            });
        }
        function createDistribution() {
            var _a, _b;
            return new (cdn_js_1.Cdn.bind.apply(cdn_js_1.Cdn, __spreadArray([void 0], (0, component_js_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.cdn, "".concat(name, "Cdn"), {
                comment: "".concat(name, " app"),
                domain: args.domain,
                origins: [
                    {
                        originId: "default",
                        domainName: "placeholder.sst.dev",
                        customOriginConfig: {
                            httpPort: 80,
                            httpsPort: 443,
                            originProtocolPolicy: "http-only",
                            originReadTimeout: 20,
                            originSslProtocols: ["TLSv1.2"],
                        },
                    },
                ],
                defaultCacheBehavior: {
                    targetOriginId: "default",
                    viewerProtocolPolicy: "redirect-to-https",
                    allowedMethods: [
                        "DELETE",
                        "GET",
                        "HEAD",
                        "OPTIONS",
                        "PATCH",
                        "POST",
                        "PUT",
                    ],
                    cachedMethods: ["GET", "HEAD"],
                    compress: true,
                    cachePolicyId: (_b = args.cachePolicy) !== null && _b !== void 0 ? _b : createCachePolicy().id,
                    // CloudFront's Managed-AllViewerExceptHostHeader policy
                    originRequestPolicyId: "b689b0a8-53d0-40ab-baf2-68738e2966ac",
                    functionAssociations: (0, pulumi_1.all)([
                        createRequestFunction(),
                        createResponseFunction(),
                    ]).apply(function (_a) {
                        var reqFn = _a[0], resFn = _a[1];
                        return __spreadArray([
                            { eventType: "viewer-request", functionArn: reqFn.arn }
                        ], (resFn
                            ? [{ eventType: "viewer-response", functionArn: resFn.arn }]
                            : []), true);
                    }),
                },
            }, { parent: self }), false)))();
        }
        var kvUpdated = createKvEntries();
        createInvalidation();
        var server = servers.apply(function (servers) { var _a; return (_a = servers[0]) === null || _a === void 0 ? void 0 : _a.server; });
        _this.bucket = bucket;
        _this.cdn = distribution;
        _this.server = server;
        _this.prodUrl = prodUrl;
        _this.registerOutputs({
            _hint: _this.url,
            _metadata: {
                mode: "deployed",
                path: sitePath,
                url: _this.url,
                edge: false,
                server: server.arn,
            },
            _dev: __assign(__assign({}, dev.outputs), { aws: { role: server.nodes.role.arn } }),
        });
        function validateDeprecatedProps() {
            if (args.cdn !== undefined)
                throw new error_js_1.VisibleError("\"cdn\" prop is deprecated. Use the \"route.router\" prop instead to use an existing \"Router\" component to serve your site.");
        }
        function normalizeDev() {
            var _a, _b, _c, _d;
            var enabled = $dev && args.dev !== false;
            var devArgs = args.dev || {};
            return {
                enabled: enabled,
                url: (0, pulumi_1.output)((_a = devArgs.url) !== null && _a !== void 0 ? _a : linkable_js_1.URL_UNAVAILABLE),
                outputs: {
                    title: devArgs.title,
                    command: (0, pulumi_1.output)((_b = devArgs.command) !== null && _b !== void 0 ? _b : "npm run dev"),
                    autostart: (0, pulumi_1.output)((_c = devArgs.autostart) !== null && _c !== void 0 ? _c : true),
                    directory: (0, pulumi_1.output)((_d = devArgs.directory) !== null && _d !== void 0 ? _d : sitePath),
                    environment: args.environment,
                    links: (0, pulumi_1.output)(args.link || [])
                        .apply(link_js_1.Link.build)
                        .apply(function (links) { return links.map(function (link) { return link.name; }); }),
                },
            };
        }
        function normalizeSitePath() {
            return (0, pulumi_1.output)(args.path).apply(function (sitePath) {
                if (!sitePath)
                    return ".";
                if (!fs_1.default.existsSync(sitePath)) {
                    throw new error_js_1.VisibleError("Site directory not found at \"".concat(path_1.default.resolve(sitePath), "\". Please check the path setting in your configuration."));
                }
                return sitePath;
            });
        }
        function normalizeRegions() {
            var _a;
            return (0, pulumi_1.output)((_a = args.regions) !== null && _a !== void 0 ? _a : [(0, aws_1.getRegionOutput)(undefined, { parent: self }).name]).apply(function (regions) {
                if (regions.length === 0)
                    throw new error_js_1.VisibleError("No deployment regions specified. Please specify at least one region in the 'regions' property.");
                return regions.map(function (region) {
                    if ([
                        "ap-south-2",
                        "ap-southeast-4",
                        "ap-southeast-5",
                        "ca-west-1",
                        "eu-south-2",
                        "eu-central-2",
                        "il-central-1",
                        "me-central-1",
                    ].includes(region))
                        throw new error_js_1.VisibleError("Region ".concat(region, " is not supported by this component. Please select a different AWS region."));
                    if (!Object.values(aws_1.Region).includes(region))
                        throw new error_js_1.VisibleError("Invalid AWS region: \"".concat(region, "\". Please specify a valid AWS region."));
                    return region;
                });
            });
        }
        function normalizeRoute() {
            var route = (0, router_js_1.normalizeRouteArgs)(args.router, args.route);
            if (route) {
                if (args.domain)
                    throw new error_js_1.VisibleError("Cannot provide both \"domain\" and \"route\". Use the \"domain\" prop on the \"Router\" component when serving your site through a Router.");
                if (args.edge)
                    throw new error_js_1.VisibleError("Cannot provide both \"edge\" and \"route\". Use the \"edge\" prop on the \"Router\" component when serving your site through a Router.");
            }
            return route;
        }
        function normalizeEdge() {
            var _a;
            return (0, pulumi_1.output)([args.edge, (_a = args.server) === null || _a === void 0 ? void 0 : _a.edge]).apply(function (_a) {
                var edge = _a[0], serverEdge = _a[1];
                if (serverEdge)
                    throw new error_js_1.VisibleError("The \"server.edge\" prop is deprecated. Use the \"edge\" prop on the top level instead.");
                if (!edge)
                    return edge;
                return edge;
            });
        }
        function normalizeServerTimeout() {
            var _a;
            return (0, pulumi_1.output)((_a = args.server) === null || _a === void 0 ? void 0 : _a.timeout).apply(function (v) {
                if (!v)
                    return v;
                var seconds = (0, duration_js_1.toSeconds)(v);
                if (seconds > 60)
                    throw new error_js_1.VisibleError("Server timeout for \"".concat(name, "\" cannot be greater than 60 seconds."));
                return v;
            });
        }
        function createDevServer() {
            var _a;
            return new (function_js_1.Function.bind.apply(function_js_1.Function, __spreadArray([void 0], (0, component_js_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.server, "".concat(name, "DevServer"), {
                description: "".concat(name, " dev server"),
                runtime: "nodejs20.x",
                timeout: "20 seconds",
                memory: "128 MB",
                bundle: path_1.default.join($cli.paths.platform, "functions", "empty-function"),
                handler: "index.handler",
                environment: args.environment,
                permissions: args.permissions,
                link: args.link,
                dev: false,
            }, { parent: self }), false)))();
        }
        function validatePlan(plan) {
            return (0, pulumi_1.all)([plan, route]).apply(function (_a) {
                var plan = _a[0], route = _a[1];
                if (plan.base) {
                    // starts with /
                    plan.base = !plan.base.startsWith("/") ? "/".concat(plan.base) : plan.base;
                    // does not end with /
                    plan.base = plan.base.replace(/\/$/, "");
                }
                if ((route === null || route === void 0 ? void 0 : route.pathPrefix) && route.pathPrefix !== "/") {
                    if (!plan.base)
                        throw new error_js_1.VisibleError("No base path found for site. You must configure the base path to match the route path prefix \"".concat(route.pathPrefix, "\"."));
                    if (!plan.base.startsWith(route.pathPrefix))
                        throw new error_js_1.VisibleError("The site base path \"".concat(plan.base, "\" must start with the route path prefix \"").concat(route.pathPrefix, "\"."));
                }
                // if copy.to has a leading slash, files will be uploaded to `/` folder in bucket
                plan.assets.forEach(function (copy) {
                    copy.to = copy.to.replace(/^\/|\/$/g, "");
                });
                if (plan.isrCache) {
                    plan.isrCache.to = plan.isrCache.to.replace(/^\/|\/$/g, "");
                }
                return plan;
            });
        }
        function createS3Bucket() {
            var _a;
            return new (bucket_js_1.Bucket.bind.apply(bucket_js_1.Bucket, __spreadArray([void 0], (0, component_js_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.assets, "".concat(name, "Assets"), { access: "cloudfront" }, { parent: self, retainOnDelete: false }), false)))();
        }
        function createServers() {
            return (0, pulumi_1.all)([regions, plan.server]).apply(function (_a) {
                var regions = _a[0], planServer = _a[1];
                if (!planServer)
                    return [];
                return regions.map(function (region) {
                    var _a, _b, _c, _d, _e, _f, _g, _h;
                    var provider = (0, provider_js_1.useProvider)(region);
                    var server = new (function_js_1.Function.bind.apply(function_js_1.Function, __spreadArray([void 0], (0, component_js_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.server, "".concat(name, "Server").concat((0, naming_js_1.logicalName)(region)), __assign(__assign({}, planServer), { description: (_b = planServer.description) !== null && _b !== void 0 ? _b : "".concat(name, " server"), runtime: (0, pulumi_1.output)((_c = args.server) === null || _c === void 0 ? void 0 : _c.runtime).apply(function (v) { var _a; return (_a = v !== null && v !== void 0 ? v : planServer.runtime) !== null && _a !== void 0 ? _a : "nodejs20.x"; }), timeout: timeout, memory: (0, pulumi_1.output)((_d = args.server) === null || _d === void 0 ? void 0 : _d.memory).apply(function (v) { var _a; return (_a = v !== null && v !== void 0 ? v : planServer.memory) !== null && _a !== void 0 ? _a : "1024 MB"; }), architecture: (0, pulumi_1.output)((_e = args.server) === null || _e === void 0 ? void 0 : _e.architecture).apply(function (v) { var _a; return (_a = v !== null && v !== void 0 ? v : planServer.architecture) !== null && _a !== void 0 ? _a : "x86_64"; }), vpc: args.vpc, nodejs: __assign({ format: "esm", install: (_f = args.server) === null || _f === void 0 ? void 0 : _f.install, loader: (_g = args.server) === null || _g === void 0 ? void 0 : _g.loader }, planServer.nodejs), environment: (0, pulumi_1.output)(args.environment).apply(function (environment) { return (__assign(__assign({}, environment), planServer.environment)); }), permissions: (0, pulumi_1.output)(args.permissions).apply(function (permissions) {
                            var _a;
                            return __spreadArray(__spreadArray([
                                {
                                    actions: ["cloudfront:CreateInvalidation"],
                                    resources: ["*"],
                                }
                            ], (permissions !== null && permissions !== void 0 ? permissions : []), true), ((_a = planServer.permissions) !== null && _a !== void 0 ? _a : []), true);
                        }), injections: __spreadArray(__spreadArray([], (args.warm
                            ? [useServerWarmingInjection(planServer.streaming)]
                            : []), true), (planServer.injections || []), true), link: (0, pulumi_1.output)(args.link).apply(function (link) {
                            var _a;
                            return __spreadArray(__spreadArray([], ((_a = planServer.link) !== null && _a !== void 0 ? _a : []), true), (link !== null && link !== void 0 ? link : []), true);
                        }), layers: (0, pulumi_1.output)((_h = args.server) === null || _h === void 0 ? void 0 : _h.layers).apply(function (layers) {
                            var _a;
                            return __spreadArray(__spreadArray([], ((_a = planServer.layers) !== null && _a !== void 0 ? _a : []), true), (layers !== null && layers !== void 0 ? layers : []), true);
                        }), url: true, dev: false, _skipHint: true }), { provider: provider, parent: self }), false)))();
                    if (args.warm) {
                        // Create cron job
                        var cron = new cron_js_1.Cron("".concat(name, "Warmer").concat((0, naming_js_1.logicalName)(region)), {
                            schedule: "rate(5 minutes)",
                            job: {
                                description: "".concat(name, " warmer"),
                                bundle: path_1.default.join($cli.paths.platform, "dist", "ssr-warmer"),
                                runtime: "nodejs20.x",
                                handler: "index.handler",
                                timeout: "900 seconds",
                                memory: "128 MB",
                                dev: false,
                                environment: {
                                    FUNCTION_NAME: server.nodes.function.name,
                                    CONCURRENCY: (0, pulumi_1.output)(args.warm).apply(function (warm) {
                                        return warm.toString();
                                    }),
                                },
                                link: [server],
                                _skipMetadata: true,
                            },
                            transform: {
                                target: function (args) {
                                    args.retryPolicy = {
                                        maximumRetryAttempts: 0,
                                        maximumEventAgeInSeconds: 60,
                                    };
                                },
                            },
                        }, { provider: provider, parent: self });
                        // Prewarm on deploy
                        new aws_1.lambda.Invocation("".concat(name, "Prewarm").concat((0, naming_js_1.logicalName)(region)), {
                            functionName: cron.nodes.job.name,
                            triggers: {
                                version: Date.now().toString(),
                            },
                            input: JSON.stringify({}),
                        }, { provider: provider, parent: self });
                    }
                    return { region: region, server: server };
                });
            });
        }
        function createImageOptimizer() {
            return (0, pulumi_1.output)(plan.imageOptimizer).apply(function (imageOptimizer) {
                if (!imageOptimizer)
                    return;
                return new function_js_1.Function("".concat(name, "ImageOptimizer"), __assign(__assign({ timeout: "25 seconds", logging: {
                        retention: "3 days",
                    }, permissions: [
                        {
                            actions: ["s3:GetObject"],
                            resources: [(0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["", "/*"], ["", "/*"])), bucket.arn)],
                        },
                    ] }, imageOptimizer.function), { url: true, dev: false, _skipMetadata: true, _skipHint: true }), { parent: self });
            });
        }
        function useServerWarmingInjection(streaming) {
            return __spreadArray(__spreadArray([
                "if (event.type === \"warmer\") {",
                "  const p = new Promise((resolve) => {",
                "    setTimeout(() => {",
                "      resolve({ serverId: \"server-\" + Math.random().toString(36).slice(2, 8) });",
                "    }, event.delay);",
                "  });"
            ], (streaming
                ? [
                    "  const response = await p;",
                    "  responseStream.write(JSON.stringify(response));",
                    "  responseStream.end();",
                    "  return;",
                ]
                : ["  return p;"]), true), [
                "}",
            ], false).join("\n");
        }
        function uploadAssets() {
            var _this = this;
            return (0, pulumi_1.all)([args.assets, route, plan, outputPath]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
                var versionedFilesTTL, nonVersionedFilesTTL, bucketFiles, _loop_1, _i, _c, copy;
                var _this = this;
                var _d, _e, _f;
                var assets = _b[0], route = _b[1], plan = _b[2], outputPath = _b[3];
                return __generator(this, function (_g) {
                    switch (_g.label) {
                        case 0:
                            versionedFilesTTL = 31536000;
                            nonVersionedFilesTTL = 86400;
                            bucketFiles = [];
                            _loop_1 = function (copy) {
                                var fileOptions, filesUploaded, _loop_2, _h, _j, fileOption;
                                return __generator(this, function (_k) {
                                    switch (_k.label) {
                                        case 0:
                                            fileOptions = __spreadArray(__spreadArray([
                                                // unversioned files
                                                {
                                                    files: "**",
                                                    ignore: copy.versionedSubDir
                                                        ? path_1.default.posix.join(copy.versionedSubDir, "**")
                                                        : undefined,
                                                    cacheControl: (_d = assets === null || assets === void 0 ? void 0 : assets.nonVersionedFilesCacheHeader) !== null && _d !== void 0 ? _d : "public,max-age=0,s-maxage=".concat(nonVersionedFilesTTL, ",stale-while-revalidate=").concat(nonVersionedFilesTTL),
                                                }
                                            ], (copy.versionedSubDir
                                                ? [
                                                    {
                                                        files: path_1.default.posix.join(copy.versionedSubDir, "**"),
                                                        cacheControl: (_e = assets === null || assets === void 0 ? void 0 : assets.versionedFilesCacheHeader) !== null && _e !== void 0 ? _e : "public,max-age=".concat(versionedFilesTTL, ",immutable"),
                                                    },
                                                ]
                                                : []), true), ((_f = assets === null || assets === void 0 ? void 0 : assets.fileOptions) !== null && _f !== void 0 ? _f : []), true);
                                            filesUploaded = [];
                                            _loop_2 = function (fileOption) {
                                                var files, _l, _m, _o;
                                                return __generator(this, function (_p) {
                                                    switch (_p.label) {
                                                        case 0:
                                                            files = (0, glob_1.globSync)(fileOption.files, {
                                                                cwd: path_1.default.resolve(outputPath, copy.from),
                                                                nodir: true,
                                                                dot: true,
                                                                ignore: fileOption.ignore,
                                                            }).filter(function (file) { return !filesUploaded.includes(file); });
                                                            _m = (_l = bucketFiles.push).apply;
                                                            _o = [bucketFiles];
                                                            return [4 /*yield*/, Promise.all(files.map(function (file) { return __awaiter(_this, void 0, void 0, function () {
                                                                    var source, content, hash;
                                                                    var _a, _b, _c;
                                                                    return __generator(this, function (_d) {
                                                                        switch (_d.label) {
                                                                            case 0:
                                                                                source = path_1.default.resolve(outputPath, copy.from, file);
                                                                                return [4 /*yield*/, fs_1.default.promises.readFile(source, "utf-8")];
                                                                            case 1:
                                                                                content = _d.sent();
                                                                                hash = crypto_1.default
                                                                                    .createHash("sha256")
                                                                                    .update(content)
                                                                                    .digest("hex");
                                                                                return [2 /*return*/, {
                                                                                        source: source,
                                                                                        key: path_1.default.posix.join(copy.to, (_b = (_a = route === null || route === void 0 ? void 0 : route.pathPrefix) === null || _a === void 0 ? void 0 : _a.replace(/^\//, "")) !== null && _b !== void 0 ? _b : "", file),
                                                                                        hash: hash,
                                                                                        cacheControl: fileOption.cacheControl,
                                                                                        contentType: (_c = fileOption.contentType) !== null && _c !== void 0 ? _c : (0, base_site_js_1.getContentType)(file, "UTF-8"),
                                                                                    }];
                                                                        }
                                                                    });
                                                                }); }))];
                                                        case 1:
                                                            _m.apply(_l, _o.concat([(_p.sent())]));
                                                            filesUploaded.push.apply(filesUploaded, files);
                                                            return [2 /*return*/];
                                                    }
                                                });
                                            };
                                            _h = 0, _j = fileOptions.reverse();
                                            _k.label = 1;
                                        case 1:
                                            if (!(_h < _j.length)) return [3 /*break*/, 4];
                                            fileOption = _j[_h];
                                            return [5 /*yield**/, _loop_2(fileOption)];
                                        case 2:
                                            _k.sent();
                                            _k.label = 3;
                                        case 3:
                                            _h++;
                                            return [3 /*break*/, 1];
                                        case 4: return [2 /*return*/];
                                    }
                                });
                            };
                            _i = 0, _c = __spreadArray(__spreadArray([], plan.assets, true), (plan.isrCache
                                ? [__assign(__assign({}, plan.isrCache), { versionedSubDir: undefined })]
                                : []), true);
                            _g.label = 1;
                        case 1:
                            if (!(_i < _c.length)) return [3 /*break*/, 4];
                            copy = _c[_i];
                            return [5 /*yield**/, _loop_1(copy)];
                        case 2:
                            _g.sent();
                            _g.label = 3;
                        case 3:
                            _i++;
                            return [3 /*break*/, 1];
                        case 4: return [2 /*return*/, new bucket_files_js_1.BucketFiles("".concat(name, "AssetFiles"), {
                                bucketName: bucket.name,
                                files: bucketFiles,
                                purge: purge,
                                region: (0, aws_1.getRegionOutput)(undefined, { parent: self }).name,
                            }, { parent: self })];
                    }
                });
            }); });
        }
        function buildKvNamespace() {
            // In the case multiple sites use the same kv store, we need to namespace the keys
            return crypto_1.default
                .createHash("md5")
                .update("".concat($app.name, "-").concat($app.stage, "-").concat(name))
                .digest("hex")
                .substring(0, 4);
        }
        function createKvEntries() {
            var entries = (0, pulumi_1.all)([
                servers,
                imageOptimizer,
                outputPath,
                plan,
                bucket.nodes.bucket.bucketRegionalDomainName,
                timeout,
            ]).apply(function (_a) {
                var servers = _a[0], imageOptimizer = _a[1], outputPath = _a[2], plan = _a[3], bucketDomain = _a[4], timeout = _a[5];
                return (0, pulumi_1.all)([
                    servers.map(function (s) { return ({ region: s.region, url: s.server.url }); }),
                    imageOptimizer === null || imageOptimizer === void 0 ? void 0 : imageOptimizer.url,
                ]).apply(function (_a) {
                    var servers = _a[0], imageOptimizerUrl = _a[1];
                    var kvEntries = {};
                    var dirs = [];
                    plan.assets.forEach(function (copy) {
                        fs_1.default.readdirSync(path_1.default.join(outputPath, copy.from), {
                            withFileTypes: true,
                        }).forEach(function (item) {
                            if (item.isFile()) {
                                kvEntries[path_1.default.posix.join("/", item.name)] = "s3";
                                return;
                            }
                            // Handle deep routes
                            // In Next.js, asset requests are prefixed with is /_next/static, and
                            // image optimization requests are prefixed with /_next/image. We cannot
                            // route by 1 level of subdirs (ie. /_next/`), so we need to route by 2
                            // levels of subdirs.
                            if (!copy.deepRoute) {
                                dirs.push(path_1.default.posix.join("/", item.name));
                                return;
                            }
                            fs_1.default.readdirSync(path_1.default.join(outputPath, copy.from, item.name), {
                                withFileTypes: true,
                            }).forEach(function (subItem) {
                                if (subItem.isFile()) {
                                    kvEntries[path_1.default.posix.join("/", item.name, subItem.name)] =
                                        "s3";
                                    return;
                                }
                                dirs.push(path_1.default.posix.join("/", item.name, subItem.name));
                            });
                        });
                    });
                    kvEntries["metadata"] = JSON.stringify({
                        base: plan.base,
                        custom404: plan.custom404,
                        s3: {
                            domain: bucketDomain,
                            dir: plan.assets[0].to ? "/" + plan.assets[0].to : "",
                            routes: dirs,
                        },
                        image: imageOptimizerUrl
                            ? {
                                host: new URL(imageOptimizerUrl).host,
                                route: plan.imageOptimizer.prefix,
                            }
                            : undefined,
                        servers: servers.map(function (s) { return [
                            new URL(s.url).host,
                            supportedRegions[s.region].lat,
                            supportedRegions[s.region].lon,
                        ]; }),
                        origin: {
                            timeouts: {
                                readTimeout: (0, duration_js_1.toSeconds)(timeout),
                            },
                        },
                    });
                    return kvEntries;
                });
            });
            return new kv_keys_js_1.KvKeys("".concat(name, "KvKeys"), {
                store: kvStoreArn,
                namespace: kvNamespace,
                entries: entries,
                purge: purge,
            }, { parent: self });
        }
        function updateRouterKvRoutes() {
            return new kv_routes_update_js_1.KvRoutesUpdate("".concat(name, "RoutesUpdate"), {
                store: route.routerKvStoreArn,
                namespace: route.routerKvNamespace,
                key: "routes",
                entry: route.apply(function (route) {
                    return ["site", kvNamespace, route.hostPattern, route.pathPrefix].join(",");
                }),
            }, { parent: self });
        }
        function createInvalidation() {
            (0, pulumi_1.all)([args.invalidation, outputPath, plan]).apply(function (_a) {
                var invalidationRaw = _a[0], outputPath = _a[1], plan = _a[2];
                // Normalize invalidation
                if (invalidationRaw === false)
                    return;
                var invalidation = __assign({ wait: false, paths: "all" }, invalidationRaw);
                // We will generate a hash based on the contents of the S3 files with cache enabled.
                // This will be used to determine if we need to invalidate our CloudFront cache.
                var s3Origin = plan.assets;
                var cachedS3Files = s3Origin.filter(function (file) { return file.cached; });
                if (cachedS3Files.length === 0)
                    return;
                // Build invalidation paths
                var invalidationPaths = [];
                if (invalidation.paths === "all") {
                    invalidationPaths.push("/*");
                }
                else if (invalidation.paths === "versioned") {
                    cachedS3Files.forEach(function (item) {
                        if (!item.versionedSubDir)
                            return;
                        invalidationPaths.push(path_1.default.posix.join("/", item.to, item.versionedSubDir, "*"));
                    });
                }
                else {
                    invalidationPaths.push.apply(invalidationPaths, ((invalidation === null || invalidation === void 0 ? void 0 : invalidation.paths) || []));
                }
                if (invalidationPaths.length === 0)
                    return;
                // Build build ID
                var invalidationBuildId;
                if (plan.buildId) {
                    invalidationBuildId = plan.buildId;
                }
                else {
                    var hash_1 = crypto_1.default.createHash("md5");
                    cachedS3Files.forEach(function (item) {
                        // The below options are needed to support following symlinks when building zip files:
                        // - nodir: This will prevent symlinks themselves from being copied into the zip.
                        // - follow: This will follow symlinks and copy the files within.
                        // For versioned files, use file path for digest since file version in name should change on content change
                        if (item.versionedSubDir) {
                            (0, glob_1.globSync)("**", {
                                dot: true,
                                nodir: true,
                                follow: true,
                                cwd: path_1.default.resolve(outputPath, item.from, item.versionedSubDir),
                            }).forEach(function (filePath) { return hash_1.update(filePath); });
                        }
                        // For non-versioned files, use file content for digest
                        if (invalidation.paths !== "versioned") {
                            (0, glob_1.globSync)("**", {
                                ignore: item.versionedSubDir
                                    ? [path_1.default.posix.join(item.versionedSubDir, "**")]
                                    : undefined,
                                dot: true,
                                nodir: true,
                                follow: true,
                                cwd: path_1.default.resolve(outputPath, item.from),
                            }).forEach(function (filePath) {
                                return hash_1.update(fs_1.default.readFileSync(path_1.default.resolve(outputPath, item.from, filePath), "utf-8"));
                            });
                        }
                    });
                    invalidationBuildId = hash_1.digest("hex");
                }
                new distribution_invalidation_js_1.DistributionInvalidation("".concat(name, "Invalidation"), {
                    distributionId: distributionId,
                    paths: invalidationPaths,
                    version: invalidationBuildId,
                    wait: invalidation.wait,
                }, {
                    parent: self,
                    dependsOn: __spreadArray([assetsUploaded, kvUpdated], invalidationDependsOn, true),
                });
            });
        }
        return _this;
    }
    Object.defineProperty(SsrSite.prototype, "url", {
        /**
         * The URL of the Astro site.
         *
         * If the `domain` is set, this is the URL with the custom domain.
         * Otherwise, it's the auto-generated CloudFront URL.
         */
        get: function () {
            return (0, pulumi_1.all)([this.prodUrl, this.devUrl]).apply(function (_a) {
                var prodUrl = _a[0], devUrl = _a[1];
                return (prodUrl !== null && prodUrl !== void 0 ? prodUrl : devUrl);
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(SsrSite.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The AWS Lambda server function that renders the site.
                 */
                server: this.server,
                /**
                 * The Amazon S3 Bucket that stores the assets.
                 */
                assets: this.bucket,
                /**
                 * The Amazon CloudFront CDN that serves the site.
                 */
                cdn: this.cdn,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    SsrSite.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
            },
        };
    };
    return SsrSite;
}(component_js_1.Component));
exports.SsrSite = SsrSite;
var templateObject_1, templateObject_2;
