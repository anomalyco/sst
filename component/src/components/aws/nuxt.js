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
exports.Nuxt = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var ssr_site_js_1 = require("./ssr-site.js");
/**
 * The `Nuxt` component lets you deploy a [Nuxt](https://nuxt.com) app to AWS.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy a Nuxt app that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.Nuxt("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the Nuxt app in the `my-nuxt-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.Nuxt("MyWeb", {
 *   path: "my-nuxt-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your Nuxt app.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.Nuxt("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.aws.Nuxt("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your Nuxt app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.Nuxt("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your Nuxt app.
 *
 * ```ts title="server/api/index.ts"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 */
var Nuxt = /** @class */ (function (_super) {
    __extends(Nuxt, _super);
    function Nuxt(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return _super.call(this, __pulumiType, name, args, opts) || this;
    }
    Nuxt.prototype.normalizeBuildCommand = function () { };
    Nuxt.prototype.buildPlan = function (outputPath) {
        return outputPath.apply(function (outputPath) {
            var _a;
            var basepath = (_a = fs_1.default
                .readFileSync(path_1.default.join(outputPath, "nuxt.config.ts"), "utf-8")
                .match(/baseURL: ['"](.*)['"]/)) === null || _a === void 0 ? void 0 : _a[1];
            return {
                base: basepath,
                server: {
                    description: "Server handler for Nuxt",
                    handler: "index.handler",
                    bundle: path_1.default.join(outputPath, ".output", "server"),
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
    Object.defineProperty(Nuxt.prototype, "url", {
        /**
         * The URL of the Nuxt app.
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
    return Nuxt;
}(ssr_site_js_1.SsrSite));
exports.Nuxt = Nuxt;
var __pulumiType = "sst:aws:Nuxt";
// @ts-expect-error
Nuxt.__pulumiType = __pulumiType;
