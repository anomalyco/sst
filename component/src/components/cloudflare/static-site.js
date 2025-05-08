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
var kv_js_1 = require("./kv.js");
var component_js_1 = require("../component.js");
var glob_1 = require("glob");
var kv_data_js_1 = require("./providers/kv-data.js");
var worker_js_1 = require("./worker.js");
var base_site_js_1 = require("../base/base-site.js");
var base_static_site_js_1 = require("../base/base-static-site.js");
var account_id_js_1 = require("./account-id.js");
/**
 * The `StaticSite` component lets you deploy a static website to Cloudflare. It uses [Cloudflare KV storage](https://developers.cloudflare.com/kv/) to store your files and [Cloudflare Workers](https://developers.cloudflare.com/workers/) to serve them.
 *
 * It can also `build` your site by running your static site generator, like [Vite](https://vitejs.dev) and uploading the build output to Cloudflare KV.
 *
 * @example
 *
 * #### Minimal example
 *
 * Simply uploads the current directory as a static site.
 *
 * ```js
 * new sst.aws.StaticSite("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Change the `path` that should be uploaded.
 *
 * ```js
 * new sst.aws.StaticSite("MyWeb", {
 *   path: "path/to/site"
 * });
 * ```
 *
 * #### Deploy a Vite SPA
 *
 * Use [Vite](https://vitejs.dev) to deploy a React/Vue/Svelte/etc. SPA by specifying the `build` config.
 *
 * ```js
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
 * ```js
 * new sst.aws.StaticSite("MyWeb", {
 *   errorPage: "404.html",
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
 * ```js
 * new sst.aws.StaticSite("MyWeb", {
 *   errorPage: "404.html",
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
 * ```js
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
 * ```js {2}
 * new sst.aws.StaticSite("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4}
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
 * ```ts {5-7}
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
        var parent = _this;
        var _a = (0, base_static_site_js_1.prepare)(args), sitePath = _a.sitePath, environment = _a.environment, indexPage = _a.indexPage;
        var outputPath = $dev
            ? path_1.default.join($cli.paths.platform, "functions", "empty-site")
            : (0, base_static_site_js_1.buildApp)(parent, name, args.build, sitePath, environment);
        var storage = createKvStorage();
        var assetManifest = generateAssetManifest();
        var kvData = uploadAssets();
        var worker = createRouter();
        _this.assets = storage;
        _this.router = worker;
        _this.registerOutputs({
            _hint: _this.url,
            _dev: {
                environment: environment,
                command: "npm run dev",
                directory: sitePath,
                autostart: true,
            },
            _metadata: {
                path: sitePath,
                environment: environment,
                url: _this.url,
            },
        });
        function createKvStorage() {
            var _a;
            return new (kv_js_1.Kv.bind.apply(kv_js_1.Kv, __spreadArray([void 0], (0, component_js_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.assets, "".concat(name, "Assets"), {}, {
                parent: parent,
                retainOnDelete: false,
            }), false)))();
        }
        function generateAssetManifest() {
            var _this = this;
            return (0, pulumi_1.all)([outputPath, args.assets]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
                var fileOptions, manifest, filesProcessed, _loop_1, _i, _c, fileOption;
                var _this = this;
                var _d, _e;
                var outputPath = _b[0], assets = _b[1];
                return __generator(this, function (_f) {
                    switch (_f.label) {
                        case 0:
                            fileOptions = (_d = assets === null || assets === void 0 ? void 0 : assets.fileOptions) !== null && _d !== void 0 ? _d : [
                                {
                                    files: "**",
                                    cacheControl: "max-age=0,no-cache,no-store,must-revalidate",
                                },
                                {
                                    files: ["**/*.js", "**/*.css"],
                                    cacheControl: "max-age=31536000,public,immutable",
                                },
                            ];
                            manifest = [];
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
                                            filesProcessed.push.apply(filesProcessed, files);
                                            _h = (_g = manifest.push).apply;
                                            _j = [manifest];
                                            return [4 /*yield*/, Promise.all(files.map(function (file) { return __awaiter(_this, void 0, void 0, function () {
                                                    var source, content, hash;
                                                    var _a;
                                                    return __generator(this, function (_b) {
                                                        switch (_b.label) {
                                                            case 0:
                                                                source = path_1.default.resolve(outputPath, file);
                                                                return [4 /*yield*/, fs_1.default.promises.readFile(source, 'utf-8')];
                                                            case 1:
                                                                content = _b.sent();
                                                                hash = crypto_1.default
                                                                    .createHash("sha256")
                                                                    .update(content)
                                                                    .digest("hex");
                                                                return [2 /*return*/, {
                                                                        source: source,
                                                                        key: file,
                                                                        hash: hash,
                                                                        cacheControl: fileOption.cacheControl,
                                                                        contentType: (_a = fileOption.contentType) !== null && _a !== void 0 ? _a : (0, base_site_js_1.getContentType)(file, "UTF-8"),
                                                                    }];
                                                        }
                                                    });
                                                }); }))];
                                        case 1:
                                            _h.apply(_g, _j.concat([(_k.sent())]));
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
                        case 4: return [2 /*return*/, manifest];
                    }
                });
            }); });
        }
        function uploadAssets() {
            return new kv_data_js_1.KvData("".concat(name, "AssetFiles"), {
                accountId: account_id_js_1.DEFAULT_ACCOUNT_ID,
                namespaceId: storage.id,
                entries: assetManifest.apply(function (manifest) {
                    return manifest.map(function (m) { return ({
                        source: m.source,
                        key: m.key,
                        hash: m.hash,
                        cacheControl: m.cacheControl,
                        contentType: m.contentType,
                    }); });
                }),
            }, { parent: parent, ignoreChanges: $dev ? ["*"] : undefined });
        }
        function createRouter() {
            return new worker_js_1.Worker("".concat(name, "Router"), {
                handler: path_1.default.join($cli.paths.platform, "functions", "cf-static-site-router-worker"),
                url: true,
                domain: args.domain,
                environment: __assign({ INDEX_PAGE: indexPage }, (args.errorPage ? { ERROR_PAGE: args.errorPage } : {})),
                build: {
                    esbuild: assetManifest.apply(function (assetManifest) { return ({
                        define: {
                            SST_ASSET_MANIFEST: JSON.stringify(Object.fromEntries(assetManifest.map(function (e) { return [e.key, e.hash]; }))),
                        },
                    }); }),
                },
                transform: {
                    worker: function (workerArgs) {
                        workerArgs.kvNamespaceBindings = [
                            {
                                name: "ASSETS",
                                namespaceId: storage.id,
                            },
                        ];
                    },
                },
            }, 
            // create worker after KV upload finishes
            { dependsOn: kvData, parent: parent });
        }
        return _this;
    }
    Object.defineProperty(StaticSite.prototype, "url", {
        /**
         * The URL of the website.
         *
         * If the `domain` is set, this is the URL with the custom domain.
         * Otherwise, it's the auto-generated worker URL.
         */
        get: function () {
            return this.router.url;
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
                 * The KV namespace that stores the assets.
                 */
                assets: this.assets,
                /**
                 * The worker that serves the requests.
                 */
                router: this.router,
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
var __pulumiType = "sst:cloudflare:StaticSite";
// @ts-expect-error
StaticSite.__pulumiType = __pulumiType;
