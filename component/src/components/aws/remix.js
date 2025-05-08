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
Object.defineProperty(exports, "__esModule", { value: true });
exports.Remix = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var pulumi_1 = require("@pulumi/pulumi");
var error_js_1 = require("../error.js");
var ssr_site_js_1 = require("./ssr-site.js");
/**
 * The `Remix` component lets you deploy a [Remix](https://remix.run) app to AWS.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy a Remix app that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.Remix("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the Remix app in the `my-remix-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.Remix("MyWeb", {
 *   path: "my-remix-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your Remix app.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.Remix("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.aws.Remix("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your Remix app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.Remix("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your Remix app.
 *
 * ```ts title="app/root.tsx"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 */
var Remix = /** @class */ (function (_super) {
    __extends(Remix, _super);
    function Remix(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return _super.call(this, __pulumiType, name, args, opts) || this;
    }
    Remix.prototype.normalizeBuildCommand = function () { };
    Remix.prototype.buildPlan = function (outputPath, _name, args) {
        var _this = this;
        return (0, pulumi_1.all)([outputPath, args.buildDirectory]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
            function loadViteConfig() {
                return __awaiter(this, void 0, void 0, function () {
                    var file, vite, config, e_1;
                    return __generator(this, function (_a) {
                        switch (_a.label) {
                            case 0:
                                file = [
                                    "vite.config.ts",
                                    "vite.config.js",
                                    "vite.config.mts",
                                    "vite.config.mjs",
                                ].find(function (filename) { return fs_1.default.existsSync(path_1.default.join(outputPath, filename)); });
                                if (!file)
                                    return [2 /*return*/];
                                _a.label = 1;
                            case 1:
                                _a.trys.push([1, 4, , 5]);
                                return [4 /*yield*/, Promise.resolve().then(function () { return require("vite"); })];
                            case 2:
                                vite = _a.sent();
                                return [4 /*yield*/, vite.loadConfigFromFile({ command: "build", mode: "production" }, path_1.default.join(outputPath, file))];
                            case 3:
                                config = _a.sent();
                                if (!config)
                                    throw new Error();
                                return [2 /*return*/, {
                                        __remixPluginContext: {
                                            remixConfig: {
                                                buildDirectory: buildDirectory !== null && buildDirectory !== void 0 ? buildDirectory : "build",
                                            },
                                        },
                                    }];
                            case 4:
                                e_1 = _a.sent();
                                throw new error_js_1.VisibleError("Could not load Vite configuration from \"".concat(file, "\". Check that your Remix project uses Vite and the file exists."));
                            case 5: return [2 /*return*/];
                        }
                    });
                });
            }
            function createServerLambdaBundle() {
                // Create a Lambda@Edge handler for the Remix server bundle.
                //
                // Note: Remix does perform their own internal ESBuild process, but it
                // doesn't bundle 3rd party dependencies by default. In the interest of
                // keeping deployments seamless for users we will create a server bundle
                // with all dependencies included. We will still need to consider how to
                // address any need for external dependencies, although I think we should
                // possibly consider this at a later date.
                // In this path we are assuming that the Remix build only outputs the
                // "core server build". We can safely assume this as we have guarded the
                // remix.config.js to ensure it matches our expectations for the build
                // configuration.
                // We need to ensure that the "core server build" is wrapped with an
                // appropriate Lambda@Edge handler. We will utilise an internal asset
                // template to create this wrapper within the "core server build" output
                // directory.
                // Ensure build directory exists
                fs_1.default.mkdirSync(buildPath, { recursive: true });
                // Copy the server lambda handler and pre-append the build injection based
                // on the config file used.
                var content = [
                    // When using Vite config, the output build will be "server/index.js"
                    // and when using Remix config it will be `server.js`.
                    "// Import the server build that was produced by 'remix build'",
                    viteConfig
                        ? "import * as remixServerBuild from \"./server/index.js\";"
                        : "import * as remixServerBuild from \"./index.js\";",
                    "",
                    fs_1.default.readFileSync(path_1.default.join($cli.paths.platform, "functions", "remix-server", "regional-server.mjs")),
                ].join("\n");
                fs_1.default.writeFileSync(path_1.default.join(buildPath, "server.mjs"), content);
                // Copy the Remix polyfil to the server build directory
                //
                // Note: We need to ensure that the polyfills are injected above other code that
                // will depend on them when not using Vite. Importing them within the top of the
                // lambda code doesn't appear to guarantee this, we therefore leverage ESBUild's
                // `inject` option to ensure that the polyfills are injected at the top of
                // the bundle.
                var polyfillDest = path_1.default.join(buildPath, "polyfill.mjs");
                fs_1.default.copyFileSync(path_1.default.join($cli.paths.platform, "functions", "remix-server", "polyfill.mjs"), polyfillDest);
                return {
                    handler: path_1.default.join(buildPath, "server.handler"),
                    nodejs: {
                        esbuild: {
                            inject: [path_1.default.resolve(polyfillDest)],
                        },
                    },
                    streaming: true,
                };
            }
            var assetsPath, assetsVersionedSubDir, buildPath, viteConfig, basepath;
            var _c;
            var outputPath = _b[0], buildDirectory = _b[1];
            return __generator(this, function (_d) {
                switch (_d.label) {
                    case 0:
                        assetsPath = "public";
                        assetsVersionedSubDir = "build";
                        buildPath = path_1.default.join(outputPath, "build");
                        return [4 /*yield*/, loadViteConfig()];
                    case 1:
                        viteConfig = _d.sent();
                        if (viteConfig) {
                            assetsPath = path_1.default.join(viteConfig.__remixPluginContext.remixConfig.buildDirectory, "client");
                            assetsVersionedSubDir = "assets";
                            buildPath = path_1.default.join(outputPath, viteConfig.__remixPluginContext.remixConfig.buildDirectory);
                        }
                        basepath = (_c = fs_1.default
                            .readFileSync(path_1.default.join(outputPath, "vite.config.ts"), "utf-8")
                            .match(/base: ['"](.*)['"]/)) === null || _c === void 0 ? void 0 : _c[1];
                        return [2 /*return*/, {
                                base: basepath,
                                server: createServerLambdaBundle(),
                                assets: [
                                    {
                                        from: assetsPath,
                                        to: "",
                                        cached: true,
                                        versionedSubDir: assetsVersionedSubDir,
                                    },
                                ],
                            }];
                }
            });
        }); });
    };
    Object.defineProperty(Remix.prototype, "url", {
        /**
         * The URL of the Remix app.
         *
         * If the `domain` is set, this is the URL with the custom domain.
         * Otherwise, it's the auto-generated CloudFront URL.
         */
        get: function () {
            return _super.prototype.url;
        },
        enumerable: false,
        configurable: true
    });
    return Remix;
}(ssr_site_js_1.SsrSite));
exports.Remix = Remix;
var __pulumiType = "sst:aws:Remix";
// @ts-expect-error
Remix.__pulumiType = __pulumiType;
