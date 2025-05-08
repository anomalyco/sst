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
exports.CF_ROUTER_INJECTION = exports.CF_BLOCK_CLOUDFRONT_URL_INJECTION = exports.Router = void 0;
exports.normalizeRouteArgs = normalizeRouteArgs;
var pulumi_1 = require("@pulumi/pulumi");
var crypto_1 = require("crypto");
var component_1 = require("../component");
var cdn_1 = require("./cdn");
var aws_1 = require("@pulumi/aws");
var naming_1 = require("../naming");
var bucket_1 = require("./bucket");
var origin_access_control_1 = require("./providers/origin-access-control");
var error_1 = require("../error");
var router_url_route_1 = require("./router-url-route");
var router_bucket_route_1 = require("./router-bucket-route");
/**
 * The `Router` component lets you use a CloudFront distribution to direct
 * requests to various parts of your application like:
 *
 * - A URL
 * - A function
 * - A frontend
 * - An S3 bucket
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Router("MyRouter");
 * ```
 *
 * #### Add a custom domain
 *
 * ```ts {2} title="sst.config.ts"
 * new sst.aws.Router("MyRouter", {
 *   domain: "myapp.com"
 * });
 * ```
 *
 * #### Sharing the router across stages
 *
 * ```ts title="sst.config.ts"
 * const router = $app.stage === "production"
 *   ? new sst.aws.Router("MyRouter", {
 *       domain: {
 *         name: "example.com",
 *         aliases: ["*.example.com"]
 *       }
 *     })
 *   : sst.aws.Router.get("MyRouter", "E1XWRGCYGTFB7Z");
 * ```
 *
 * #### Route to a URL
 *
 * ```ts title="sst.config.ts" {3}
 * const router = new sst.aws.Router("MyRouter");
 *
 * router.route("/", "https://some-external-service.com");
 * ```
 *
 * #### Route to an S3 bucket
 *
 * ```ts title="sst.config.ts" {2,6}
 * const myBucket = new sst.aws.Bucket("MyBucket", {
 *   access: "cloudfront"
 * });
 *
 * const router = new sst.aws.Router("MyRouter");
 * router.routeBucket("/files", myBucket);
 * ```
 *
 * You need to allow CloudFront to access the bucket by setting the `access` prop
 * on the bucket.
 *
 * #### Route to a function
 *
 * ```ts title="sst.config.ts" {8-11}
 * const router = new sst.aws.Router("MyRouter", {
 *   domain: "example.com"
 * });
 *
 * const myFunction = new sst.aws.Function("MyFunction", {
 *   handler: "src/api.handler",
 *   url: {
 *     router: {
 *       instance: router,
 *       path: "/api"
 *     }
 *   }
 * });
 * ```
 *
 * Setting the route through the function, instead of `router.route()` makes
 * it so that `myFunction.url` gives you the URL based on the Router domain.
 *
 * #### Route to a frontend
 *
 * ```ts title="sst.config.ts" {4-6}
 * const router = new sst.aws.Router("MyRouter");
 *
 * const mySite = new sst.aws.Nextjs("MyWeb", {
 *   router: {
 *     instance: router
 *   }
 * });
 * ```
 *
 * Setting the route through the site, instead of `router.route()` makes
 * it so that `mySite.url` gives you the URL based on the Router domain.
 *
 * #### Route to a frontend on a path
 *
 * ```ts title="sst.config.ts" {4-7}
 * const router = new sst.aws.Router("MyRouter");
 *
 * new sst.aws.Nextjs("MyWeb", {
 *   router: {
 *     instance: router,
 *     path: "/docs"
 *   }
 * });
 * ```
 *
 * If you are routing to a path, you'll need to configure the base path in your
 * frontend app as well. [Learn more](/docs/component/aws/nextjs/#router).
 *
 * #### Route to a frontend on a subdomain
 *
 * ```ts title="sst.config.ts" {4,9-12}
 * const router = new sst.aws.Router("MyRouter", {
 *   domain: {
 *     name: "example.com",
 *     aliases: ["*.example.com"]
 *   }
 * });
 *
 * new sst.aws.Nextjs("MyWeb", {
 *   router: {
 *     instance: router,
 *     domain: "docs.example.com"
 *   }
 * });
 * ```
 *
 * We configure `*.example.com` as an alias so that we can route to a subdomain.
 *
 * #### How it works
 *
 * This uses a CloudFront KeyValueStore to store the routing data and a CloudFront
 * function to route the request. As routes are added, the store is updated.
 *
 * So when a request comes in, it does a lookup in the store and dynamically sets
 * the origin based on the routing data. For frontends, that have their server
 * functions deployed to multiple `regions`, it routes to the closest region based
 * on the user's location.
 *
 * You might notice a _placeholder.sst.dev_ behavior in CloudFront. This is not
 * used and is only there because CloudFront requires a default behavior.
 *
 * #### Limits
 *
 * There are some limits on this setup but it's managed by SST.
 *
 * - The CloudFront function can be a maximum of 10KB in size. But because all
 *   the route data is stored in the KeyValueStore, the function can be kept small.
 * - Each value in the KeyValueStore needs to be less than 1KB. This component
 *   splits the routes into multiple values to keep it under the limit.
 * - The KeyValueStore can be a maximum of 5MB. This is fairly large. But to
 *   handle sites that have a lot of files, only top-level assets get individual
 *   entries.
 */
