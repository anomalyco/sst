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
exports.Analog = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var error_js_1 = require("../error.js");
var ssr_site_js_1 = require("./ssr-site.js");
/**
 * The `Analog` component lets you deploy a [Analog](https://analogjs.org) app to AWS.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy an Analog app that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.Analog("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the Analog app in the `my-analog-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.Analog("MyWeb", {
 *   path: "my-analog-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your Analog app.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.Analog("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.aws.Analog("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your Analog app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.Analog("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your Analog app.
 *
 * ```ts title="src/app/app.config.ts"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 */
var Analog = /** @class */ (function (_super) {
    __extends(Analog, _super);
    function Analog(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return _super.call(this, __pulumiType, name, args, opts) || this;
    }
    Analog.prototype.normalizeBuildCommand = function () { };
    Analog.prototype.buildPlan = function (outputPath) {
        return outputPath.apply(function (outputPath) {
            var _a;
            var nitro = JSON.parse(fs_1.default.readFileSync(path_1.default.join(outputPath, "dist", "analog", "nitro.json"), "utf-8"));
            if (!["aws-lambda"].includes(nitro.preset)) {
                throw new error_js_1.VisibleError("Analog's vite.config.ts must be configured to use the \"aws-lambda\" preset. It is currently set to \"".concat(nitro.preset, "\"."));
            }
            var basepath = (_a = fs_1.default
                .readFileSync(path_1.default.join(outputPath, "vite.config.ts"), "utf-8")
                .match(/base: ['"](.*)['"]/)) === null || _a === void 0 ? void 0 : _a[1];
            return {
                base: basepath,
                server: {
                    description: "Server handler for Analog",
                    handler: "index.handler",
                    bundle: path_1.default.join(outputPath, "dist", "analog", "server"),
                },
                assets: [
                    {
                        from: path_1.default.join("dist", "analog", "public"),
                        to: "",
                        cached: true,
                    },
                ],
            };
        });
    };
    Object.defineProperty(Analog.prototype, "url", {
        /**
         * The URL of the Analog app.
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
    return Analog;
}(ssr_site_js_1.SsrSite));
exports.Analog = Analog;
var __pulumiType = "sst:aws:Analog";
// @ts-expect-error
Analog.__pulumiType = __pulumiType;
