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
exports.SvelteKit = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var ssr_site_js_1 = require("./ssr-site.js");
/**
 * The `SvelteKit` component lets you deploy a [SvelteKit](https://kit.svelte.dev/) app to AWS.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy a SvelteKit app that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.SvelteKit("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the SvelteKit app in the `my-svelte-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.SvelteKit("MyWeb", {
 *   path: "my-svelte-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your SvelteKit app.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.SvelteKit("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.aws.SvelteKit("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your SvelteKit app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.SvelteKit("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your SvelteKit app.
 *
 * ```ts title="src/routes/+page.server.ts"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 */
var SvelteKit = /** @class */ (function (_super) {
    __extends(SvelteKit, _super);
    function SvelteKit(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return _super.call(this, __pulumiType, name, args, opts) || this;
    }
    SvelteKit.prototype.normalizeBuildCommand = function () { };
    SvelteKit.prototype.buildPlan = function (outputPath) {
        return outputPath.apply(function (outputPath) {
            var _a, _b;
            var serverOutputPath = path_1.default.join(outputPath, ".svelte-kit", "svelte-kit-sst", "server");
            var basepath;
            try {
                var manifest = fs_1.default
                    .readFileSync(path_1.default.join(serverOutputPath, "manifest.js"))
                    .toString();
                var appDir = (_a = manifest.match(/appDir: "(.+?)"/)) === null || _a === void 0 ? void 0 : _a[1];
                var appPath = (_b = manifest.match(/appPath: "(.+?)"/)) === null || _b === void 0 ? void 0 : _b[1];
                if (appDir && appPath && appPath.endsWith(appDir)) {
                    basepath = appPath.substring(0, appPath.length - appDir.length);
                }
            }
            catch (e) { }
            return {
                base: basepath,
                server: {
                    handler: path_1.default.join(serverOutputPath, "lambda-handler", "index.handler"),
                    nodejs: {
                        esbuild: {
                            minify: process.env.SST_DEBUG ? false : true,
                            sourcemap: process.env.SST_DEBUG ? "inline" : false,
                            define: {
                                "process.env.SST_DEBUG": process.env.SST_DEBUG
                                    ? "true"
                                    : "false",
                            },
                        },
                    },
                    copyFiles: [
                        {
                            from: path_1.default.join(outputPath, ".svelte-kit", "svelte-kit-sst", "prerendered"),
                            to: "prerendered",
                        },
                    ],
                },
                assets: [
                    {
                        from: path_1.default.join(".svelte-kit", "svelte-kit-sst", "client"),
                        to: "",
                        cached: true,
                        versionedSubDir: "_app",
                    },
                    {
                        from: path_1.default.join(".svelte-kit", "svelte-kit-sst", "prerendered"),
                        to: "",
                        cached: false,
                    },
                ],
            };
        });
    };
    Object.defineProperty(SvelteKit.prototype, "url", {
        /**
         * The URL of the SvelteKit app.
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
    return SvelteKit;
}(ssr_site_js_1.SsrSite));
exports.SvelteKit = SvelteKit;
var __pulumiType = "sst:aws:SvelteKit";
// @ts-expect-error
SvelteKit.__pulumiType = __pulumiType;