var Router = /** @class */ (function (_super) {
    __extends(Router, _super);
    function Router(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var _refVersion = 2;
        var self = _this;
        _this.constructorName = name;
        _this.constructorOpts = opts;
        if (args && "ref" in args) {
            var ref = reference();
            _this.cdn = (0, pulumi_1.output)(ref.cdn);
            _this.kvStoreArn = ref.kvStoreArn;
            _this.kvNamespace = ref.kvNamespace;
            _this.hasInlineRoutes = ref.hasInlineRoutes;
            registerOutputs();
            return _this;
        }
        var hasInlineRoutes = args.routes !== undefined;
        var cdn, kvStoreArn, kvNamespace;
        if (hasInlineRoutes) {
            cdn = handleInlineRoutes();
        }
        else {
            var r = handleLazyRoutes();
            cdn = (0, pulumi_1.output)(r.distribution);
            kvStoreArn = r.kvStoreArn;
            kvNamespace = (0, pulumi_1.output)(r.kvNamespace);
        }
        _this.cdn = cdn;
        _this.kvStoreArn = kvStoreArn;
        _this.kvNamespace = kvNamespace;
        _this.hasInlineRoutes = (0, pulumi_1.output)(hasInlineRoutes);
        registerOutputs();
        function reference() {
            var ref = args;
            var cdn = cdn_1.Cdn.get("".concat(name, "Cdn"), ref.distributionID, { parent: self });
            var tags = cdn.nodes.distribution.tags.apply(function (tags) {
                if ((tags === null || tags === void 0 ? void 0 : tags["sst:ref:version"]) !== _refVersion.toString()) {
                    throw new error_1.VisibleError([
                        "There have been some minor changes to the \"Router\" component that's being referenced by \"".concat(name, "\".\n"),
                        "To update, you'll need to redeploy the stage where the Router was created. And then redeploy this stage.",
                    ].join("\n"));
                }
                return {
                    kvStoreArn: tags === null || tags === void 0 ? void 0 : tags["sst:ref:kv"],
                    kvNamespace: tags === null || tags === void 0 ? void 0 : tags["sst:ref:kv-namespace"],
                    hasInlineRoutes: (tags === null || tags === void 0 ? void 0 : tags["sst:ref:kv"]) === undefined,
                };
            });
            return {
                cdn: cdn,
                kvStoreArn: tags.kvStoreArn,
                kvNamespace: tags.kvNamespace,
                hasInlineRoutes: tags.hasInlineRoutes,
            };
        }
        function registerOutputs() {
            self.registerOutputs({
                _hint: args._skipHint ? undefined : self.url,
            });
        }
        function handleInlineRoutes() {
            var defaultCachePolicy;
            var defaultCfFunction;
            var defaultOac;
            var routes = normalizeRoutes();
            var cdn = createCdn();
            return cdn;
            function normalizeRoutes() {
                return (0, pulumi_1.output)(args.routes).apply(function (routes) {
                    var _a;
                    var normalizedRoutes = Object.fromEntries(Object.entries(routes).map(function (_a) {
                        var path = _a[0], route = _a[1];
                        // Route path must start with "/"
                        if (!path.startsWith("/"))
                            throw new Error("In \"".concat(name, "\" Router, the route path \"").concat(path, "\" must start with a \"/\""));
                        route = typeof route === "string" ? { url: route } : route;
                        var hasUrl = "url" in route ? 1 : 0;
                        var hasBucket = "bucket" in route ? 1 : 0;
                        if (hasUrl + hasBucket !== 1)
                            throw new Error("In \"".concat(name, "\" Router, the route path \"").concat(path, "\" can only have one of url or bucket"));
                        return [path, route];
                    }));
                    normalizedRoutes["/*"] = (_a = normalizedRoutes["/*"]) !== null && _a !== void 0 ? _a : {
                        url: "https://do-not-exist.sst.dev",
                    };
                    return normalizedRoutes;
                });
            }
            function createCfRequestDefaultFunction() {
                defaultCfFunction =
                    defaultCfFunction !== null && defaultCfFunction !== void 0 ? defaultCfFunction : new aws_1.cloudfront.Function("".concat(name, "CloudfrontFunction"), {
                        runtime: "cloudfront-js-2.0",
                        code: [
                            "async function handler(event) {",
                            "  event.request.headers[\"x-forwarded-host\"] = event.request.headers.host;",
                            "  return event.request;",
                            "}",
                        ].join("\n"),
                    }, { parent: self });
                return defaultCfFunction;
            }
            function createCfRequestFunction(path, config, rewrite, injectHostHeader) {
                var _a, _b;
                return new aws_1.cloudfront.Function("".concat(name, "CloudfrontFunction").concat((0, naming_1.hashStringToPrettyString)(path, 8)), {
                    runtime: "cloudfront-js-2.0",
                    keyValueStoreAssociations: (config === null || config === void 0 ? void 0 : config.kvStore)
                        ? [config.kvStore]
                        : (_a = config === null || config === void 0 ? void 0 : config.kvStores) !== null && _a !== void 0 ? _a : [],
                    code: "\nasync function handler(event) {\n  ".concat(injectHostHeader
                        ? "event.request.headers[\"x-forwarded-host\"] = event.request.headers.host;"
                        : "", "\n  ").concat(rewrite
                        ? "\nconst re = new RegExp(\"".concat(rewrite.regex, "\");\nevent.request.uri = event.request.uri.replace(re, \"").concat(rewrite.to, "\");")
                        : "", "\n  ").concat((_b = config === null || config === void 0 ? void 0 : config.injection) !== null && _b !== void 0 ? _b : "", "\n  return event.request;\n}"),
                }, { parent: self });
            }
            function createCfResponseFunction(path, config) {
                var _a, _b;
                return new aws_1.cloudfront.Function("".concat(name, "CloudfrontFunctionResponse").concat((0, naming_1.hashStringToPrettyString)(path, 8)), {
                    runtime: "cloudfront-js-2.0",
                    keyValueStoreAssociations: config.kvStore
                        ? [config.kvStore]
                        : (_a = config.kvStores) !== null && _a !== void 0 ? _a : [],
                    code: "\nasync function handler(event) {\n  ".concat((_b = config.injection) !== null && _b !== void 0 ? _b : "", "\n  return event.response;\n}"),
                }, { parent: self });
            }
            function createOriginAccessControl() {
                defaultOac =
                    defaultOac !== null && defaultOac !== void 0 ? defaultOac : new origin_access_control_1.OriginAccessControl("".concat(name, "S3AccessControl"), { name: (0, naming_1.physicalName)(64, name) }, { parent: self, ignoreChanges: ["name"] });
                return defaultOac;
            }
            function createCachePolicy() {
                var _a;
                var _b;
                defaultCachePolicy =
                    defaultCachePolicy !== null && defaultCachePolicy !== void 0 ? defaultCachePolicy : new ((_a = aws_1.cloudfront.CachePolicy).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.cachePolicy, "".concat(name, "CachePolicy"), {
                        comment: "".concat(name, " router cache policy"),
                        defaultTtl: 0,
                        maxTtl: 31536000, // 1 year
                        minTtl: 0,
                        parametersInCacheKeyAndForwardedToOrigin: {
                            cookiesConfig: {
                                cookieBehavior: "none",
                            },
                            headersConfig: {
                                headerBehavior: "none",
                            },
                            queryStringsConfig: {
                                queryStringBehavior: "all",
                            },
                            enableAcceptEncodingBrotli: true,
                            enableAcceptEncodingGzip: true,
                        },
                    }, { parent: self }), false)))();
                return defaultCachePolicy;
            }
            function createCdn() {
                return routes.apply(function (routes) {
                    var _a;
                    var distributionData = Object.entries(routes).map(function (_a) {
                        var _b, _c, _d, _e, _f, _g, _h, _j, _k;
                        var path = _a[0], route = _a[1];
                        if ("url" in route) {
                            return {
                                origin: {
                                    originId: path,
                                    domainName: new URL(route.url).host,
                                    customOriginConfig: {
                                        httpPort: 80,
                                        httpsPort: 443,
                                        originProtocolPolicy: "https-only",
                                        originReadTimeout: 20,
                                        originSslProtocols: ["TLSv1.2"],
                                    },
                                },
                                behavior: {
                                    pathPattern: path,
                                    targetOriginId: path,
                                    functionAssociations: __spreadArray([
                                        {
                                            eventType: "viewer-request",
                                            functionArn: ((_b = route.edge) === null || _b === void 0 ? void 0 : _b.viewerRequest) || route.rewrite
                                                ? createCfRequestFunction(path, (_c = route.edge) === null || _c === void 0 ? void 0 : _c.viewerRequest, route.rewrite, true).arn
                                                : createCfRequestDefaultFunction().arn,
                                        }
                                    ], (((_d = route.edge) === null || _d === void 0 ? void 0 : _d.viewerResponse)
                                        ? [
                                            {
                                                eventType: "viewer-response",
                                                functionArn: createCfResponseFunction(path, route.edge.viewerResponse).arn,
                                            },
                                        ]
                                        : []), true),
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
                                    defaultTtl: 0,
                                    compress: true,
                                    cachePolicyId: (_e = route.cachePolicy) !== null && _e !== void 0 ? _e : createCachePolicy().id,
                                    // CloudFront's Managed-AllViewerExceptHostHeader policy
                                    originRequestPolicyId: "b689b0a8-53d0-40ab-baf2-68738e2966ac",
                                },
                            };
                        }
                        else if ("bucket" in route) {
                            return {
                                origin: {
                                    originId: path,
                                    domainName: route.bucket instanceof bucket_1.Bucket
                                        ? route.bucket.nodes.bucket.bucketRegionalDomainName
                                        : route.bucket,
                                    originPath: "",
                                    originAccessControlId: createOriginAccessControl().id,
                                },
                                behavior: {
                                    pathPattern: path,
                                    targetOriginId: path,
                                    functionAssociations: __spreadArray(__spreadArray([], (((_f = route.edge) === null || _f === void 0 ? void 0 : _f.viewerRequest) || route.rewrite
                                        ? [
                                            {
                                                eventType: "viewer-request",
                                                functionArn: ((_g = route.edge) === null || _g === void 0 ? void 0 : _g.viewerRequest) || route.rewrite
                                                    ? createCfRequestFunction(path, (_h = route.edge) === null || _h === void 0 ? void 0 : _h.viewerRequest, route.rewrite, false).arn
                                                    : createCfRequestDefaultFunction().arn,
                                            },
                                        ]
                                        : []), true), (((_j = route.edge) === null || _j === void 0 ? void 0 : _j.viewerResponse)
                                        ? [
                                            {
                                                eventType: "viewer-response",
                                                functionArn: createCfResponseFunction(path, route.edge.viewerResponse).arn,
                                            },
                                        ]
                                        : []), true),
                                    viewerProtocolPolicy: "redirect-to-https",
                                    allowedMethods: ["GET", "HEAD", "OPTIONS"],
                                    cachedMethods: ["GET", "HEAD"],
                                    compress: true,
                                    // CloudFront's managed CachingOptimized policy
                                    cachePolicyId: (_k = route.cachePolicy) !== null && _k !== void 0 ? _k : "658327ea-f89d-4fab-a63d-7e88639e58f6",
                                },
                            };
                        }
                        throw new Error("Invalid route type");
                    });
                    return new (cdn_1.Cdn.bind.apply(cdn_1.Cdn, __spreadArray([void 0], (0, component_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.cdn, "".concat(name, "Cdn"), {
                        comment: "".concat(name, " router"),
                        origins: distributionData.map(function (d) { return d.origin; }),
                        defaultCacheBehavior: __assign(__assign({}, distributionData.find(function (d) { return d.behavior.pathPattern === "/*"; }).behavior), { 
                            // @ts-expect-error
                            pathPattern: undefined }),
                        orderedCacheBehaviors: distributionData
                            .filter(function (d) { return d.behavior.pathPattern !== "/*"; })
                            .map(function (d) { return d.behavior; }),
                        domain: args.domain,
                        wait: true,
                    }, { parent: self }), false)))();
                });
            }
        }
        function handleLazyRoutes() {
            var kvNamespace = buildRequestKvNamespace();
            var kvStoreArn = createRequestKvStore();
            var requestFunction = createRequestFunction();
            var responseFunction = createResponseFunction();
            var cachePolicyId = createCachePolicy().id;
            var distribution = createDistribution();
            return { kvNamespace: kvNamespace, kvStoreArn: kvStoreArn, distribution: distribution };
            function buildRequestKvNamespace() {
                // In the case multiple routers use the same kv store, we need to namespace the keys
                return crypto_1.default
                    .createHash("md5")
                    .update("".concat($app.name, "-").concat($app.stage, "-").concat(name))
                    .digest("hex")
                    .substring(0, 4);
            }
            function createRequestKvStore() {
                return (0, pulumi_1.output)(args.edge).apply(function (edge) {
                    var viewerRequest = edge === null || edge === void 0 ? void 0 : edge.viewerRequest;
                    var userKvStore = viewerRequest === null || viewerRequest === void 0 ? void 0 : viewerRequest.kvStore;
                    if (userKvStore)
                        return (0, pulumi_1.output)(userKvStore);
                    return new aws_1.cloudfront.KeyValueStore("".concat(name, "KvStore"), {}, { parent: self }).arn;
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
            function createRequestFunction() {
                return (0, pulumi_1.output)(args.edge).apply(function (edge) {
                    var _a, _b;
                    var userInjection = (_b = (_a = edge === null || edge === void 0 ? void 0 : edge.viewerRequest) === null || _a === void 0 ? void 0 : _a.injection) !== null && _b !== void 0 ? _b : "";
                    var blockCloudfrontUrlInjection = args.domain
                        ? exports.CF_BLOCK_CLOUDFRONT_URL_INJECTION
                        : "";
                    return new aws_1.cloudfront.Function("".concat(name, "CloudfrontFunctionRequest"), {
                        runtime: "cloudfront-js-2.0",
                        keyValueStoreAssociations: kvStoreArn ? [kvStoreArn] : [],
                        code: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["\nimport cf from \"cloudfront\";\nasync function handler(event) {\n  ", "\n  ", "\n  ", "\n\n  const routerNS = \"", "\";\n\n  async function getRoutes() {\n    let routes = [];\n    try {\n      const v = await cf.kvs().get(routerNS + \":routes\");\n      routes = JSON.parse(v);\n\n      // handle chunked routes\n      if (routes.parts) {\n        const chunkPromises = [];\n        for (let i = 0; i < routes.parts; i++) {\n          chunkPromises.push(cf.kvs().get(routerNS + \":routes:\" + i));\n        }\n        const chunks = await Promise.all(chunkPromises);\n        routes = JSON.parse(chunks.join(\"\"));\n      }\n    } catch (e) {}\n    return routes;\n  }\n\n  async function matchRoute(routes) {\n    const requestHost = event.request.headers.host.value;\n    const match = routes\n      ", "\n      .map(r => {\n        var parts = r.split(\",\");\n        return { \n          type: parts[0], \n          routeNs: parts[1], \n          host: parts[2],\n          path: parts[3]\n        };\n      })\n      .sort((a, b) => {\n        return (a.host.length !== b.host.length)\n          ? b.host.length - a.host.length\n          : b.path.length - a.path.length;\n      })\n      .find(r => {\n        const hostMatches = r.host === \"\" || new RegExp(r.host).test(\"^\" + requestHost + \"$\");\n        const pathMatches = event.request.uri.startsWith(r.path);\n        return hostMatches && pathMatches;\n      });\n\n    // Load metadata\n    if (match) {\n      try {\n        const v = await cf.kvs().get(match.routeNs + \":metadata\");\n        return { type: match.type, routeNs: match.routeNs, metadata: JSON.parse(v) };\n      } catch (e) {}\n    }\n  }\n\n  // Look up the route\n  const routes = await getRoutes();\n  const route = await matchRoute(routes);\n  if (!route) return event.request;\n  if (route.metadata.rewrite) {\n    const rw = route.metadata.rewrite;\n    event.request.uri = event.request.uri.replace(new RegExp(rw.regex), rw.to);\n  }\n  if (route.type === \"url\") setUrlOrigin(route.metadata.host, route.metadata.origin);\n  if (route.type === \"bucket\") setS3Origin(route.metadata.domain, route.metadata.origin);\n  if (route.type === \"site\") await routeSite(route.routeNs, route.metadata);\n  return event.request;\n}"], ["\nimport cf from \"cloudfront\";\nasync function handler(event) {\n  ", "\n  ", "\n  ", "\n\n  const routerNS = \"", "\";\n\n  async function getRoutes() {\n    let routes = [];\n    try {\n      const v = await cf.kvs().get(routerNS + \":routes\");\n      routes = JSON.parse(v);\n\n      // handle chunked routes\n      if (routes.parts) {\n        const chunkPromises = [];\n        for (let i = 0; i < routes.parts; i++) {\n          chunkPromises.push(cf.kvs().get(routerNS + \":routes:\" + i));\n        }\n        const chunks = await Promise.all(chunkPromises);\n        routes = JSON.parse(chunks.join(\"\"));\n      }\n    } catch (e) {}\n    return routes;\n  }\n\n  async function matchRoute(routes) {\n    const requestHost = event.request.headers.host.value;\n    const match = routes\n      "
                            /*
                            Route format: [type, routeNamespace, hostRegex, pathPrefix]
                            - First sort by host pattern (longest first)
                            - Then sort by path prefix (longest first)
                          */ , "\n      .map(r => {\n        var parts = r.split(\",\");\n        return { \n          type: parts[0], \n          routeNs: parts[1], \n          host: parts[2],\n          path: parts[3]\n        };\n      })\n      .sort((a, b) => {\n        return (a.host.length !== b.host.length)\n          ? b.host.length - a.host.length\n          : b.path.length - a.path.length;\n      })\n      .find(r => {\n        const hostMatches = r.host === \"\" || new RegExp(r.host).test(\"^\" + requestHost + \"$\");\n        const pathMatches = event.request.uri.startsWith(r.path);\n        return hostMatches && pathMatches;\n      });\n\n    // Load metadata\n    if (match) {\n      try {\n        const v = await cf.kvs().get(match.routeNs + \":metadata\");\n        return { type: match.type, routeNs: match.routeNs, metadata: JSON.parse(v) };\n      } catch (e) {}\n    }\n  }\n\n  // Look up the route\n  const routes = await getRoutes();\n  const route = await matchRoute(routes);\n  if (!route) return event.request;\n  if (route.metadata.rewrite) {\n    const rw = route.metadata.rewrite;\n    event.request.uri = event.request.uri.replace(new RegExp(rw.regex), rw.to);\n  }\n  if (route.type === \"url\") setUrlOrigin(route.metadata.host, route.metadata.origin);\n  if (route.type === \"bucket\") setS3Origin(route.metadata.domain, route.metadata.origin);\n  if (route.type === \"site\") await routeSite(route.routeNs, route.metadata);\n  return event.request;\n}"])), userInjection, blockCloudfrontUrlInjection, exports.CF_ROUTER_INJECTION, kvNamespace, 
                        /*
                        Route format: [type, routeNamespace, hostRegex, pathPrefix]
                        - First sort by host pattern (longest first)
                        - Then sort by path prefix (longest first)
                      */ ""),
                    }, { parent: self });
                });
            }
            function createResponseFunction() {
                return (0, pulumi_1.output)(args.edge).apply(function (edge) {
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
                var _a;
                return new (cdn_1.Cdn.bind.apply(cdn_1.Cdn, __spreadArray([void 0], (0, component_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.cdn, "".concat(name, "Cdn"), {
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
                        cachePolicyId: cachePolicyId,
                        // CloudFront's Managed-AllViewerExceptHostHeader policy
                        originRequestPolicyId: "b689b0a8-53d0-40ab-baf2-68738e2966ac",
                        functionAssociations: (0, pulumi_1.all)([
                            requestFunction,
                            responseFunction,
                        ]).apply(function (_a) {
                            var reqFn = _a[0], resFn = _a[1];
                            return __spreadArray([
                                { eventType: "viewer-request", functionArn: reqFn.arn }
                            ], (resFn
                                ? [{ eventType: "viewer-response", functionArn: resFn.arn }]
                                : []), true);
                        }),
                    },
                    tags: {
                        "sst:ref:kv": kvStoreArn,
                        "sst:ref:kv-namespace": kvNamespace,
                        "sst:ref:version": _refVersion.toString(),
                    },
                }, { parent: self }), false)))();
            }
        }
        return _this;
    }
    Object.defineProperty(Router.prototype, "distributionID", {
        /**
         * The ID of the Router distribution.
         */
        get: function () {
            return this.cdn.nodes.distribution.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Router.prototype, "url", {
        /**
         * The URL of the Router.
         *
         * If the `domain` is set, this is the URL with the custom domain.
         * Otherwise, it's the auto-generated CloudFront URL.
         */
        get: function () {
            return (0, pulumi_1.all)([this.cdn.domainUrl, this.cdn.url]).apply(function (_a) {
                var domainUrl = _a[0], url = _a[1];
                return domainUrl !== null && domainUrl !== void 0 ? domainUrl : url;
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Router.prototype, "_kvStoreArn", {
        /** @internal */
        get: function () {
            return this.kvStoreArn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Router.prototype, "_kvNamespace", {
        /** @internal */
        get: function () {
            return this.kvNamespace;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Router.prototype, "_hasInlineRoutes", {
        /** @internal */
        get: function () {
            return this.hasInlineRoutes;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Router.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon CloudFront CDN resource.
                 */
                cdn: this.cdn,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Add a route to a destination URL.
     *
     * @param pattern The path pattern to match for this route.
     * @param url The destination URL to route matching requests to.
     * @param args Configure the route.
     *
     * @example
     *
     * You can match a route based on:
     *
     * - A path like `/api`
     * - A domain pattern like `api.example.com`
     * - A combined pattern like `dev.example.com/api`
     *
     * For example, to match a path.
     *
     * ```ts title="sst.config.ts"
     * router.route("/api", "https://api.example.com");
     * ```
     *
     * Or match a domain.
     *
     * ```ts title="sst.config.ts"
     * router.route("api.myapp.com/", "https://api.example.com");
     * ```
     *
     * Or a combined pattern.
     *
     * ```ts title="sst.config.ts"
     * router.route("dev.myapp.com/api", "https://api.example.com");
     * ```
     *
     * You can also rewrite the request path.
     *
     * ```ts title="sst.config.ts"
     * router.route("/api", "https://api.example.com", {
     *   rewrite: {
     *     regex: "^/api/(.*)$",
     *     to: "/$1"
     *   }
     * });
     * ```
     *
     * Here something like `/api/users/profile` will be routed to
     * `https://api.example.com/users/profile`.
     */
    Router.prototype.route = function (pattern, url, args) {
        var _this = this;
        (0, pulumi_1.all)([pattern, args, this.hasInlineRoutes]).apply(function (_a) {
            var pattern = _a[0], args = _a[1], hasInlineRoutes = _a[2];
            if (hasInlineRoutes)
                throw new error_1.VisibleError("Cannot use both `routes` and `.route()` function to add routes.");
            new router_url_route_1.RouterUrlRoute("".concat(_this.constructorName, "Route").concat(pattern), {
                store: _this.kvStoreArn,
                routerNamespace: _this.kvNamespace,
                pattern: pattern,
                url: url,
                routeArgs: args,
            }, { provider: _this.constructorOpts.provider });
        });
    };
    /**
     * Add a route to an S3 bucket.
     *
     * @param pattern The path pattern to match for this route.
     * @param bucket The S3 bucket to route matching requests to.
     * @param args Configure the route.
     *
     * @example
     *
     * Let's say you have an S3 bucket that gives CloudFront `access`.
     *
     * ```ts title="sst.config.ts" {2}
     * const bucket = new sst.aws.Bucket("MyBucket", {
     *   access: "cloudfront"
     * });
     * ```
     *
     * You can match a pattern and route to it based on:
     *
     * - A path like `/api`
     * - A domain pattern like `api.example.com`
     * - A combined pattern like `dev.example.com/api`
     *
     * For example, to match a path.
     *
     * ```ts title="sst.config.ts"
     * router.routeBucket("/files", bucket);
     * ```
     *
     * Or match a domain.
     *
     * ```ts title="sst.config.ts"
     * router.routeBucket("files.example.com", bucket);
     * ```
     *
     * Or a combined pattern.
     *
     * ```ts title="sst.config.ts"
     * router.routeBucket("dev.example.com/files", bucket);
     * ```
     *
     * You can also rewrite the request path.
     *
     * ```ts title="sst.config.ts"
     * router.routeBucket("/files", bucket, {
     *   rewrite: {
     *     regex: "^/files/(.*)$",
     *     to: "/$1"
     *   }
     * });
     * ```
     *
     * Here something like `/files/logo.png` will be routed to
     * `/logo.png`.
     */
    Router.prototype.routeBucket = function (pattern, bucket, args) {
        var _this = this;
        (0, pulumi_1.all)([pattern, args, this.hasInlineRoutes]).apply(function (_a) {
            var pattern = _a[0], args = _a[1], hasInlineRoutes = _a[2];
            if (hasInlineRoutes)
                throw new error_1.VisibleError("Cannot use both `routes` and `.routeBucket()` function to add routes.");
            new router_bucket_route_1.RouterBucketRoute("".concat(_this.constructorName, "Route").concat(pattern), {
                store: _this.kvStoreArn,
                routerNamespace: _this.kvNamespace,
                pattern: pattern,
                bucket: bucket,
                routeArgs: args,
            }, { provider: _this.constructorOpts.provider });
        });
    };
    /**
     * Add a route to a frontend or static site.
     *
     * @param pattern The path pattern to match for this route.
     * @param site The frontend or static site to route matching requests to.
     *
     * @deprecated The `routeSite` function has been deprecated. Set the `route` on the
     * site components to route the site through this Router.
     */
    Router.prototype.routeSite = function (pattern, site) {
        throw new error_1.VisibleError("The \"routeSite\" function has been deprecated. Configure the new \"route\" prop on the site component to route the site through this Router.");
    };
    /** @internal */
    Router.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
            },
        };
    };
    /**
     * Reference an existing Router with the given Router distribution ID.
     *
     * @param name The name of the component.
     * @param distributionID The ID of the existing Router distribution.
     * @param opts? Resource options.
     *
     * This is useful when you create a Router in one stage and want to share it in
     * another. It avoids having to create a new Router in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share a Router across stages.
     * :::
     *
     * @example
     * Let's say you create a Router in the `dev` stage. And in your personal stage
     * `frank`, you want to share the same Router.
     *
     * ```ts title="sst.config.ts"
     * const router = $app.stage === "frank"
     *   ? sst.aws.Router.get("MyRouter", "E2IDLMESRN6V62")
     *   : new sst.aws.Router("MyRouter");
     * ```
     *
     * Here `E2IDLMESRN6V62` is the ID of the Router distribution created in the
     * `dev` stage. You can find this by outputting the distribution ID in the `dev`
     * stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   router: router.distributionID
     * };
     * ```
     *
     * Learn more about [how to configure a router for your app](/docs/configure-a-router).
     */
    Router.get = function (name, distributionID, opts) {
        return new Router(name, {
            ref: true,
            distributionID: distributionID,
        }, opts);
    };
    return Router;
}(component_1.Component));
exports.Router = Router;
var __pulumiType = "sst:aws:Router";
// @ts-expect-error
Router.__pulumiType = __pulumiType;
exports.CF_BLOCK_CLOUDFRONT_URL_INJECTION = "\nif (event.request.headers.host.value.includes('cloudfront.net')) {\n  return {\n    statusCode: 403,\n    statusDescription: 'Forbidden',\n    body: {\n      encoding: \"text\",\n      data: '<html><head><title>403 Forbidden</title></head><body><center><h1>403 Forbidden</h1></center></body></html>'\n    }\n  };\n}";
exports.CF_ROUTER_INJECTION = "\nasync function routeSite(kvNamespace, metadata) {\n  const baselessUri = metadata.base\n    ? event.request.uri.replace(metadata.base, \"\")\n    : event.request.uri;\n\n  // Route to S3 files\n  try {\n    // check using baselessUri b/c files are stored in the root\n    const u = decodeURIComponent(baselessUri);\n    const postfixes = u.endsWith(\"/\")\n      ? [\"index.html\"]\n      : [\"\", \".html\", \"/index.html\"];\n    const v = await Promise.any(postfixes.map(p => cf.kvs().get(kvNamespace + \":\" + u + p).then(v => p)));\n    // files are stored in a subdirectory, add it to the request uri\n    event.request.uri = metadata.s3.dir + event.request.uri + v;\n    setS3Origin(metadata.s3.domain);\n    return;\n  } catch (e) {}\n\n  // Route to S3 routes\n  if (metadata.s3 && metadata.s3.routes) {\n    for (var i=0, l=metadata.s3.routes.length; i<l; i++) {\n      const route = metadata.s3.routes[i];\n      if (baselessUri.startsWith(route)) {\n        event.request.uri = metadata.s3.dir + event.request.uri;\n        // uri ends with /, ie. /usage/ -> /usage/index.html\n        if (event.request.uri.endsWith(\"/\")) {\n          event.request.uri += \"index.html\";\n        }\n        // uri ends with non-file, ie. /usage -> /usage/index.html\n        else if (!event.request.uri.split(\"/\").pop().includes(\".\")) {\n          event.request.uri += \"/index.html\";\n        }\n        setS3Origin(metadata.s3.domain);\n        return;\n      }\n    }\n  }\n\n  // Route to S3 custom 404 (no servers)\n  if (metadata.custom404) {\n    event.request.uri = metadata.s3.dir + (metadata.base ? metadata.base : \"\") + metadata.custom404;\n    setS3Origin(metadata.s3.domain);\n    return;\n  }\n\n  // Route to image optimizer\n  if (metadata.image && baselessUri.startsWith(metadata.image.route)) {\n    setUrlOrigin(metadata.image.host);\n    return;\n  }\n\n  // Route to servers\n  if (metadata.servers){\n    event.request.headers[\"x-forwarded-host\"] = event.request.headers.host;\n    ".concat(
// Note: In SvelteKit, form action requests contain "/" in request query string
//  ie. POST request with query string "?/action"
//  CloudFront does not allow query string with "/". It needs to be encoded.
"", "\n    for (var key in event.request.querystring) {\n      if (key.includes(\"/\")) {\n        event.request.querystring[encodeURIComponent(key)] = event.request.querystring[key];\n        delete event.request.querystring[key];\n      }\n    }\n    setNextjsGeoHeaders();\n    setNextjsCacheKey();\n    setUrlOrigin(findNearestServer(metadata.servers), metadata.origin);\n  }\n\n  function setNextjsGeoHeaders() {\n    ").concat(
// Inject the CloudFront viewer country, region, latitude, and longitude headers into
// the request headers for OpenNext to use them for OpenNext to use them
"", "\n    if(event.request.headers[\"cloudfront-viewer-city\"]) {\n      event.request.headers[\"x-open-next-city\"] = event.request.headers[\"cloudfront-viewer-city\"];\n    }\n    if(event.request.headers[\"cloudfront-viewer-country\"]) {\n      event.request.headers[\"x-open-next-country\"] = event.request.headers[\"cloudfront-viewer-country\"];\n    }\n    if(event.request.headers[\"cloudfront-viewer-region\"]) {\n      event.request.headers[\"x-open-next-region\"] = event.request.headers[\"cloudfront-viewer-region\"];\n    }\n    if(event.request.headers[\"cloudfront-viewer-latitude\"]) {\n      event.request.headers[\"x-open-next-latitude\"] = event.request.headers[\"cloudfront-viewer-latitude\"];\n    }\n    if(event.request.headers[\"cloudfront-viewer-longitude\"]) {\n      event.request.headers[\"x-open-next-longitude\"] = event.request.headers[\"cloudfront-viewer-longitude\"];\n    }\n  }\n\n  function setNextjsCacheKey() {\n    ").concat(
// This function is used to improve cache hit ratio by setting the cache key
// based on the request headers and the path. `next/image` only needs the
// accept header, and this header is not useful for the rest of the query
"", "\n    var cacheKey = \"\";\n    if (event.request.uri.startsWith(\"/_next/image\")) {\n      cacheKey = getHeader(\"accept\");\n    } else {\n      cacheKey =\n        getHeader(\"rsc\") +\n        getHeader(\"next-router-prefetch\") +\n        getHeader(\"next-router-state-tree\") +\n        getHeader(\"next-url\") +\n        getHeader(\"x-prerender-revalidate\");\n    }\n    if (event.request.cookies[\"__prerender_bypass\"]) {\n      cacheKey += event.request.cookies[\"__prerender_bypass\"]\n        ? event.request.cookies[\"__prerender_bypass\"].value\n        : \"\";\n    }\n    var crypto = require(\"crypto\");\n    var hashedKey = crypto.createHash(\"md5\").update(cacheKey).digest(\"hex\");\n    event.request.headers[\"x-open-next-cache-key\"] = { value: hashedKey };\n  }\n\n  function getHeader(key) {\n    var header = event.request.headers[key];\n    if (header) {\n      if (header.multiValue) {\n        return header.multiValue.map((header) => header.value).join(\",\");\n      }\n      if (header.value) {\n        return header.value;\n      }\n    }\n    return \"\";\n  }\n\n  function findNearestServer(servers) {\n    if (servers.length === 1) return servers[0][0];\n\n    const h = event.request.headers;\n    const lat = h[\"cloudfront-viewer-latitude\"] && h[\"cloudfront-viewer-latitude\"].value;\n    const lon = h[\"cloudfront-viewer-longitude\"] && h[\"cloudfront-viewer-longitude\"].value;\n    if (!lat || !lon) return servers[0][0];\n\n    return servers\n      .map((s) => ({\n        distance: haversineDistance(lat, lon, s[1], s[2]),\n        host: s[0],\n      }))\n      .sort((a, b) => a.distance - b.distance)[0]\n      .host;\n  }\n\n  function haversineDistance(lat1, lon1, lat2, lon2) {\n    const toRad = angle => angle * Math.PI / 180;\n    const radLat1 = toRad(lat1);\n    const radLat2 = toRad(lat2);\n    const dLat = toRad(lat2 - lat1);\n    const dLon = toRad(lon2 - lon1);\n    const a = Math.sin(dLat / 2) ** 2 + Math.cos(radLat1) * Math.cos(radLat2) * Math.sin(dLon / 2) ** 2;\n    return 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));\n  }\n}\n\nfunction setUrlOrigin(urlHost, override) {\n  event.request.headers[\"x-forwarded-host\"] = event.request.headers.host;\n  const origin = {\n    domainName: urlHost,\n    customOriginConfig: {\n      port: 443,\n      protocol: \"https\",\n      sslProtocols: [\"TLSv1.2\"],\n    },\n    originAccessControlConfig: {\n      enabled: false,\n    }\n  };\n  override = override ?? {};\n  if (override.protocol === \"http\") {\n    delete origin.customOriginConfig;\n  }\n  if (override.connectionAttempts) {\n    origin.connectionAttempts = override.connectionAttempts;\n  }\n  if (override.timeouts) {\n    origin.timeouts = override.timeouts;\n  }\n  cf.updateRequestOrigin(origin);\n}\n\nfunction setS3Origin(s3Domain, override) {\n  delete event.request.headers[\"Cookies\"];\n  delete event.request.headers[\"cookies\"];\n  delete event.request.cookies;\n\n  const origin = {\n    domainName: s3Domain,\n    originAccessControlConfig: {\n      enabled: true,\n      signingBehavior: \"always\",\n      signingProtocol: \"sigv4\",\n      originType: \"s3\",\n    }\n  };\n  override = override ?? {};\n  if (override.connectionAttempts) {\n    origin.connectionAttempts = override.connectionAttempts;\n  }\n  if (override.timeouts) {\n    origin.timeouts = override.timeouts;\n  }\n  cf.updateRequestOrigin(origin);\n}");
function normalizeRouteArgs(route, routeDeprecated) {
    if (!route && !routeDeprecated)
        return undefined;
    return (0, pulumi_1.all)([route, routeDeprecated]).apply(function (_a) {
        var route = _a[0], routeDeprecated = _a[1];
        var v = route
            ? route
            : __assign(__assign({}, routeDeprecated), { instance: routeDeprecated.router });
        return v.instance._hasInlineRoutes.apply(function (hasInlineRoutes) {
            if (hasInlineRoutes)
                throw new error_1.VisibleError("Cannot route the site using the provided router. The Router component uses inline routes which has been deprecated.");
            var pathPrefix = v.path
                ? "/" + v.path.replace(/^\//, "").replace(/\/$/, "")
                : undefined;
            return {
                hostPattern: v.domain
                    ? v.domain
                        .replace(/[.+?^${}()|[\]\\]/g, "\\$&") // Escape special regex chars
                        .replace(/\*/g, ".*") // Replace * with .*
                    : undefined,
                pathPrefix: pathPrefix,
                routerDistributionId: v.instance.nodes.cdn.nodes.distribution.id,
                routerUrl: v.instance.url.apply(function (url) {
                    return (v.domain ? "https://".concat(v.domain) : url) + (pathPrefix !== null && pathPrefix !== void 0 ? pathPrefix : "");
                }),
                routerKvNamespace: v.instance._kvNamespace,
                routerKvStoreArn: v.instance._kvStoreArn,
            };
        });
    });
}
var templateObject_1;
