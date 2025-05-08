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
exports.StaticSite = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var crypto_1 = require("crypto");
var pulumi_1 = require("@pulumi/pulumi");
var cdn_js_1 = require("./cdn.js");
var bucket_js_1 = require("./bucket.js");
var component_js_1 = require("../component.js");
var glob_1 = require("glob");
var bucket_files_js_1 = require("./providers/bucket-files.js");
var base_site_js_1 = require("../base/base-site.js");
var base_static_site_js_1 = require("../base/base-static-site.js");
var aws_1 = require("@pulumi/aws");
var linkable_js_1 = require("./linkable.js");
var kv_keys_js_1 = require("./providers/kv-keys.js");
var router_js_1 = require("./router.js");
var distribution_invalidation_js_1 = require("./providers/distribution-invalidation.js");
var error_js_1 = require("../error.js");
var kv_routes_update_js_1 = require("./providers/kv-routes-update.js");
/**
 * The `StaticSite` component lets you deploy a static website to AWS. It uses [Amazon S3](https://aws.amazon.com/s3/) to store your files and [Amazon CloudFront](https://aws.amazon.com/cloudfront/) to serve them.
 *
 * It can also `build` your site by running your static site generator, like [Vite](https://vitejs.dev) and uploading the build output to S3.
 *
 * @example
 *
 * #### Minimal example
 *
 * Simply uploads the current directory as a static site.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.StaticSite("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Change the `path` that should be uploaded.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.StaticSite("MyWeb", {
 *   path: "path/to/site"
 * });
 * ```
 *
 * #### Running locally
 *
 * In `sst dev`, we don't deploy your site to AWS because we assume you are running it locally.
 *
 * :::note
 * Your static site will not be deployed when run locally with `sst dev`.
 * :::
 *
 * For example, for a Vite site, you can run it locally with.
 *
 * ```bash
 * sst dev vite dev
 * ```
 *
 * This will start the Vite dev server and pass in any environment variables that you've set in your config. But it will not deploy your site to AWS.
 *
 * #### Deploy a Vite SPA
 *
 * Use [Vite](https://vitejs.dev) to deploy a React/Vue/Svelte/etc. SPA by specifying the `build` config.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.StaticSite("MyWeb", {
 *   build: {
 *     command: "npm run build",
 *     output: "dist"
 *   }
 * });
 * ```
 *
 * #### Deploy a Jekyll site
 *
 * Use [Jekyll](https://jekyllrb.com) to deploy a static site.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.StaticSite("MyWeb", {
 *   errorPage: "/404.html",
 *   build: {
 *     command: "bundle exec jekyll build",
 *     output: "_site"
 *   }
 * });
 * ```
 *
 * #### Deploy a Gatsby site
 *
 * Use [Gatsby](https://www.gatsbyjs.com) to deploy a static site.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.StaticSite("MyWeb", {
 *   errorPage: "/404.html",
 *   build: {
 *     command: "npm run build",
 *     output: "public"
 *   }
 * });
 * ```
 *
 * #### Deploy an Angular SPA
 *
 * Use [Angular](https://angular.dev) to deploy a SPA.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.StaticSite("MyWeb", {
 *   build: {
 *     command: "ng build --output-path dist",
 *     output: "dist"
 *   }
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your site.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.StaticSite("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.aws.StaticSite("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Set environment variables
 *
 * Set `environment` variables for the build process of your static site. These will be used locally and on deploy.
 *
 * :::tip
 * For Vite, the types for the environment variables are also generated. This can be configured through the `vite` prop.
 * :::
 *
 * For some static site generators like Vite, [environment variables](https://vitejs.dev/guide/env-and-mode) prefixed with `VITE_` can be accessed in the browser.
 *
 * ```ts {5-7} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.StaticSite("MyWeb", {
 *   environment: {
 *     BUCKET_NAME: bucket.name,
 *     // Accessible in the browser
 *     VITE_STRIPE_PUBLISHABLE_KEY: "pk_test_123"
 *   },
 *   build: {
 *     command: "npm run build",
 *     output: "dist"
 *   }
 * });
 * ```
 */
