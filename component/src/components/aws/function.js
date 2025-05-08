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
exports.Function = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var crypto_1 = require("crypto");
var archiver_1 = require("archiver");
var glob_1 = require("glob");
var pulumi_1 = require("@pulumi/pulumi");
var bootstrap_js_1 = require("./helpers/bootstrap.js");
var duration_js_1 = require("../duration.js");
var size_js_1 = require("../size.js");
var component_js_1 = require("../component.js");
var link_js_1 = require("../link.js");
var error_js_1 = require("../error.js");
var naming_js_1 = require("../naming.js");
var logging_js_1 = require("./logging.js");
var aws_1 = require("@pulumi/aws");
var permission_js_1 = require("./permission.js");
var vpc_js_1 = require("./vpc.js");
var docker_build_1 = require("@pulumi/docker-build");
var rpc_js_1 = require("../rpc/rpc.js");
var arn_js_1 = require("./helpers/arn.js");
var random_1 = require("@pulumi/random");
var lazy_js_1 = require("../../util/lazy.js");
var efs_js_1 = require("./efs.js");
var function_environment_update_js_1 = require("./providers/function-environment-update.js");
var warn_js_1 = require("../../util/warn.js");
var router_js_1 = require("./router.js");
var kv_routes_update_js_1 = require("./providers/kv-routes-update.js");
var kv_keys_js_1 = require("./providers/kv-keys.js");
/**
 * The `Function` component lets you add serverless functions to your app.
 * It uses [AWS Lambda](https://aws.amazon.com/lambda/).
 *
 * #### Supported runtimes
 *
 * Currently supports **Node.js** and **Golang** functions. Python and Rust are community
 * supported and are currently a work in progress. Other runtimes are on the roadmap.
 *
 * @example
 *
 * #### Minimal example
 *
 *
 * <Tabs>
 *   <TabItem label="Node">
 *   Pass in the path to your handler function.
 *
 *   ```ts title="sst.config.ts"
 *   new sst.aws.Function("MyFunction", {
 *     handler: "src/lambda.handler"
 *   });
 *   ```
 *   </TabItem>
 *   <TabItem label="Go">
 *   Pass in the directory to your Go app.
 *
 *   ```ts title="sst.config.ts"
 *   new sst.aws.Function("MyFunction", {
 *     runtime: "go",
 *     handler: "./src"
 *   });
 *   ```
 *   </TabItem>
 *   <TabItem label="Rust">
 *   Pass in the directory where your Cargo.toml lives.
 *
 *   ```ts title="sst.config.ts"
 *   new sst.aws.Function("MyFunction", {
 *     runtime: "runtime",
 *     handler: "./crates/api/"
 *   });
 *   ```
 *   </TabItem>
 * </Tabs>
 *
 * #### Set additional config
 *
 * Pass in additional Lambda config.
 *
 * ```ts {3,4} title="sst.config.ts"
 * new sst.aws.Function("MyFunction", {
 *   handler: "src/lambda.handler",
 *   timeout: "3 minutes",
 *   memory: "1024 MB"
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to the function. This will grant permissions
 * to the resources and allow you to access it in your handler.
 *
 * ```ts {5} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.Function("MyFunction", {
 *   handler: "src/lambda.handler",
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your handler.
 *
 * <Tabs>
 *   <TabItem label="Node">
 *   ```ts title="src/lambda.ts"
 *   import { Resource } from "sst";
 *
 *   console.log(Resource.MyBucket.name);
 *   ```
 *   </TabItem>
 *   <TabItem label="Go">
 *   ```go title="src/main.go"
 *   import (
 *     "github.com/sst/sst/v3/sdk/golang/resource"
 *   )
 *
 *   resource.Get("MyBucket", "name")
 *   ```
 *   </TabItem>
 *   <TabItem label="Rust">
 *   ```rust title="src/main.rs"
 *   use sst_sdk::Resource;
 *   #[derive(serde::Deserialize, Debug)]
 *   struct Bucket {
 *      name: String,
 *   }
 *
 *   let resource = Resource::init().unwrap();
 *   let Bucket { name } = resource.get("Bucket").unwrap();
 *   ```
 *   </TabItem>
 * </Tabs>
 *
 * #### Set environment variables
 *
 * Set environment variables that you can read in your function. For example, using
 * `process.env` in your Node.js functions.
 *
 * ```ts {4} title="sst.config.ts"
 * new sst.aws.Function("MyFunction", {
 *   handler: "src/lambda.handler",
 *   environment: {
 *     DEBUG: "true"
 *   }
 * });
 * ```
 *
 * #### Enable function URLs
 *
 * Enable function URLs to invoke the function over HTTP.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.aws.Function("MyFunction", {
 *   handler: "src/lambda.handler",
 *   url: true
 * });
 * ```
 *
 * #### Bundling
 *
 * Customize how SST uses [esbuild](https://esbuild.github.io/) to bundle your Node.js
 * functions with the `nodejs` property.
 *
 * ```ts title="sst.config.ts" {3-5}
 * new sst.aws.Function("MyFunction", {
 *   handler: "src/lambda.handler",
 *   nodejs: {
 *     install: ["pg"]
 *   }
 * });
 * ```
 *
 * Or override it entirely by passing in your own function `bundle`.
 */
