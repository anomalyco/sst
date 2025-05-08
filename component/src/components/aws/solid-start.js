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
exports.SolidStart = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var error_js_1 = require("../error.js");
var ssr_site_js_1 = require("./ssr-site.js");
/**
 * The `SolidStart` component lets you deploy a [SolidStart](https://start.solidjs.com) app to AWS.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy a SolidStart app that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.SolidStart("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the SolidStart app in the `my-solid-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.SolidStart("MyWeb", {
 *   path: "my-solid-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your SolidStart app.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.SolidStart("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.aws.SolidStart("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your SolidStart app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.SolidStart("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your SolidStart app.
 *
 * ```ts title="src/app.tsx"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 */
var SolidStart = /** @class */ (function (_super) {
    __extends(SolidStart, _super);
    function SolidStart(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return _super.call(this, __pulumiType, name, args, opts) || this;
    }
    SolidStart.prototype.normalizeBuildCommand = function () { };
    SolidStart.prototype.buildPlan = function (outputPath) {
        return outputPath.apply(function (outputPath) {
            var _a, _b, _c;
            // Make sure aws-lambda preset is used in nitro.json
            var nitro = JSON.parse(fs_1.default.readFileSync(path_1.default.join(outputPath, ".output", "nitro.json"), "utf-8"));
            if (!["aws-lambda"].includes(nitro.preset)) {
                throw new error_js_1.VisibleError("SolidStart's app.config.ts must be configured to use the \"aws-lambda\" preset. It is currently set to \"".concat(nitro.preset, "\"."));
            }
            // Get base path
            var appConfig = fs_1.default.readFileSync(path_1.default.join(outputPath, "app.config.ts"), "utf-8");
            var basepath = (_a = appConfig.match(/baseURL: ['"](.*)['"]/)) === null || _a === void 0 ? void 0 : _a[1];
            return {
                base: basepath,
                server: {
                    description: "Server handler for Solid",
                    handler: "index.handler",
                    bundle: path_1.default.join(outputPath, ".output", "server"),
                    streaming: ((_c = (_b = nitro === null || nitro === void 0 ? void 0 : nitro.config) === null || _b === void 0 ? void 0 : _b.awsLambda) === null || _c === void 0 ? void 0 : _c.streaming) === true,
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
    Object.defineProperty(SolidStart.prototype, "url", {
        /**
         * The URL of the SolidStart app.
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
    return SolidStart;
}(ssr_site_js_1.SsrSite));
exports.SolidStart = SolidStart;
var __pulumiType = "sst:aws:SolidStart";
// @ts-expect-error
SolidStart.__pulumiType = __pulumiType;
