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
exports.Worker = void 0;
var promises_1 = require("fs/promises");
var path_1 = require("path");
var pulumi_1 = require("@pulumi/pulumi");
var cf = require("@pulumi/cloudflare");
var component_1 = require("../component");
var worker_url_js_1 = require("./providers/worker-url.js");
var link_js_1 = require("../link.js");
var zone_lookup_js_1 = require("./providers/zone-lookup.js");
var aws_1 = require("@pulumi/aws");
var binding_js_1 = require("./binding.js");
var account_id_js_1 = require("./account-id.js");
var rpc_js_1 = require("../rpc/rpc.js");
/**
 * The `Worker` component lets you create a Cloudflare Worker.
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "src/worker.handler"
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to the Worker. This will handle the credentials
 * and allow you to access it in your handler.
 *
 * ```ts {5} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "src/worker.handler",
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your handler.
 *
 * ```ts title="src/worker.ts" {3}
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 *
 * #### Enable URLs
 *
 * Enable worker URLs to invoke the worker over HTTP.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "src/worker.handler",
 *   url: true
 * });
 * ```
 *
 * #### Bundling
 *
 * Customize how SST uses [esbuild](https://esbuild.github.io/) to bundle your worker code with the `build` property.
 *
 * ```ts title="sst.config.ts" {3-5}
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "src/worker.handler",
 *   build: {
 *     install: ["pg"]
 *   }
 * });
 * ```
 */