var Function = /** @class */ (function (_super) {
    __extends(Function, _super);
    function Function(name, args, opts) {
        var _a, _b;
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        _this.constructorName = name;
        var parent = _this;
        var dev = normalizeDev();
        var isContainer = (0, pulumi_1.all)([args.python, dev]).apply(function (_a) {
            var _b;
            var python = _a[0], dev = _a[1];
            return !dev && ((_b = python === null || python === void 0 ? void 0 : python.container) !== null && _b !== void 0 ? _b : false);
        });
        var partition = (0, aws_1.getPartitionOutput)({}, opts).partition;
        var region = (0, aws_1.getRegionOutput)({}, opts).name;
        var bootstrapData = region.apply(function (region) { return bootstrap_js_1.bootstrap.forRegion(region); });
        var injections = normalizeInjections();
        var runtime = (0, pulumi_1.output)((_a = args.runtime) !== null && _a !== void 0 ? _a : "nodejs20.x");
        var timeout = normalizeTimeout();
        var memory = normalizeMemory();
        var storage = (0, pulumi_1.output)(args.storage).apply(function (v) { return v !== null && v !== void 0 ? v : "512 MB"; });
        var architecture = (0, pulumi_1.output)(args.architecture).apply(function (v) { return v !== null && v !== void 0 ? v : "x86_64"; });
        var environment = normalizeEnvironment();
        var streaming = normalizeStreaming();
        var logging = normalizeLogging();
        var volume = normalizeVolume();
        var url = normalizeUrl();
        var copyFiles = normalizeCopyFiles();
        var policies = (0, pulumi_1.output)((_b = args.policies) !== null && _b !== void 0 ? _b : []);
        var vpc = normalizeVpc();
        var linkData = buildLinkData();
        var linkPermissions = buildLinkPermissions();
        var _c = buildHandler(), bundle = _c.bundle, handler0 = _c.handler, sourcemaps = _c.sourcemaps;
        var _d = buildHandlerWrapper(), handler = _d.handler, wrapper = _d.wrapper;
        var role = createRole();
        var imageAsset = createImageAsset();
        var logGroup = createLogGroup();
        var zipAsset = createZipAsset();
        var fn = createFunction();
        var urlEndpoint = createUrl();
        createProvisioned();
        var eventInvokeConfig = createEventInvokeConfig();
        var links = linkData.apply(function (input) { return input.map(function (item) { return item.name; }); });
        _this.function = fn;
        _this.role = role;
        _this.logGroup = logGroup;
        _this.urlEndpoint = urlEndpoint;
        _this.eventInvokeConfig = eventInvokeConfig;
        var buildInput = (0, pulumi_1.output)({
            functionID: name,
            handler: args.handler,
            bundle: args.bundle,
            logGroup: logGroup.apply(function (l) { return l === null || l === void 0 ? void 0 : l.name; }),
            encryptionKey: Function.encryptionKey().base64,
            runtime: runtime,
            links: (0, pulumi_1.output)(linkData).apply(function (input) {
                return Object.fromEntries(input.map(function (item) { return [item.name, item.properties]; }));
            }),
            copyFiles: copyFiles,
            properties: (0, pulumi_1.output)({ nodejs: args.nodejs, python: args.python }).apply(function (val) { return (__assign(__assign({}, (val.nodejs || val.python)), { architecture: architecture })); }),
            dev: dev,
        });
        buildInput.apply(function (input) { return __awaiter(_this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!input.dev)
                            return [2 /*return*/];
                        return [4 /*yield*/, rpc_js_1.rpc.call("Runtime.AddTarget", input)];
                    case 1:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        }); });
        _this.registerOutputs({
            _live: (0, pulumi_1.unsecret)((0, pulumi_1.output)(dev).apply(function (dev) {
                if (!dev)
                    return undefined;
                return (0, pulumi_1.all)([
                    name,
                    links,
                    args.handler,
                    args.bundle,
                    args.runtime,
                    args.nodejs,
                    copyFiles,
                ]).apply(function (_a) {
                    var name = _a[0], links = _a[1], handler = _a[2], bundle = _a[3], runtime = _a[4], nodejs = _a[5], copyFiles = _a[6];
                    return {
                        functionID: name,
                        links: links,
                        handler: handler,
                        bundle: bundle,
                        runtime: runtime || "nodejs20.x",
                        copyFiles: copyFiles,
                        properties: nodejs,
                    };
                });
            })),
            _metadata: {
                handler: args.handler,
                internal: args._skipMetadata,
                dev: dev,
            },
            _hint: args._skipHint ? undefined : urlEndpoint,
        });
        function normalizeDev() {
            return (0, pulumi_1.all)([args.dev, args.live]).apply(function (_a) {
                var d = _a[0], l = _a[1];
                return $dev && d !== false && l !== false;
            });
        }
        function normalizeInjections() {
            return (0, pulumi_1.output)(args.injections).apply(function (injections) { return injections !== null && injections !== void 0 ? injections : []; });
        }
        function normalizeTimeout() {
            return (0, pulumi_1.output)(args.timeout).apply(function (timeout) { return timeout !== null && timeout !== void 0 ? timeout : "20 seconds"; });
        }
        function normalizeMemory() {
            return (0, pulumi_1.output)(args.memory).apply(function (memory) { return memory !== null && memory !== void 0 ? memory : "1024 MB"; });
        }
        function normalizeEnvironment() {
            var _this = this;
            return (0, pulumi_1.all)([
                args.environment,
                dev,
                bootstrapData,
                Function.encryptionKey().base64,
                args.link,
            ]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
                var result, _i, _c, linkable, def, _d, _e, item, appsync;
                var environment = _b[0], dev = _b[1], bootstrap = _b[2], key = _b[3], link = _b[4];
                return __generator(this, function (_f) {
                    switch (_f.label) {
                        case 0:
                            result = environment !== null && environment !== void 0 ? environment : {};
                            result.SST_RESOURCE_App = JSON.stringify({
                                name: $app.name,
                                stage: $app.stage,
                            });
                            for (_i = 0, _c = link || []; _i < _c.length; _i++) {
                                linkable = _c[_i];
                                if (!link_js_1.Link.isLinkable(linkable))
                                    continue;
                                def = linkable.getSSTLink();
                                for (_d = 0, _e = def.include || []; _d < _e.length; _d++) {
                                    item = _e[_d];
                                    if (item.type === "environment")
                                        Object.assign(result, item.env);
                                }
                            }
                            result.SST_KEY = key;
                            result.SST_KEY_FILE = "resource.enc";
                            if (!dev) return [3 /*break*/, 2];
                            return [4 /*yield*/, Function.appsync()];
                        case 1:
                            appsync = _f.sent();
                            result.SST_REGION = process.env.SST_AWS_REGION;
                            result.SST_APPSYNC_HTTP = appsync.http;
                            result.SST_APPSYNC_REALTIME = appsync.realtime;
                            result.SST_FUNCTION_ID = name;
                            result.SST_APP = $app.name;
                            result.SST_STAGE = $app.stage;
                            result.SST_ASSET_BUCKET = bootstrap.asset;
                            if (process.env.SST_FUNCTION_TIMEOUT) {
                                result.SST_FUNCTION_TIMEOUT = process.env.SST_FUNCTION_TIMEOUT;
                            }
                            _f.label = 2;
                        case 2: return [2 /*return*/, result];
                    }
                });
            }); });
        }
        function normalizeStreaming() {
            return (0, pulumi_1.output)(args.streaming).apply(function (streaming) { return streaming !== null && streaming !== void 0 ? streaming : false; });
        }
        function normalizeLogging() {
            return (0, pulumi_1.output)(args.logging).apply(function (logging) {
                var _a, _b;
                if (logging === false)
                    return undefined;
                if ((logging === null || logging === void 0 ? void 0 : logging.retention) && (logging === null || logging === void 0 ? void 0 : logging.logGroup)) {
                    throw new error_js_1.VisibleError("Cannot set both \"logging.retention\" and \"logging.logGroup\"");
                }
                return {
                    logGroup: logging === null || logging === void 0 ? void 0 : logging.logGroup,
                    retention: (_a = logging === null || logging === void 0 ? void 0 : logging.retention) !== null && _a !== void 0 ? _a : "1 month",
                    format: (_b = logging === null || logging === void 0 ? void 0 : logging.format) !== null && _b !== void 0 ? _b : "text",
                };
            });
        }
        function normalizeVolume() {
            if (!args.volume)
                return;
            return (0, pulumi_1.output)(args.volume).apply(function (volume) {
                var _a;
                return ({
                    efs: volume.efs instanceof efs_js_1.Efs
                        ? volume.efs.nodes.accessPoint.arn
                        : (0, pulumi_1.output)(volume.efs),
                    path: (_a = volume.path) !== null && _a !== void 0 ? _a : "/mnt/efs",
                });
            });
        }
        function normalizeUrl() {
            return (0, pulumi_1.output)(args.url).apply(function (url) {
                var _a;
                if (url === false || url === undefined)
                    return;
                if (url === true) {
                    url = {};
                }
                // normalize authorization
                var defaultAuthorization = "none";
                var authorization = (_a = url.authorization) !== null && _a !== void 0 ? _a : defaultAuthorization;
                // normalize cors
                var defaultCors = {
                    allowHeaders: ["*"],
                    allowMethods: ["*"],
                    allowOrigins: ["*"],
                };
                var cors = url.cors === false
                    ? undefined
                    : url.cors === true || url.cors === undefined
                        ? defaultCors
                        : __assign(__assign(__assign({}, defaultCors), url.cors), { maxAge: url.cors.maxAge && (0, duration_js_1.toSeconds)(url.cors.maxAge) });
                return {
                    authorization: authorization,
                    cors: cors,
                    route: (0, router_js_1.normalizeRouteArgs)(url.router, url.route),
                };
            });
        }
        function normalizeCopyFiles() {
            var _this = this;
            var _a;
            return (0, pulumi_1.output)((_a = args.copyFiles) !== null && _a !== void 0 ? _a : []).apply(function (copyFiles) {
                return Promise.all(copyFiles.map(function (entry) { return __awaiter(_this, void 0, void 0, function () {
                    var from, to, stats, isDir;
                    return __generator(this, function (_a) {
                        switch (_a.label) {
                            case 0:
                                from = path_1.default.join($cli.paths.root, entry.from);
                                to = entry.to || entry.from;
                                if (path_1.default.isAbsolute(to)) {
                                    throw new error_js_1.VisibleError("Copy destination path \"".concat(to, "\" must be relative"));
                                }
                                return [4 /*yield*/, fs_1.default.promises.stat(from)];
                            case 1:
                                stats = _a.sent();
                                isDir = stats.isDirectory();
                                return [2 /*return*/, { from: from, to: to, isDir: isDir }];
                        }
                    });
                }); }));
            });
        }
        function normalizeVpc() {
            // "vpc" is undefined
            if (!args.vpc)
                return;
            // "vpc" is a Vpc component
            if (args.vpc instanceof vpc_js_1.Vpc) {
                var result_1 = {
                    privateSubnets: args.vpc.privateSubnets,
                    securityGroups: args.vpc.securityGroups,
                };
                return (0, pulumi_1.all)([
                    args.vpc.id,
                    args.vpc.nodes.natGateways,
                    args.vpc.nodes.natInstances,
                ]).apply(function (_a) {
                    var id = _a[0], natGateways = _a[1], natInstances = _a[2];
                    if (natGateways.length === 0 && natInstances.length === 0) {
                        (0, warn_js_1.warnOnce)("\nWarning: One or more functions are deployed in the \"".concat(id, "\" VPC, which does not have a NAT gateway. As a result, these functions cannot access the internet. If your functions need internet access, enable it by setting the \"nat\" prop on the \"Vpc\" component.\n"));
                    }
                    return result_1;
                });
            }
            return (0, pulumi_1.output)(args.vpc).apply(function (vpc) {
                // "vpc" is object
                if (vpc.subnets) {
                    throw new error_js_1.VisibleError("The \"vpc.subnets\" property has been renamed to \"vpc.privateSubnets\". Update your code to use \"vpc.privateSubnets\" instead.");
                }
                return vpc;
            });
        }
        function buildLinkData() {
            return (0, pulumi_1.output)(args.link || []).apply(function (links) { return link_js_1.Link.build(links); });
        }
        function buildLinkPermissions() {
            return link_js_1.Link.getInclude("aws.permission", args.link);
        }
        function buildHandler() {
            var _this = this;
            return (0, pulumi_1.all)([runtime, dev, isContainer]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
                var buildResult;
                var _this = this;
                var runtime = _b[0], dev = _b[1], isContainer = _b[2];
                return __generator(this, function (_c) {
                    if (dev) {
                        return [2 /*return*/, {
                                handler: "bootstrap",
                                bundle: path_1.default.join($cli.paths.platform, "dist", "bridge"),
                            }];
                    }
                    buildResult = buildInput.apply(function (input) { return __awaiter(_this, void 0, void 0, function () {
                        var result;
                        var _a;
                        return __generator(this, function (_b) {
                            switch (_b.label) {
                                case 0: return [4 /*yield*/, rpc_js_1.rpc.call("Runtime.Build", __assign(__assign({}, input), { isContainer: isContainer }))];
                                case 1:
                                    result = _b.sent();
                                    if (result.errors.length > 0) {
                                        throw new Error(result.errors.join("\n"));
                                    }
                                    if (!((_a = args.hook) === null || _a === void 0 ? void 0 : _a.postbuild)) return [3 /*break*/, 3];
                                    return [4 /*yield*/, args.hook.postbuild(result.out)];
                                case 2:
                                    _b.sent();
                                    _b.label = 3;
                                case 3: return [2 /*return*/, result];
                            }
                        });
                    }); });
                    return [2 /*return*/, {
                            handler: buildResult.handler,
                            bundle: buildResult.out,
                            sourcemaps: buildResult.sourcemaps,
                        }];
                });
            }); });
        }
        function buildHandlerWrapper() {
            var _this = this;
            var ret = (0, pulumi_1.all)([
                dev,
                bundle,
                handler0,
                linkData,
                streaming,
                injections,
                runtime,
            ]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
                var hasUserInjections, parsed, handlerDir, oldHandlerFileName, oldHandlerFunction, newHandlerFileName, newHandlerFunction, newHandlerFileExt, split;
                var dev = _b[0], bundle = _b[1], handler = _b[2], linkData = _b[3], streaming = _b[4], injections = _b[5], runtime = _b[6];
                return __generator(this, function (_c) {
                    if (dev)
                        return [2 /*return*/, { handler: handler }];
                    if (!runtime.startsWith("nodejs")) {
                        return [2 /*return*/, { handler: handler }];
                    }
                    hasUserInjections = injections.length > 0;
                    if (!hasUserInjections)
                        return [2 /*return*/, { handler: handler }];
                    parsed = path_1.default.posix.parse(handler);
                    handlerDir = parsed.dir;
                    oldHandlerFileName = parsed.name;
                    oldHandlerFunction = parsed.ext.replace(/^\./, "");
                    newHandlerFileName = "server-index";
                    newHandlerFunction = "handler";
                    newHandlerFileExt = [".js", ".mjs", ".cjs"].find(function (ext) {
                        return fs_1.default.existsSync(path_1.default.join(bundle, handlerDir, oldHandlerFileName + ext));
                    });
                    if (!newHandlerFileExt) {
                        throw new error_js_1.VisibleError("Could not find handler file \"".concat(handler, "\" for function \"").concat(name, "\""));
                    }
                    split = injections.reduce(function (acc, item) {
                        if (item.startsWith("outer:")) {
                            acc.outer.push(item.substring("outer:".length));
                            return acc;
                        }
                        acc.inner.push(item);
                        return acc;
                    }, { outer: [], inner: [] });
                    return [2 /*return*/, {
                            handler: path_1.default.posix.join(handlerDir, "".concat(newHandlerFileName, ".").concat(newHandlerFunction)),
                            wrapper: {
                                name: path_1.default.posix.join(handlerDir, "".concat(newHandlerFileName, ".mjs")),
                                content: streaming
                                    ? __spreadArray(__spreadArray(__spreadArray(__spreadArray([], split.outer, true), [
                                        "export const ".concat(newHandlerFunction, " = awslambda.streamifyResponse(async (event, responseStream, context) => {")
                                    ], false), split.inner, true), [
                                        "  const { ".concat(oldHandlerFunction, ": rawHandler} = await import(\"./").concat(oldHandlerFileName).concat(newHandlerFileExt, "\");"),
                                        "  return rawHandler(event, responseStream, context);",
                                        "});",
                                    ], false).join("\n")
                                    : __spreadArray(__spreadArray(__spreadArray(__spreadArray([], split.outer, true), [
                                        "export const ".concat(newHandlerFunction, " = async (event, context) => {")
                                    ], false), split.inner, true), [
                                        "  const { ".concat(oldHandlerFunction, ": rawHandler} = await import(\"./").concat(oldHandlerFileName).concat(newHandlerFileExt, "\");"),
                                        "  return rawHandler(event, context);",
                                        "};",
                                    ], false).join("\n"),
                            },
                        }];
                });
            }); });
            return {
                handler: ret.handler,
                wrapper: ret.wrapper,
            };
        }
        function createRole() {
            var _a;
            var _b;
            if (args.role) {
                return aws_1.iam.Role.get("".concat(name, "Role"), (0, pulumi_1.output)(args.role).apply(arn_js_1.parseRoleArn).roleName, {}, { parent: parent });
            }
            var policy = (0, pulumi_1.all)([args.permissions || [], linkPermissions, dev]).apply(function (_a) {
                var argsPermissions = _a[0], linkPermissions = _a[1], dev = _a[2];
                return aws_1.iam.getPolicyDocumentOutput({
                    statements: __spreadArray(__spreadArray(__spreadArray([], argsPermissions, true), linkPermissions, true), (dev
                        ? [
                            {
                                effect: "allow",
                                actions: ["appsync:*"],
                                resources: ["*"],
                            },
                            {
                                effect: "allow",
                                actions: ["s3:*"],
                                resources: [
                                    (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["arn:", ":s3:::", ""], ["arn:", ":s3:::", ""])), partition, bootstrapData.asset),
                                    (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["arn:", ":s3:::", "/*"], ["arn:", ":s3:::", "/*"])), partition, bootstrapData.asset),
                                ],
                            },
                        ]
                        : []), true).map(function (item) { return ({
                        effect: (function () {
                            var _a;
                            var effect = (_a = item.effect) !== null && _a !== void 0 ? _a : "allow";
                            return effect.charAt(0).toUpperCase() + effect.slice(1);
                        })(),
                        actions: item.actions,
                        resources: item.resources,
                    }); }),
                });
            });
            return new ((_a = aws_1.iam.Role).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.role, "".concat(name, "Role"), {
                assumeRolePolicy: !dev
                    ? aws_1.iam.assumeRolePolicyForPrincipal({
                        Service: "lambda.amazonaws.com",
                    })
                    : aws_1.iam.getPolicyDocumentOutput({
                        statements: [
                            {
                                actions: ["sts:AssumeRole"],
                                principals: [
                                    {
                                        type: "Service",
                                        identifiers: ["lambda.amazonaws.com"],
                                    },
                                    {
                                        type: "AWS",
                                        identifiers: [
                                            (0, pulumi_1.interpolate)(templateObject_3 || (templateObject_3 = __makeTemplateObject(["arn:", ":iam::", ":root"], ["arn:", ":iam::", ":root"])), partition, (0, aws_1.getCallerIdentityOutput)({}, opts).accountId),
                                        ],
                                    },
                                ],
                            },
                        ],
                    }).json,
                // if there are no statements, do not add an inline policy.
                // adding an inline policy with no statements will cause an error.
                inlinePolicies: policy.apply(function (_a) {
                    var statements = _a.statements;
                    return statements ? [{ name: "inline", policy: policy.json }] : [];
                }),
                managedPolicyArns: (0, pulumi_1.all)([logging, policies]).apply(function (_a) {
                    var logging = _a[0], policies = _a[1];
                    return __spreadArray(__spreadArray(__spreadArray([], policies, true), (logging
                        ? [
                            (0, pulumi_1.interpolate)(templateObject_4 || (templateObject_4 = __makeTemplateObject(["arn:", ":iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"], ["arn:", ":iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"])), partition),
                        ]
                        : []), true), (vpc
                        ? [
                            (0, pulumi_1.interpolate)(templateObject_5 || (templateObject_5 = __makeTemplateObject(["arn:", ":iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"], ["arn:", ":iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"])), partition),
                        ]
                        : []), true);
                }),
            }, { parent: parent }), false)))();
        }
        function createImageAsset() {
            // The build artifact directory already exists, with all the user code and
            // config files. It also has the dockerfile, we need to now just build and push to
            // the container registry.
            return (0, pulumi_1.all)([isContainer, dev, bundle]).apply(function (_a) {
                var isContainer = _a[0], dev = _a[1], bundle = _a[2];
                if (!isContainer || dev)
                    return;
                var authToken = aws_1.ecr.getAuthorizationTokenOutput({
                    registryId: bootstrapData.assetEcrRegistryId,
                });
                return new docker_build_1.Image("".concat(name, "Image"), {
                    tags: [$interpolate(templateObject_6 || (templateObject_6 = __makeTemplateObject(["", ":latest"], ["", ":latest"])), bootstrapData.assetEcrUrl)],
                    context: {
                        location: path_1.default.join($cli.paths.work, "artifacts", "".concat(name, "-src")),
                    },
                    cacheFrom: [
                        {
                            registry: {
                                ref: $interpolate(templateObject_7 || (templateObject_7 = __makeTemplateObject(["", ":", "-cache"], ["", ":", "-cache"])), bootstrapData.assetEcrUrl, name),
                            },
                        },
                    ],
                    cacheTo: [
                        {
                            registry: {
                                ref: $interpolate(templateObject_8 || (templateObject_8 = __makeTemplateObject(["", ":", "-cache"], ["", ":", "-cache"])), bootstrapData.assetEcrUrl, name),
                                imageManifest: true,
                                ociMediaTypes: true,
                                mode: "max",
                            },
                        },
                    ],
                    platforms: [
                        architecture.apply(function (v) {
                            return v === "arm64" ? "linux/arm64" : "linux/amd64";
                        }),
                    ],
                    push: true,
                    registries: [
                        authToken.apply(function (authToken) { return ({
                            address: authToken.proxyEndpoint,
                            username: authToken.userName,
                            password: (0, pulumi_1.secret)(authToken.password),
                        }); }),
                    ],
                }, { parent: parent });
            });
        }
        function createZipAsset() {
            var _this = this;
            // Note: cannot point the bundle to the `.open-next/server-function`
            //       b/c the folder contains node_modules. And pnpm node_modules
            //       contains symlinks. Pulumi cannot zip symlinks correctly.
            //       We will zip the folder ourselves.
            return (0, pulumi_1.all)([
                bundle,
                wrapper,
                sourcemaps,
                copyFiles,
                isContainer,
                logGroup.apply(function (l) { return l === null || l === void 0 ? void 0 : l.arn; }),
                dev,
            ]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
                var zipPath, hash, _c, _d, hashValue, assetBucket, index, _i, sourcemaps_1, file;
                var _this = this;
                var bundle = _b[0], wrapper = _b[1], sourcemaps = _b[2], copyFiles = _b[3], isContainer = _b[4], logGroupArn = _b[5], dev = _b[6];
                return __generator(this, function (_e) {
                    switch (_e.label) {
                        case 0:
                            if (isContainer)
                                return [2 /*return*/];
                            zipPath = path_1.default.resolve($cli.paths.work, "artifacts", name, "code.zip");
                            return [4 /*yield*/, fs_1.default.promises.mkdir(path_1.default.dirname(zipPath), {
                                    recursive: true,
                                })];
                        case 1:
                            _e.sent();
                            return [4 /*yield*/, new Promise(function (resolve, reject) { return __awaiter(_this, void 0, void 0, function () {
                                    var ws, archive, files, _loop_1, _i, _a, item, _b, files_1, file;
                                    return __generator(this, function (_c) {
                                        switch (_c.label) {
                                            case 0:
                                                ws = fs_1.default.createWriteStream(zipPath);
                                                archive = (0, archiver_1.default)("zip", {
                                                    // Ensure deterministic zip file hashes
                                                    // https://github.com/archiverjs/node-archiver/issues/397#issuecomment-554327338
                                                    statConcurrency: 1,
                                                });
                                                archive.on("warning", reject);
                                                archive.on("error", reject);
                                                // archive has been finalized and the output file descriptor has closed, resolve promise
                                                // this has to be done before calling `finalize` since the events may fire immediately after.
                                                // see https://www.npmjs.com/package/archiver
                                                ws.once("close", function () {
                                                    resolve(zipPath);
                                                });
                                                archive.pipe(ws);
                                                files = [];
                                                _loop_1 = function (item) {
                                                    var found;
                                                    return __generator(this, function (_d) {
                                                        switch (_d.label) {
                                                            case 0:
                                                                if (!item.isDir) {
                                                                    files.push({
                                                                        from: item.from,
                                                                        to: item.to,
                                                                    });
                                                                }
                                                                return [4 /*yield*/, (0, glob_1.glob)("**", {
                                                                        cwd: item.from,
                                                                        dot: true,
                                                                        ignore: (sourcemaps === null || sourcemaps === void 0 ? void 0 : sourcemaps.map(function (item) { return path_1.default.relative(bundle, item); })) || [],
                                                                    })];
                                                            case 1:
                                                                found = _d.sent();
                                                                files.push.apply(files, found.map(function (file) { return ({
                                                                    from: path_1.default.join(item.from, file),
                                                                    to: path_1.default.join(item.to, file),
                                                                }); }));
                                                                return [2 /*return*/];
                                                        }
                                                    });
                                                };
                                                _i = 0, _a = __spreadArray([
                                                    {
                                                        from: bundle,
                                                        to: ".",
                                                        isDir: true,
                                                    }
                                                ], (!dev ? copyFiles : []), true);
                                                _c.label = 1;
                                            case 1:
                                                if (!(_i < _a.length)) return [3 /*break*/, 4];
                                                item = _a[_i];
                                                return [5 /*yield**/, _loop_1(item)];
                                            case 2:
                                                _c.sent();
                                                _c.label = 3;
                                            case 3:
                                                _i++;
                                                return [3 /*break*/, 1];
                                            case 4:
                                                files.sort(function (a, b) { return a.to.localeCompare(b.to); });
                                                for (_b = 0, files_1 = files; _b < files_1.length; _b++) {
                                                    file = files_1[_b];
                                                    archive.file(file.from, {
                                                        name: file.to,
                                                        date: new Date(0),
                                                    });
                                                }
                                                // Add handler wrapper into the zip
                                                if (wrapper) {
                                                    archive.append(wrapper.content, {
                                                        name: wrapper.name,
                                                        date: new Date(0),
                                                    });
                                                }
                                                return [4 /*yield*/, archive.finalize()];
                                            case 5:
                                                _c.sent();
                                                return [2 /*return*/];
                                        }
                                    });
                                }); })];
                        case 2:
                            _e.sent();
                            hash = crypto_1.default.createHash("sha256");
                            _d = (_c = hash).update;
                            return [4 /*yield*/, fs_1.default.promises.readFile(zipPath, "utf-8")];
                        case 3:
                            _d.apply(_c, [_e.sent()]);
                            hashValue = hash.digest("hex");
                            assetBucket = region.apply(function (region) {
                                return bootstrap_js_1.bootstrap.forRegion(region).then(function (d) { return d.asset; });
                            });
                            if (logGroupArn && sourcemaps) {
                                index = 0;
                                for (_i = 0, sourcemaps_1 = sourcemaps; _i < sourcemaps_1.length; _i++) {
                                    file = sourcemaps_1[_i];
                                    new aws_1.s3.BucketObjectv2("".concat(name, "Sourcemap").concat(index), {
                                        key: (0, pulumi_1.interpolate)(templateObject_9 || (templateObject_9 = __makeTemplateObject(["sourcemap/", "/", ".", ""], ["sourcemap/", "/", ".", ""])), logGroupArn, hashValue, path_1.default.basename(file)),
                                        bucket: assetBucket,
                                        source: new pulumi_1.asset.FileAsset(file),
                                    }, { parent: parent, retainOnDelete: true });
                                    index++;
                                }
                            }
                            return [2 /*return*/, new aws_1.s3.BucketObjectv2("".concat(name, "Code"), {
                                    key: (0, pulumi_1.interpolate)(templateObject_10 || (templateObject_10 = __makeTemplateObject(["assets/", "-code-", ".zip"], ["assets/", "-code-", ".zip"])), name, hashValue),
                                    bucket: assetBucket,
                                    source: new pulumi_1.asset.FileArchive(zipPath),
                                }, { parent: parent })];
                    }
                });
            }); });
        }
        function createLogGroup() {
            return logging.apply(function (logging) {
                var _a;
                var _b, _c;
                if (!logging)
                    return;
                if (logging.logGroup)
                    return;
                return new ((_a = aws_1.cloudwatch.LogGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.logGroup, "".concat(name, "LogGroup"), {
                    name: (0, pulumi_1.interpolate)(templateObject_11 || (templateObject_11 = __makeTemplateObject(["/aws/lambda/", ""], ["/aws/lambda/", ""])), (_c = args.name) !== null && _c !== void 0 ? _c : (0, naming_js_1.physicalName)(64, "".concat(name, "Function"))),
                    retentionInDays: logging_js_1.RETENTION[logging.retention],
                }, { parent: parent, ignoreChanges: ["name"] }), false)))();
            });
        }
        function createFunction() {
            return (0, pulumi_1.all)([
                logging,
                logGroup,
                isContainer,
                imageAsset,
                zipAsset,
                args.concurrency,
                dev,
            ]).apply(function (_a) {
                var _b, _c, _d, _e;
                var logging = _a[0], logGroup = _a[1], isContainer = _a[2], imageAsset = _a[3], zipAsset = _a[4], concurrency = _a[5], dev = _a[6];
                // This is a hack to avoid handler being marked as having propertyDependencies.
                // There is an unresolved bug in pulumi that causes issues when it does
                // @ts-expect-error
                handler.allResources = function () { return Promise.resolve(new Set()); };
                var transformed = (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.function, "".concat(name, "Function"), __assign({ name: args.name, description: (_c = args.description) !== null && _c !== void 0 ? _c : "", role: (_d = args.role) !== null && _d !== void 0 ? _d : role.arn, timeout: timeout.apply(function (timeout) { return (0, duration_js_1.toSeconds)(timeout); }), memorySize: memory.apply(function (memory) { return (0, size_js_1.toMBs)(memory); }), ephemeralStorage: { size: storage.apply(function (v) { return (0, size_js_1.toMBs)(v); }) }, environment: {
                        variables: environment,
                    }, architectures: [architecture], loggingConfig: logging && {
                        logFormat: logging.format === "json" ? "JSON" : "Text",
                        logGroup: (_e = logging.logGroup) !== null && _e !== void 0 ? _e : logGroup.name,
                    }, vpcConfig: vpc && {
                        securityGroupIds: vpc.securityGroups,
                        subnetIds: vpc.privateSubnets,
                    }, fileSystemConfig: volume && {
                        arn: volume.efs,
                        localMountPath: volume.path,
                    }, layers: args.layers, tags: args.tags, publish: (0, pulumi_1.output)(args.versioning).apply(function (v) { return v !== null && v !== void 0 ? v : false; }), reservedConcurrentExecutions: concurrency === null || concurrency === void 0 ? void 0 : concurrency.reserved }, (isContainer
                    ? {
                        packageType: "Image",
                        imageUri: imageAsset.ref.apply(function (ref) { return ref === null || ref === void 0 ? void 0 : ref.replace(":latest", ""); }),
                        imageConfig: {
                            commands: [
                                (0, pulumi_1.all)([handler, runtime]).apply(function (_a) {
                                    var handler = _a[0], runtime = _a[1];
                                    // If a python container image we have to rewrite the handler path so lambdaric is happy
                                    // This means no leading . and replace all / with .
                                    if (isContainer && runtime.includes("python")) {
                                        return handler
                                            .replace(/\.\//g, "")
                                            .replace(/\//g, ".");
                                    }
                                    return handler;
                                }),
                            ],
                        },
                    }
                    : {
                        packageType: "Zip",
                        s3Bucket: zipAsset.bucket,
                        s3Key: zipAsset.key,
                        handler: (0, pulumi_1.unsecret)(handler),
                        runtime: runtime.apply(function (v) {
                            return v === "go" || v === "rust" ? "provided.al2023" : v;
                        }),
                    })), { parent: parent });
                return new aws_1.lambda.Function(transformed[0], __assign(__assign({}, transformed[1]), (dev
                    ? {
                        description: transformed[1].description
                            ? (0, pulumi_1.output)(transformed[1].description).apply(function (v) { return "".concat(v.substring(0, 240), " (live)"); })
                            : "live",
                        runtime: "provided.al2023",
                        architectures: ["x86_64"],
                    }
                    : {})), transformed[2]);
            });
        }
        function createUrl() {
            return url.apply(function (url) {
                if (url === undefined)
                    return (0, pulumi_1.output)(undefined);
                // create the function url
                var fnUrl = new aws_1.lambda.FunctionUrl("".concat(name, "Url"), {
                    functionName: fn.name,
                    authorizationType: url.authorization === "iam" ? "AWS_IAM" : "NONE",
                    invokeMode: streaming.apply(function (streaming) {
                        return streaming ? "RESPONSE_STREAM" : "BUFFERED";
                    }),
                    cors: url.cors,
                }, { parent: parent });
                if (!url.route)
                    return fnUrl.functionUrl;
                // add router route
                var routeNamespace = crypto_1.default
                    .createHash("md5")
                    .update("".concat($app.name, "-").concat($app.stage, "-").concat(name))
                    .digest("hex")
                    .substring(0, 4);
                new kv_keys_js_1.KvKeys("".concat(name, "RouteKey"), {
                    store: url.route.routerKvStoreArn,
                    namespace: routeNamespace,
                    entries: fnUrl.functionUrl.apply(function (fnUrl) { return ({
                        metadata: JSON.stringify({
                            host: new URL(fnUrl).host,
                        }),
                    }); }),
                    purge: false,
                }, { parent: parent });
                new kv_routes_update_js_1.KvRoutesUpdate("".concat(name, "RoutesUpdate"), {
                    store: url.route.routerKvStoreArn,
                    namespace: url.route.routerKvNamespace,
                    key: "routes",
                    entry: url.route.apply(function (route) {
                        return ["url", routeNamespace, route.hostPattern, route.pathPrefix].join(",");
                    }),
                }, { parent: parent });
                return url.route.routerUrl;
            });
        }
        function createProvisioned() {
            return (0, pulumi_1.all)([args.concurrency, fn.publish]).apply(function (_a) {
                var concurrency = _a[0], publish = _a[1];
                if (!(concurrency === null || concurrency === void 0 ? void 0 : concurrency.provisioned) || concurrency.provisioned === 0) {
                    return;
                }
                if (publish !== true) {
                    throw new error_js_1.VisibleError("Provisioned concurrency requires function versioning. Set \"versioning: true\" to enable function versioning.");
                }
                return new aws_1.lambda.ProvisionedConcurrencyConfig("".concat(name, "Provisioned"), {
                    functionName: fn.name,
                    qualifier: fn.version,
                    provisionedConcurrentExecutions: concurrency.provisioned,
                }, { parent: parent });
            });
        }
        function createEventInvokeConfig() {
            var _a;
            var _b;
            if (args.retries === undefined) {
                return undefined;
            }
            return new ((_a = aws_1.lambda.FunctionEventInvokeConfig).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.eventInvokeConfig, "".concat(name, "EventInvokeConfig"), {
                functionName: fn.name,
                maximumRetryAttempts: args.retries,
            }, { parent: parent }), false)))();
        }
        return _this;
    }
    Object.defineProperty(Function.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The IAM Role the function will use.
                 */
                role: this.role,
                /**
                 * The AWS Lambda function.
                 */
                function: this.function,
                /**
                 * The CloudWatch Log Group the function logs are stored.
                 */
                logGroup: this.logGroup,
                /**
                 * The Function Event Invoke Config resource if retries are configured.
                 */
                eventInvokeConfig: this.eventInvokeConfig,
            };
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Function.prototype, "url", {
        /**
         * The Lambda function URL if `url` is enabled.
         */
        get: function () {
            return this.urlEndpoint.apply(function (url) {
                if (!url) {
                    throw new error_js_1.VisibleError("Function URL is not enabled. Enable it with \"url: true\".");
                }
                return url;
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Function.prototype, "name", {
        /**
         * The name of the Lambda function.
         */
        get: function () {
            return this.function.name;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Function.prototype, "arn", {
        /**
         * The ARN of the Lambda function.
         */
        get: function () {
            return this.function.arn;
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Add environment variables lazily to the function after the function is created.
     *
     * This is useful for adding environment variables that are only available after the
     * function is created, like the function URL.
     *
     * @param environment The environment variables to add to the function.
     *
     * @example
     * Add the function URL as an environment variable.
     *
     * ```ts title="sst.config.ts"
     * const fn = new sst.aws.Function("MyFunction", {
     *   handler: "src/handler.handler",
     *   url: true,
     * });
     *
     * fn.addEnvironment({
     *   URL: fn.url,
     * });
     * ```
     */
    Function.prototype.addEnvironment = function (environment) {
        return new function_environment_update_js_1.FunctionEnvironmentUpdate("".concat(this.constructorName, "EnvironmentUpdate"), {
            functionName: this.name,
            environment: environment,
            region: (0, aws_1.getRegionOutput)(undefined, { parent: this }).name,
        }, { parent: this });
    };
    /** @internal */
    Function.fromDefinition = function (name, definition, override, argsTransform, opts) {
        return (0, pulumi_1.output)(definition).apply(function (definition) {
            if (typeof definition === "string") {
                return new (Function.bind.apply(Function, __spreadArray([void 0], (0, component_js_1.transform)(argsTransform, name, __assign({ handler: definition }, override), opts || {}), false)))();
            }
            else if (definition.handler) {
                return new (Function.bind.apply(Function, __spreadArray([void 0], (0, component_js_1.transform)(argsTransform, name, __assign(__assign(__assign({}, definition), override), { permissions: (0, pulumi_1.all)([
                        definition.permissions,
                        override === null || override === void 0 ? void 0 : override.permissions,
                    ]).apply(function (_a) {
                        var permissions = _a[0], overridePermissions = _a[1];
                        return __spreadArray(__spreadArray([], (permissions !== null && permissions !== void 0 ? permissions : []), true), (overridePermissions !== null && overridePermissions !== void 0 ? overridePermissions : []), true);
                    }) }), opts || {}), false)))();
            }
            throw new Error("Invalid function definition for the \"".concat(name, "\" Function"));
        });
    };
    /** @internal */
    Function.prototype.getSSTLink = function () {
        return {
            properties: {
                name: this.name,
                url: this.urlEndpoint,
            },
            include: [
                (0, permission_js_1.permission)({
                    actions: ["lambda:InvokeFunction"],
                    resources: [this.function.arn],
                }),
            ],
        };
    };
    Function.encryptionKey = (0, lazy_js_1.lazy)(function () {
        return new random_1.RandomBytes("LambdaEncryptionKey", {
            length: 32,
        });
    });
    Function.appsync = (0, lazy_js_1.lazy)(function () {
        return rpc_js_1.rpc.call("Provider.Aws.Appsync", {});
    });
    return Function;
}(component_js_1.Component));
exports.Function = Function;
var __pulumiType = "sst:aws:Function";
// @ts-expect-error
Function.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2, templateObject_3, templateObject_4, templateObject_5, templateObject_6, templateObject_7, templateObject_8, templateObject_9, templateObject_10, templateObject_11;
