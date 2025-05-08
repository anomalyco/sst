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
exports.Remix = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var pulumi_1 = require("@pulumi/pulumi");
var ssr_site_js_1 = require("./ssr-site.js");
var component_js_1 = require("../component.js");
var hint_js_1 = require("../hint.js");
var base_ssr_site_js_1 = require("../base/base-ssr-site.js");
var path_to_regex_js_1 = require("../../util/path-to-regex.js");
/**
 * The `Remix` component lets you deploy a [Remix](https://remix.run) app to AWS.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy a Remix app that's in the project root.
 *
 * ```js
 * new sst.aws.Remix("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the Remix app in the `my-remix-app/` directory.
 *
 * ```js {2}
 * new sst.aws.Remix("MyWeb", {
 *   path: "my-remix-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your Remix app.
 *
 * ```js {2}
 * new sst.aws.Remix("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4}
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
 * ```ts {4}
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
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var sitePath = (0, ssr_site_js_1.prepare)(args).sitePath;
        var isUsingVite = checkIsUsingVite();
        var storage = (0, ssr_site_js_1.createKvStorage)(parent, name, args);
        var outputPath = $dev ? sitePath : (0, base_ssr_site_js_1.buildApp)(parent, name, args, sitePath);
        var buildMeta = loadBuildOutput().buildMeta;
        var plan = buildPlan();
        var _a = (0, ssr_site_js_1.createRouter)(parent, name, args, outputPath, storage, plan), router = _a.router, server = _a.server;
        _this.assets = storage;
        _this.router = router;
        _this.server = server;
        if (!$dev) {
            hint_js_1.Hint.register(_this.urn, _this.url);
        }
        _this.registerOutputs({
            _metadata: {
                mode: $dev ? "placeholder" : "deployed",
                path: sitePath,
                url: _this.url,
            },
        });
        function checkIsUsingVite() {
            return sitePath.apply(function (sitePath) {
                return fs_1.default.existsSync(path_1.default.join(sitePath, "vite.config.ts")) ||
                    fs_1.default.existsSync(path_1.default.join(sitePath, "vite.config.js"));
            });
        }
        function loadBuildOutput() {
            return {
                buildMeta: $dev ? loadBuildMetadataPlaceholder() : loadBuildMetadata(),
            };
        }
        function loadBuildMetadata() {
            return (0, pulumi_1.all)([outputPath, isUsingVite]).apply(function (_a) {
                var outputPath = _a[0], isUsingVite = _a[1];
                // The path for all files that need to be in the "/" directory (static assets)
                // is different when using Vite. These will be located in the "build/client"
                // path of the output. It will be the "public" folder when using remix config.
                var assetsPath = isUsingVite
                    ? path_1.default.join("build", "client")
                    : "public";
                var assetsVersionedSubDir = isUsingVite ? undefined : "build";
                return {
                    assetsPath: assetsPath,
                    assetsVersionedSubDir: assetsVersionedSubDir,
                    // create 1 behaviour for each top level asset file/folder
                    staticRoutes: fs_1.default
                        .readdirSync(path_1.default.join(outputPath, assetsPath), {
                        withFileTypes: true,
                    })
                        .map(function (item) {
                        return item.isDirectory() ? "".concat(item.name, "/(.*)") : item.name;
                    }),
                };
            });
        }
        function loadBuildMetadataPlaceholder() {
            return {
                assetsPath: "placeholder",
                assetsVersionedSubDir: undefined,
                staticRoutes: [],
            };
        }
        function buildPlan() {
            return (0, pulumi_1.all)([isUsingVite, outputPath, buildMeta]).apply(function (_a) {
                var isUsingVite = _a[0], outputPath = _a[1], buildMeta = _a[2];
                return (0, ssr_site_js_1.validatePlan)({
                    server: createServerLambdaBundle(isUsingVite, outputPath),
                    assets: {
                        copy: [
                            {
                                from: buildMeta.assetsPath,
                                to: "",
                                cached: true,
                                versionedSubDir: buildMeta.assetsVersionedSubDir,
                            },
                        ],
                    },
                    routes: [
                        {
                            regex: (0, path_to_regex_js_1.pathToRegexp)(buildMeta.staticRoutes).source,
                            origin: "assets",
                        },
                        {
                            regex: (0, path_to_regex_js_1.pathToRegexp)("(.*)").source,
                            origin: "server",
                        },
                    ],
                });
            });
        }
        function createServerLambdaBundle(isUsingVite, outputPath) {
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
            var buildPath = path_1.default.join(outputPath, "build");
            fs_1.default.mkdirSync(buildPath, { recursive: true });
            // Copy the server lambda handler and pre-append the build injection based
            // on the config file used.
            var content = [
                // When using Vite config, the output build will be "server/index.js"
                // and when using Remix config it will be `server.js`.
                //isUsingVite
                //  ? `import * as remixServerBuild from "./server/index.js";`
                //  : `import * as remixServerBuild from "./index.js";`,
                //`import { createRequestHandler } from "@remix-run/cloudflare";`,
                //`import * as remixServerBuild from "./server";`,
                //`import { createRequestHandler } from "@remix-run/cloudflare";`,
                //`export default {`,
                //`  async fetch(request) {`,
                //`    const requestHandler = createRequestHandler(remixServerBuild);`,
                //`    return await requestHandler(request);`,
                //`  },`,
                //`};`,
                "import { createRequestHandler } from \"@remix-run/cloudflare\";",
                "import * as build from \"./server/index.js\";",
                "export default {",
                "  async fetch(request) {",
                "    console.log(\"fetch\");",
                "    console.log(\"build\", build);",
                "    console.log(\"build mode\", build.mode);",
                "    const handleRequest = createRequestHandler(build);",
                "    console.log(\"handleRequest\", handleRequest);",
                "    return await handleRequest(request);",
                "  },",
                "};",
            ].join("\n");
            fs_1.default.writeFileSync(path_1.default.join(buildPath, "server.ts"), content);
            var nodeBuiltInModulesPlugin = {
                name: "node:built-in:modules",
                setup: function (build) {
                    build.onResolve({ filter: /^(util|stream)$/ }, function (_a) {
                        var kind = _a.kind, path = _a.path;
                        // this plugin converts `require("node:*")` calls, those are the only ones that
                        // need updating (esm imports to "node:*" are totally valid), so here we tag with the
                        // node-buffer namespace only imports that are require calls
                        return kind === "require-call"
                            ? { path: path, namespace: "node-built-in-modules" }
                            : undefined;
                    });
                    // we convert the imports we tagged with the node-built-in-modules namespace so that instead of `require("node:*")`
                    // they import from `export * from "node:*";`
                    build.onLoad({ filter: /.*/, namespace: "node-built-in-modules" }, function (_a) {
                        var path = _a.path;
                        return {
                            contents: "export * from 'node:".concat(path, "'"),
                            loader: "js",
                        };
                    });
                },
            };
            return {
                handler: path_1.default.join(buildPath, "server.ts"),
                build: {
                    esbuild: {
                        define: {
                            process: JSON.stringify({
                                env: {
                                    //NODE_ENV: "production",
                                    NODE_ENV: "development",
                                },
                            }),
                        },
                        plugins: [nodeBuiltInModulesPlugin],
                    },
                },
            };
        }
        return _this;
    }
    Object.defineProperty(Remix.prototype, "url", {
        /**
         * The URL of the Remix app.
         *
         * If the `domain` is set, this is the URL with the custom domain.
         * Otherwise, it's the auto-generated CloudFront URL.
         */
        get: function () {
            return this.router.url;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Remix.prototype, "nodes", {
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
                assets: this.assets,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Remix.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
            },
        };
    };
    return Remix;
}(component_js_1.Component));
exports.Remix = Remix;
var __pulumiType = "sst:cloudflare:Remix";
// @ts-expect-error
Remix.__pulumiType = __pulumiType;
