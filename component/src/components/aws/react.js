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
exports.React = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var pulumi_1 = require("@pulumi/pulumi");
var ssr_site_js_1 = require("./ssr-site.js");
/**
 * The `React` component lets you deploy a React app built with [React Router](https://reactrouter.com/) app to AWS.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy a React app that's in the project root.
 *
 * ```js
 * new sst.aws.React("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the React app in the `my-react-app/` directory.
 *
 * ```js {2}
 * new sst.aws.React("MyWeb", {
 *   path: "my-react-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your React app.
 *
 * ```js {2}
 * new sst.aws.React("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4}
 * new sst.aws.React("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your React app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4}
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.React("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your React app.
 *
 * ```ts title="app/root.tsx"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 */
var React = /** @class */ (function (_super) {
    __extends(React, _super);
    function React(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return _super.call(this, __pulumiType, name, args, opts) || this;
    }
    React.prototype.normalizeBuildCommand = function () { };
    React.prototype.buildPlan = function (outputPath) {
        return (0, pulumi_1.output)(outputPath).apply(function (outputPath) {
            var assetsPath = path_1.default.join("build", "client");
            var serverPath = (function () {
                var p = path_1.default.join("build", "server");
                return fs_1.default.existsSync(path_1.default.join(outputPath, p)) ? p : undefined;
            })();
            var indexPage = "index.html";
            // Get base configured in vite config ie. "/docs/"
            var viteBase = (function () {
                try {
                    var viteConfig = path_1.default.join(outputPath, "vite.config.ts");
                    var content = fs_1.default.readFileSync(viteConfig, "utf-8");
                    var match = content.match(/["']?base["']?:\s*["']([^"]+)["']/);
                    return match ? match[1] : undefined;
                }
                catch (e) { }
            })();
            // Get base configured in react-router config ie. "/docs/"
            var reactRouterBase = (function () {
                try {
                    var rrConfig = path_1.default.join(outputPath, "react-router.config.ts");
                    var content = fs_1.default.readFileSync(rrConfig, "utf-8");
                    var match = content.match(/["']?basename["']?:\s*["']([^"]+)["']/);
                    return match ? match[1] : undefined;
                }
                catch (e) { }
            })();
            if (viteBase) {
                if (!viteBase.endsWith("/"))
                    throw new Error("The \"base\" value in vite.config.ts must end with a trailing slash (\"/\"). This is required for correct asset path construction.");
                if (!reactRouterBase)
                    throw new Error("Found \"base\" configured in vite.config.ts but missing \"basename\" in react-router.config.ts. Both configurations are required.");
            }
            if (reactRouterBase) {
                if (reactRouterBase.endsWith("/"))
                    throw new Error("The \"basename\" value in react-router.config.ts must not end with a trailing slash (\"/\"). This ensures the root URL is accessible without a trailing slash.");
                if (!viteBase)
                    throw new Error("Found \"basename\" configured in react-router.config.ts but missing \"base\" in vite.config.ts. Both configurations are required.");
            }
            return {
                base: reactRouterBase,
                server: serverPath
                    ? (function () {
                        // React does perform their own internal ESBuild process, but it doesn't bundle
                        // 3rd party dependencies by default. In the interest of keeping deployments
                        // seamless for users we will create a server bundle with all dependencies included.
                        fs_1.default.copyFileSync(path_1.default.join($cli.paths.platform, "functions", "react-server", "server.mjs"), path_1.default.join(outputPath, "build", "server.mjs"));
                        return {
                            handler: path_1.default.join(outputPath, "build", "server.handler"),
                            streaming: true,
                        };
                    })()
                    : undefined,
                assets: [
                    {
                        from: assetsPath,
                        to: "",
                        cached: true,
                        versionedSubDir: "assets",
                    },
                ],
                custom404: serverPath ? undefined : "/".concat(indexPage),
            };
        });
    };
    Object.defineProperty(React.prototype, "url", {
        /**
         * The URL of the React app.
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
    return React;
}(ssr_site_js_1.SsrSite));
exports.React = React;
var __pulumiType = "sst:aws:React";
// @ts-expect-error
React.__pulumiType = __pulumiType;