var Worker = /** @class */ (function (_super) {
    __extends(Worker, _super);
    function Worker(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var dev = normalizeDev();
        var urlEnabled = normalizeUrl();
        var bindings = buildBindings();
        var iamCredentials = createAwsCredentials();
        var buildInput = (0, pulumi_1.all)([name, args.handler, args.build, dev]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
            var name = _b[0], handler = _b[1], build = _b[2];
            return __generator(this, function (_c) {
                return [2 /*return*/, {
                        functionID: name,
                        links: {},
                        handler: handler,
                        runtime: "worker",
                        properties: {
                            accountID: account_id_js_1.DEFAULT_ACCOUNT_ID,
                            build: build,
                        },
                    }];
            });
        }); });
        var build = buildHandler();
        var script = createScript();
        var workerUrl = createWorkersUrl();
        var workerDomain = createWorkersDomain();
        _this.script = script;
        _this.workerUrl = workerUrl;
        _this.workerDomain = workerDomain;
        (0, pulumi_1.all)([dev, buildInput, script.name]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
            var dev = _b[0], buildInput = _b[1], scriptName = _b[2];
            return __generator(this, function (_c) {
                switch (_c.label) {
                    case 0:
                        if (!dev)
                            return [2 /*return*/, undefined];
                        return [4 /*yield*/, rpc_js_1.rpc.call("Runtime.AddTarget", __assign(__assign({}, buildInput), { properties: __assign(__assign({}, buildInput.properties), { scriptName: scriptName }) }))];
                    case 1:
                        _c.sent();
                        return [2 /*return*/];
                }
            });
        }); });
        _this.registerOutputs({
            _live: (0, pulumi_1.all)([name, args.handler, args.build, dev]).apply(function (_a) {
                var name = _a[0], handler = _a[1], build = _a[2], dev = _a[3];
                if (!dev)
                    return undefined;
                return {
                    functionID: name,
                    links: [],
                    handler: handler,
                    runtime: "worker",
                    properties: {
                        accountID: account_id_js_1.DEFAULT_ACCOUNT_ID,
                        scriptName: script.name,
                        build: build,
                    },
                };
            }),
            _metadata: {
                handler: args.handler,
            },
        });
        function normalizeDev() {
            return (0, pulumi_1.output)(args.dev).apply(function (v) { return $dev && v !== false; });
        }
        function normalizeUrl() {
            return (0, pulumi_1.output)(args.url).apply(function (v) { return v !== null && v !== void 0 ? v : false; });
        }
        function buildBindings() {
            var result = {
                plainTextBindings: [
                    {
                        name: "SST_RESOURCE_App",
                        text: (0, pulumi_1.jsonStringify)({
                            name: $app.name,
                            stage: $app.stage,
                        }),
                    },
                ],
            };
            if (!args.link)
                return result;
            return (0, pulumi_1.output)(args.link).apply(function (links) {
                var _a;
                for (var _i = 0, links_1 = links; _i < links_1.length; _i++) {
                    var link = links_1[_i];
                    if (!link_js_1.Link.isLinkable(link))
                        continue;
                    var name_1 = (0, pulumi_1.output)(link.urn).apply(function (uri) { return uri.split("::").at(-1); });
                    var item = link.getSSTLink();
                    var b = (_a = item.include) === null || _a === void 0 ? void 0 : _a.find(function (i) { return i.type === "cloudflare.binding"; });
                    if (b) {
                        if (!result[b.binding])
                            result[b.binding] = [];
                        result[b.binding].push(__assign(__assign({}, (b.binding === "queueBindings"
                            ? {
                                binding: name_1,
                            }
                            : {
                                name: name_1,
                            })), b.properties));
                        continue;
                    }
                    if (!result.secretTextBindings)
                        result.secretTextBindings = [];
                    result.secretTextBindings.push({
                        name: name_1,
                        text: (0, pulumi_1.jsonStringify)(item.properties),
                    });
                }
                return result;
            });
        }
        function createAwsCredentials() {
            return (0, pulumi_1.output)(link_js_1.Link.getInclude("aws.permission", args.link)).apply(function (permissions) {
                if (permissions.length === 0)
                    return;
                var user = new aws_1.iam.User("".concat(name, "AwsUser"), { forceDestroy: true }, { parent: parent });
                new aws_1.iam.UserPolicy("".concat(name, "AwsPolicy"), {
                    user: user.name,
                    policy: (0, pulumi_1.jsonStringify)({
                        Statement: permissions.map(function (p) { return ({
                            Effect: (function () {
                                var _a;
                                var effect = (_a = p.effect) !== null && _a !== void 0 ? _a : "allow";
                                return effect.charAt(0).toUpperCase() + effect.slice(1);
                            })(),
                            Action: p.actions,
                            Resource: p.resources,
                        }); }),
                    }),
                }, { parent: parent });
                var keys = new aws_1.iam.AccessKey("".concat(name, "AwsCredentials"), { user: user.name }, { parent: parent });
                return keys;
            });
        }
        function buildHandler() {
            var _this = this;
            var buildResult = buildInput.apply(function (input) { return __awaiter(_this, void 0, void 0, function () {
                var result;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0: return [4 /*yield*/, rpc_js_1.rpc.call("Runtime.Build", input)];
                        case 1:
                            result = _a.sent();
                            if (result.errors.length > 0) {
                                throw new Error(result.errors.join("\n"));
                            }
                            return [2 /*return*/, result];
                    }
                });
            }); });
            return buildResult;
        }
        function createScript() {
            var _this = this;
            return (0, pulumi_1.all)([build, args.environment, iamCredentials, bindings]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
                var _c, _d, _e, _f, _g, _h;
                var _j, _k;
                var _l;
                var build = _b[0], environment = _b[1], iamCredentials = _b[2], bindings = _b[3];
                return __generator(this, function (_m) {
                    switch (_m.label) {
                        case 0:
                            _d = (_c = (_j = cf.WorkerScript).bind).apply;
                            _e = [_j];
                            _f = [[void 0]];
                            _g = component_1.transform;
                            _h = [(_l = args.transform) === null || _l === void 0 ? void 0 : _l.worker, "".concat(name, "Script")];
                            _k = { name: "", accountId: account_id_js_1.DEFAULT_ACCOUNT_ID };
                            return [4 /*yield*/, promises_1.default.readFile(path_1.default.join(build.out, build.handler))];
                        case 1: return [2 /*return*/, new (_d.apply(_c, _e.concat([__spreadArray.apply(void 0, _f.concat([_g.apply(void 0, _h.concat([__assign.apply(void 0, [__assign.apply(void 0, [(_k.content = (_m.sent()).toString(), _k.module = true, _k.compatibilityDate = "2024-09-23", _k.compatibilityFlags = ["nodejs_compat"], _k), bindings]), { plainTextBindings: __spreadArray(__spreadArray(__spreadArray([], (iamCredentials
                                                    ? [
                                                        {
                                                            name: "AWS_ACCESS_KEY_ID",
                                                            text: iamCredentials.id,
                                                        },
                                                    ]
                                                    : []), true), Object.entries(environment !== null && environment !== void 0 ? environment : {}).map(function (_a) {
                                                    var key = _a[0], value = _a[1];
                                                    return ({
                                                        name: key,
                                                        text: value,
                                                    });
                                                }), true), (bindings.plainTextBindings || []), true), secretTextBindings: __spreadArray(__spreadArray([], (iamCredentials
                                                    ? [
                                                        {
                                                            name: "AWS_SECRET_ACCESS_KEY",
                                                            text: iamCredentials.secret,
                                                        },
                                                    ]
                                                    : []), true), (bindings.secretTextBindings || []), true) }]), { parent: parent }])), false]))])))()];
                    }
                });
            }); });
        }
        function createWorkersUrl() {
            return new worker_url_js_1.WorkerUrl("".concat(name, "Url"), {
                accountId: account_id_js_1.DEFAULT_ACCOUNT_ID,
                scriptName: script.name,
                enabled: urlEnabled,
            }, { parent: parent });
        }
        function createWorkersDomain() {
            if (!args.domain)
                return;
            var zone = new zone_lookup_js_1.ZoneLookup("".concat(name, "ZoneLookup"), {
                accountId: account_id_js_1.DEFAULT_ACCOUNT_ID,
                domain: args.domain,
            }, { parent: parent });
            return new cf.WorkerDomain("".concat(name, "Domain"), {
                accountId: account_id_js_1.DEFAULT_ACCOUNT_ID,
                service: script.name,
                hostname: args.domain,
                zoneId: zone.id,
            }, { parent: parent });
        }
        return _this;
    }
    Object.defineProperty(Worker.prototype, "url", {
        /**
         * The Worker URL if `url` is enabled.
         */
        get: function () {
            return this.workerDomain
                ? (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["https://", ""], ["https://", ""])), this.workerDomain.hostname) : this.workerUrl.url.apply(function (url) { return (url ? "https://".concat(url) : url); });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Worker.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Cloudflare Worker script.
                 */
                worker: this.script,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * When you link a worker, say WorkerA, to another worker, WorkerB; it automatically creates
     * a service binding between the workers. It allows WorkerA to call WorkerB without going
     * through a publicly-accessible URL.
     *
     * @example
     * ```ts title="index.ts" {3}
     * import { Resource } from "sst";
     *
     * await Resource.WorkerB.fetch(request);
     * ```
     *
     * Read more about [binding Workers](https://developers.cloudflare.com/workers/runtime-apis/bindings/service-bindings/).
     *
     * @internal
     */
    Worker.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
            },
            include: [
                (0, binding_js_1.binding)({
                    type: "serviceBindings",
                    properties: {
                        service: this.script.id,
                    },
                }),
            ],
        };
    };
    return Worker;
}(component_1.Component));
exports.Worker = Worker;
var __pulumiType = "sst:cloudflare:Worker";
// @ts-expect-error
Worker.__pulumiType = __pulumiType;
var templateObject_1;
