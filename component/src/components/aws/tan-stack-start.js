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
exports.TanStackStart = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var error_js_1 = require("../error.js");
var ssr_site_js_1 = require("./ssr-site.js");
/**
 * The `TanStackStart` component lets you deploy a [TanStack Start](https://tanstack.com/start/latest) app to AWS.
 *
 * :::note
 * You need to make sure the `server.preset` value in the `app.config.ts` is set to `aws-lambda`.
 * :::
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy a TanStack Start app that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.TanStackStart("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the TanStack Start app in the `my-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.TanStackStart("MyWeb", {
 *   path: "my-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your TanStack Start app.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.TanStackStart("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.aws.TanStackStart("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your TanStack Start app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.TanStackStart("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your TanStack Start app.
 *
 * ```ts title="src/app.tsx"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 */
var TanStackStart = /** @class */ (function (_super) {
    __extends(TanStackStart, _super);
    function TanStackStart(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return _super.call(this, __pulumiType, name, args, opts) || this;
    }
    TanStackStart.prototype.normalizeBuildCommand = function () { };
    TanStackStart.prototype.buildPlan = function (outputPath) {
        return outputPath.apply(function (outputPath) {
            var _a, _b;
            var nitro = JSON.parse(fs_1.default.readFileSync(path_1.default.join(outputPath, ".output", "nitro.json"), "utf-8"));
            if (!["aws-lambda"].includes(nitro.preset)) {
                throw new error_js_1.VisibleError("TanStackStart's app.config.ts must be configured to use the \"aws-lambda\" preset. It is currently set to \"".concat(nitro.preset, "\"."));
            }
            var serverOutputPath = path_1.default.join(outputPath, ".output", "server");
            // If basepath is configured, nitro.mjs will have a line that looks like this:
            // return createRouter$2({ routeTree: Nr, defaultPreload: "intent", defaultErrorComponent: ce, defaultNotFoundComponent: () => jsx(de, {}), scrollRestoration: true, basepath: "/tan" });
            var basepath;
            // TanStack Start currently doesn't support basepaths.
            //try {
            //  const serverNitroChunk = fs.readFileSync(
            //    path.join(serverOutputPath, "chunks", "nitro", "nitro.mjs"),
            //    "utf-8",
            //  );
            //  basepath = serverNitroChunk.match(/basepath: "(.*)"/)?.[1];
            //} catch (e) {}
            // Remove the .output/public/_server directory from the assets
            // b/c all `_server` requests should go to the server function. If this folder is
            // not removed, it will create an s3 route that conflicts with the `_server` route.
            fs_1.default.rmSync(path_1.default.join(outputPath, ".output", "public", "_server"), {
                recursive: true,
                force: true,
            });
            fs_1.default.rmSync(path_1.default.join(outputPath, ".output", "public", "api"), {
                recursive: true,
                force: true,
            });
            return {
                base: basepath,
                server: {
                    description: "Server handler for TanStack",
                    handler: "index.handler",
                    bundle: serverOutputPath,
                    streaming: ((_b = (_a = nitro === null || nitro === void 0 ? void 0 : nitro.config) === null || _a === void 0 ? void 0 : _a.awsLambda) === null || _b === void 0 ? void 0 : _b.streaming) === true,
                },
                assets: [
                    {
                        from: path_1.default.join(".output", "public"),
                        to: "",
                        cached: true,
                    },
                ],
            };
        });
    };
    Object.defineProperty(TanStackStart.prototype, "url", {
        /**
         * The URL of the TanStack Start app.
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
    return TanStackStart;
}(ssr_site_js_1.SsrSite));
exports.TanStackStart = TanStackStart;
var __pulumiType = "sst:aws:TanstackStart";
// @ts-expect-error
TanStackStart.__pulumiType = __pulumiType;
