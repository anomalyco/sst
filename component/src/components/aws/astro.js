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
exports.Astro = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var compare_semver_js_1 = require("../../util/compare-semver.js");
var error_js_1 = require("../error.js");
var ssr_site_js_1 = require("./ssr-site.js");
/**
 * The `Astro` component lets you deploy an [Astro](https://astro.build) site to AWS.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy the Astro site that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.Astro("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the Astro site in the `my-astro-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.Astro("MyWeb", {
 *   path: "my-astro-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your Astro site.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.Astro("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.aws.Astro("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your Astro site. This will grant permissions
 * to the resources and allow you to access it in your site.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.Astro("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your Astro site.
 *
 * ```astro title="src/pages/index.astro"
 * ---
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ---
 * ```
 */
var Astro = /** @class */ (function (_super) {
    __extends(Astro, _super);
    function Astro(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return _super.call(this, __pulumiType, name, args, opts) || this;
    }
    Astro.prototype.normalizeBuildCommand = function () { };
    Astro.prototype.buildPlan = function (outputPath) {
        return outputPath.apply(function (outputPath) {
            var BUILD_META_FILE_NAME = "sst.buildMeta.json";
            var filePath = path_1.default.join(outputPath, "dist", BUILD_META_FILE_NAME);
            if (!fs_1.default.existsSync(filePath)) {
                throw new error_js_1.VisibleError("Build metadata file not found at \"".concat(filePath, "\". Update your \"astro-sst\" adapter and rebuild your Astro site."));
            }
            var buildMeta = JSON.parse(fs_1.default.readFileSync(filePath, "utf-8"));
            var serverOutputPath = path_1.default.join(outputPath, "dist", "server");
            if (buildMeta.pluginVersion === undefined ||
                (0, compare_semver_js_1.isALtB)(buildMeta.pluginVersion, "3.1.2")) {
                throw new error_js_1.VisibleError("Incompatible \"astro-sst\" adapter version detected. The Astro component requires \"astro-sst\" adapter version 3.1.2 or later.");
            }
            // Note about handling 404 pages. Here is Astro's behavior:
            // - when static/prerendered, Astro builds a /404.html file in the client build output dir
            // - when SSR, Astro server handles /404 route
            //
            // We could handle the /404.html with CloudFront's custom error response feature, but that will not work when routing the Astro through the `Router` component. It does not make sense for `Router` to have a custom error response shared across all routes (ie. API). Each route's 404 behavior are different.
            //
            // So here is what we do when a request comes in for ie. /garbage:
            //
            // - Case 1: static (no server) => In CF function S3 look up will fail, and uri will rewrite to /404.html
            //   x that's why we set `plan.custom404` to `/404.html`
            //
            // - Case 2: prerendered (has server) => In CF function S3 look up will fail, and request will be sent to the server function. Server fails to serve /garbage, and cannot find the route. Server tries to serve /404, and cannot find the route. Server finally serves the 404.html file manually bundled into it.
            //   x that's why we configure `plan.server.copyFiles` include /404.html
            //
            // - Case 3: SSR (has server) => In CF function S3 look up will fail, and request is sent to the server function. Server fails to serve /garbage, and cannot find the route. Server tries to serve /404.
            var isStatic = buildMeta.outputMode === "static";
            var base = buildMeta.base === "/" ? undefined : buildMeta.base;
            return {
                base: base,
                server: isStatic
                    ? undefined
                    : {
                        handler: path_1.default.join(serverOutputPath, "entry.handler"),
                        nodejs: { install: ["sharp"] },
                        streaming: buildMeta.responseMode === "stream",
                        copyFiles: fs_1.default.existsSync(path_1.default.join(serverOutputPath, "404.html"))
                            ? [
                                {
                                    from: path_1.default.join(serverOutputPath, "404.html"),
                                    to: "404.html",
                                },
                            ]
                            : [],
                    },
                assets: [
                    {
                        from: buildMeta.clientBuildOutputDir,
                        to: "",
                        cached: true,
                        versionedSubDir: buildMeta.clientBuildVersionedSubDir,
                    },
                ],
                custom404: isStatic &&
                    fs_1.default.existsSync(path_1.default.join(outputPath, buildMeta.clientBuildOutputDir, "404.html"))
                    ? "/404.html"
                    : undefined,
            };
        });
    };
    Object.defineProperty(Astro.prototype, "url", {
        /**
         * The URL of the Astro site.
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
    return Astro;
}(ssr_site_js_1.SsrSite));
exports.Astro = Astro;
var __pulumiType = "sst:aws:Astro";
// @ts-expect-error
Astro.__pulumiType = __pulumiType;