var StaticSite = /** @class */ (function (_super) {
    __extends(StaticSite, _super);
    function StaticSite(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        validateDeprecatedProps();
        var _a = (0, base_static_site_js_1.prepare)(args), sitePath = _a.sitePath, environment = _a.environment, indexPage = _a.indexPage;
        var dev = normalizeDev();
        if (dev.enabled) {
            _this.devUrl = dev.url;
            _this.registerOutputs({
                _metadata: {
                    mode: "placeholder",
                    path: sitePath,
                    environment: environment,
                    url: _this.url,
                },
                _dev: dev.outputs,
            });
            return _this;
        }
        var route = normalizeRoute();
        var errorPage = normalizeErrorPage();
        var assets = normalizeAsssets();
        var outputPath = (0, base_static_site_js_1.buildApp)(self, name, args.build, sitePath, environment);
        var bucket = createBucket();
        var _b = getBucketDetails(), bucketName = _b.bucketName, bucketDomain = _b.bucketDomain;
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
        var kvUpdated = createKvEntries();
        createInvalidation();
        _this.bucket = bucket;
        _this.cdn = distribution;
        _this.prodUrl = prodUrl;
        _this.registerOutputs({
            _hint: _this.url,
            _metadata: {
                mode: "deployed",
                path: sitePath,
                environment: environment,
                url: _this.url,
            },
            _dev: dev.outputs,
        });
        function validateDeprecatedProps() {
            if (args.base !== undefined)
                throw new error_js_1.VisibleError("\"base\" prop is deprecated. Use the \"route.path\" prop instead to set the base path of the site.");
            if (args.cdn !== undefined)
                throw new error_js_1.VisibleError("\"cdn\" prop is deprecated. Use the \"route.router\" prop instead to use an existing \"Router\" component to serve your site.");
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
        function normalizeDev() {
            var _a, _b, _c, _d;
            var enabled = $dev && args.dev !== false;
            var devArgs = args.dev || {};
            return {
                enabled: enabled,
                url: (0, pulumi_1.output)((_a = devArgs.url) !== null && _a !== void 0 ? _a : linkable_js_1.URL_UNAVAILABLE),
                outputs: {
                    title: devArgs.title,
                    environment: environment,
                    command: (0, pulumi_1.output)((_b = devArgs.command) !== null && _b !== void 0 ? _b : "npm run dev"),
                    autostart: (0, pulumi_1.output)((_c = devArgs.autostart) !== null && _c !== void 0 ? _c : true),
                    directory: (0, pulumi_1.output)((_d = devArgs.directory) !== null && _d !== void 0 ? _d : sitePath),
                },
            };
        }
        function normalizeErrorPage() {
            return (0, pulumi_1.all)([indexPage, args.errorPage]).apply(function (_a) {
                var indexPage = _a[0], errorPage = _a[1];
                return "/" + (errorPage !== null && errorPage !== void 0 ? errorPage : indexPage).replace(/^\//, "");
            });
        }
        function normalizeAsssets() {
            var _a, _b, _c, _d, _e, _f;
            return __assign(__assign({}, args.assets), { 
                // remove leading and trailing slashes from the path
                path: ((_a = args.assets) === null || _a === void 0 ? void 0 : _a.path)
                    ? (0, pulumi_1.output)((_b = args.assets) === null || _b === void 0 ? void 0 : _b.path).apply(function (v) {
                        return v.replace(/^\//, "").replace(/\/$/, "");
                    })
                    : undefined, purge: (0, pulumi_1.output)((_d = (_c = args.assets) === null || _c === void 0 ? void 0 : _c.purge) !== null && _d !== void 0 ? _d : true), 
                // normalize to /path format
                routes: ((_e = args.assets) === null || _e === void 0 ? void 0 : _e.routes)
                    ? (0, pulumi_1.output)((_f = args.assets) === null || _f === void 0 ? void 0 : _f.routes).apply(function (v) {
                        return v.map(function (route) { return "/" + route.replace(/^\//, "").replace(/\/$/, ""); });
                    })
                    : [] });
        }
        function createBucket() {
            var _a;
            if (assets.bucket)
                return;
            return new (bucket_js_1.Bucket.bind.apply(bucket_js_1.Bucket, __spreadArray([void 0], (0, component_js_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.assets, "".concat(name, "Assets"), { access: "cloudfront" }, { parent: self, retainOnDelete: false }), false)))();
        }
        function getBucketDetails() {
            var s3Bucket = bucket
                ? bucket.nodes.bucket
                : aws_1.s3.BucketV2.get("".concat(name, "Assets"), assets.bucket, undefined, {
                    parent: self,
                });
            return {
                bucketName: s3Bucket.bucket,
                bucketDomain: s3Bucket.bucketRegionalDomainName,
            };
        }
        function uploadAssets() {
            var _this = this;
            return (0, pulumi_1.all)([outputPath, assets, route]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
                var bucketFiles, fileOptions, filesProcessed, _loop_1, _i, _c, fileOption;
                var _this = this;
                var _d, _e;
                var outputPath = _b[0], assets = _b[1], route = _b[2];
                return __generator(this, function (_f) {
                    switch (_f.label) {
                        case 0:
                            bucketFiles = [];
                            fileOptions = (_d = assets === null || assets === void 0 ? void 0 : assets.fileOptions) !== null && _d !== void 0 ? _d : [
                                {
                                    files: "**",
                                    cacheControl: "max-age=31536000,public,immutable",
                                },
                                {
                                    files: "**/*.html",
                                    cacheControl: "max-age=0,no-cache,no-store,must-revalidate",
                                },
                            ];
                            filesProcessed = [];
                            _loop_1 = function (fileOption) {
                                var files, _g, _h, _j;
                                return __generator(this, function (_k) {
                                    switch (_k.label) {
                                        case 0:
                                            files = (0, glob_1.globSync)(fileOption.files, {
                                                cwd: path_1.default.resolve(outputPath),
                                                nodir: true,
                                                dot: true,
                                                ignore: __spreadArray([
                                                    ".sst/**"
                                                ], (typeof fileOption.ignore === "string"
                                                    ? [fileOption.ignore]
                                                    : (_e = fileOption.ignore) !== null && _e !== void 0 ? _e : []), true),
                                            }).filter(function (file) { return !filesProcessed.includes(file); });
                                            _h = (_g = bucketFiles.push).apply;
                                            _j = [bucketFiles];
                                            return [4 /*yield*/, Promise.all(files.map(function (file) { return __awaiter(_this, void 0, void 0, function () {
                                                    var source, content, hash;
                                                    var _a, _b, _c, _d;
                                                    return __generator(this, function (_e) {
                                                        switch (_e.label) {
                                                            case 0:
                                                                source = path_1.default.resolve(outputPath, file);
                                                                return [4 /*yield*/, fs_1.default.promises.readFile(source, "utf-8")];
                                                            case 1:
                                                                content = _e.sent();
                                                                hash = crypto_1.default
                                                                    .createHash("sha256")
                                                                    .update(content)
                                                                    .digest("hex");
                                                                return [2 /*return*/, {
                                                                        source: source,
                                                                        key: path_1.default.posix.join((_a = assets.path) !== null && _a !== void 0 ? _a : "", (_c = (_b = route === null || route === void 0 ? void 0 : route.pathPrefix) === null || _b === void 0 ? void 0 : _b.replace(/^\//, "")) !== null && _c !== void 0 ? _c : "", file),
                                                                        hash: hash,
                                                                        cacheControl: fileOption.cacheControl,
                                                                        contentType: (_d = fileOption.contentType) !== null && _d !== void 0 ? _d : (0, base_site_js_1.getContentType)(file, "UTF-8"),
                                                                    }];
                                                        }
                                                    });
                                                }); }))];
                                        case 1:
                                            _h.apply(_g, _j.concat([(_k.sent())]));
                                            filesProcessed.push.apply(filesProcessed, files);
                                            return [2 /*return*/];
                                    }
                                });
                            };
                            _i = 0, _c = fileOptions.reverse();
                            _f.label = 1;
                        case 1:
                            if (!(_i < _c.length)) return [3 /*break*/, 4];
                            fileOption = _c[_i];
                            return [5 /*yield**/, _loop_1(fileOption)];
                        case 2:
                            _f.sent();
                            _f.label = 3;
                        case 3:
                            _i++;
                            return [3 /*break*/, 1];
                        case 4: return [2 /*return*/, new bucket_files_js_1.BucketFiles("".concat(name, "AssetFiles"), {
                                bucketName: bucketName,
                                files: bucketFiles,
                                purge: assets.purge,
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
            var _this = this;
            var entries = (0, pulumi_1.all)([
                outputPath,
                assets,
                bucketDomain,
                errorPage,
                route,
            ]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
                var kvEntries, dirs;
                var outputPath = _b[0], assets = _b[1], bucketDomain = _b[2], errorPage = _b[3], route = _b[4];
                return __generator(this, function (_c) {
                    kvEntries = {};
                    dirs = [];
                    fs_1.default.readdirSync(outputPath, { withFileTypes: true }).forEach(function (item) {
                        if (item.isDirectory()) {
                            dirs.push(path_1.default.posix.join("/", item.name));
                            return;
                        }
                        kvEntries[path_1.default.posix.join("/", item.name)] = "s3";
                    });
                    kvEntries["metadata"] = JSON.stringify({
                        base: (route === null || route === void 0 ? void 0 : route.pathPrefix) === "/" ? undefined : route === null || route === void 0 ? void 0 : route.pathPrefix,
                        custom404: errorPage,
                        s3: {
                            domain: bucketDomain,
                            dir: assets.path ? "/" + assets.path : "",
                            routes: __spreadArray(__spreadArray([], assets.routes, true), dirs, true),
                        },
                    });
                    return [2 /*return*/, kvEntries];
                });
            }); });
            return new kv_keys_js_1.KvKeys("".concat(name, "KvKeys"), {
                store: kvStoreArn,
                namespace: kvNamespace,
                entries: entries,
                purge: assets.purge,
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
        function createRequestKvStore() {
            return (0, pulumi_1.output)(args.edge).apply(function (edge) {
                var viewerRequest = edge === null || edge === void 0 ? void 0 : edge.viewerRequest;
                if (viewerRequest === null || viewerRequest === void 0 ? void 0 : viewerRequest.kvStore)
                    return (0, pulumi_1.output)(viewerRequest === null || viewerRequest === void 0 ? void 0 : viewerRequest.kvStore);
                return new aws_1.cloudfront.KeyValueStore("".concat(name, "KvStore"), {}, { parent: self }).arn;
            });
        }
        function createRequestFunction() {
            return (0, pulumi_1.output)(args.edge).apply(function (edge) {
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
            return (0, pulumi_1.output)(args.edge).apply(function (edge) {
                var _a, _b;
                var userConfig = edge === null || edge === void 0 ? void 0 : edge.viewerResponse;
                var userInjection = userConfig === null || userConfig === void 0 ? void 0 : userConfig.injection;
                var kvStoreArn = (_a = userConfig === null || userConfig === void 0 ? void 0 : userConfig.kvStore) !== null && _a !== void 0 ? _a : (_b = userConfig === null || userConfig === void 0 ? void 0 : userConfig.kvStores) === null || _b === void 0 ? void 0 : _b[0];
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
            return new (cdn_js_1.Cdn.bind.apply(cdn_js_1.Cdn, __spreadArray([void 0], (0, component_js_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.cdn, "".concat(name, "Cdn"), {
                comment: "".concat(name, " site"),
                domain: args.domain,
                origins: [
                    {
                        originId: "default",
                        domainName: "placeholder.sst.dev",
                        customOriginConfig: {
                            httpPort: 80,
                            httpsPort: 443,
                            originProtocolPolicy: "https-only",
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
                    // CloudFront's managed CachingOptimized policy
                    cachePolicyId: "658327ea-f89d-4fab-a63d-7e88639e58f6",
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
        function createInvalidation() {
            (0, pulumi_1.all)([outputPath, args.assets, args.invalidation]).apply(function (_a) {
                var outputPath = _a[0], assets = _a[1], invalidationRaw = _a[2];
                // Normalize invalidation
                if (invalidationRaw === false)
                    return;
                var invalidation = __assign({ wait: false, paths: "all" }, invalidationRaw);
                // Build invalidation paths
                var invalidationPaths = invalidation.paths === "all" ? ["/*"] : invalidation.paths;
                if (invalidationPaths.length === 0)
                    return;
                // Calculate a hash based on the contents of the S3 files. This will be
                // used to determine if we need to invalidate our CloudFront cache.
                //
                // The below options are needed to support following symlinks when building zip files:
                // - nodir: This will prevent symlinks themselves from being copied into the zip.
                // - follow: This will follow symlinks and copy the files within.
                var hash = crypto_1.default.createHash("md5");
                hash.update(JSON.stringify(assets !== null && assets !== void 0 ? assets : {}));
                (0, glob_1.globSync)("**", {
                    dot: true,
                    nodir: true,
                    follow: true,
                    cwd: path_1.default.resolve(outputPath),
                }).forEach(function (filePath) {
                    return hash.update(fs_1.default.readFileSync(path_1.default.resolve(outputPath, filePath), "utf-8"));
                });
                new distribution_invalidation_js_1.DistributionInvalidation("".concat(name, "Invalidation"), {
                    distributionId: distributionId,
                    paths: invalidationPaths,
                    version: hash.digest("hex"),
                    wait: invalidation.wait,
                }, {
                    parent: self,
                    dependsOn: __spreadArray([assetsUploaded, kvUpdated], invalidationDependsOn, true),
                });
            });
        }
        return _this;
    }
    Object.defineProperty(StaticSite.prototype, "url", {
        /**
         * The URL of the website.
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
    Object.defineProperty(StaticSite.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
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
    StaticSite.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
            },
        };
    };
    return StaticSite;
}(component_js_1.Component));
exports.StaticSite = StaticSite;
var __pulumiType = "sst:aws:StaticSite";
// @ts-expect-error
StaticSite.__pulumiType = __pulumiType;
var templateObject_1;
