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
exports.Auth = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var _1 = require(".");
var function_builder_1 = require("./helpers/function-builder");
var linkable_1 = require("../linkable");
var auth_v1_1 = require("./auth-v1");
/**
 * The `Auth` component lets you create centralized auth servers on AWS. It deploys
 * [OpenAuth](https://openauth.js.org) to [AWS Lambda](https://aws.amazon.com/lambda/)
 * and uses [Amazon DynamoDB](https://aws.amazon.com/dynamodb/) for storage.
 *
 * :::note
 * `Auth` and OpenAuth are currently in beta.
 * :::
 *
 * @example
 *
 * #### Create an OpenAuth server
 *
 * ```ts title="sst.config.ts"
 * const auth = new sst.aws.Auth("MyAuth", {
 *   issuer: "src/auth.handler"
 * });
 * ```
 *
 * Where the `issuer` function might look like this.
 *
 * ```ts title="src/auth.ts"
 * import { handle } from "hono/aws-lambda";
 * import { issuer } from "@openauthjs/openauth";
 * import { CodeProvider } from "@openauthjs/openauth/provider/code";
 * import { subjects } from "./subjects";
 *
 * const app = issuer({
 *   subjects,
 *   providers: {
 *     code: CodeProvider()
 *   },
 *   success: async (ctx, value) => {}
 * });
 *
 * export const handler = handle(app);
 * ```
 *
 * This `Auth` component will always use the
 * [`DynamoStorage`](https://openauth.js.org/docs/storage/dynamo/) storage provider.
 *
 * Learn more on the [OpenAuth docs](https://openauth.js.org/docs/issuer/) on how to configure
 * the `issuer` function.
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your auth server.
 *
 * ```js {3} title="sst.config.ts"
 * new sst.aws.Auth("MyAuth", {
 *   issuer: "src/auth.handler",
 *   domain: "auth.example.com"
 * });
 * ```
 *
 * #### Link to a resource
 *
 * You can link the auth server to other resources, like a function or your Next.js app,
 * that needs authentication.
 *
 * ```ts title="sst.config.ts" {2}
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [auth]
 * });
 * ```
 *
 * Once linked, you can now use it to create an [OpenAuth
 * client](https://openauth.js.org/docs/client/).
 *
 * ```ts title="app/page.tsx" {1,6}
 * import { Resource } from "sst"
 * import { createClient } from "@openauthjs/openauth/client"
 *
 * export const client = createClient({
 *   clientID: "nextjs",
 *   issuer: Resource.MyAuth.url
 * });
 * ```
 */
var Auth = /** @class */ (function (_super) {
    __extends(Auth, _super);
    function Auth(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var _version = 2;
        var self = _this;
        self.registerVersion({
            new: _version,
            old: $cli.state.version[name],
            message: [
                "There is a new version of \"Auth\" that has breaking changes.",
                "",
                "What changed:",
                "  - The latest version is now powered by OpenAuth - https://openauth.js.org",
                "",
                "To upgrade:",
                "  - Set `forceUpgrade: \"v".concat(_version, "\"` on the \"Auth\" component. Learn more https://sst.dev/docs/component/aws/auth#forceupgrade"),
                "",
                "To continue using v".concat($cli.state.version[name], ":"),
                "  - Rename \"Auth\" to \"Auth.v".concat($cli.state.version[name], "\". Learn more about versioning - https://sst.dev/docs/components/#versioning"),
            ].join("\n"),
            forceUpgrade: args.forceUpgrade,
        });
        var table = createTable();
        var issuer = createIssuer();
        var router = createRouter();
        _this._table = table;
        _this._issuer = issuer;
        _this._router = router;
        registerOutputs();
        function registerOutputs() {
            self.registerOutputs({
                _hint: self.url,
            });
        }
        function createTable() {
            return new _1.Dynamo("".concat(name, "Storage"), {
                fields: { pk: "string", sk: "string" },
                primaryIndex: { hashKey: "pk", rangeKey: "sk" },
                ttl: "expiry",
            }, { parent: self });
        }
        function createIssuer() {
            var fn = args.authorizer || args.issuer;
            if (!fn)
                throw new Error("Auth: issuer field must be set");
            return (0, function_builder_1.functionBuilder)("".concat(name, "Issuer"), fn, {
                link: [table],
                environment: {
                    OPENAUTH_STORAGE: (0, pulumi_1.jsonStringify)({
                        type: "dynamo",
                        options: { table: table.name },
                    }),
                },
                _skipHint: true,
            }, function (args) {
                args.url = {
                    cors: false,
                };
            }, { parent: self }).apply(function (v) { return v.getFunction(); });
        }
        function createRouter() {
            if (!args.domain)
                return;
            var router = new _1.Router("".concat(name, "Router"), {
                domain: args.domain,
                _skipHint: true,
            }, { parent: self });
            router.route("/", issuer.url);
            return router;
        }
        return _this;
    }
    Object.defineProperty(Auth.prototype, "url", {
        /**
         * The URL of the Auth component.
         *
         * If the `domain` is set, this is the URL with the custom domain.
         * Otherwise, it's the auto-generated function URL for the issuer.
         */
        get: function () {
            var _a, _b;
            return (_b = (_a = this._router) === null || _a === void 0 ? void 0 : _a.url) !== null && _b !== void 0 ? _b : this._issuer.url.apply(function (v) { return v.slice(0, -1); });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Auth.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The DynamoDB component.
                 */
                table: this._table,
                /**
                 * The Function component for the issuer.
                 */
                issuer: this._issuer,
                /**
                 * @deprecated Use `issuer` instead.
                 * The Function component for the issuer.
                 */
                authorizer: this._issuer,
                /**
                 * The Router component for the custom domain.
                 */
                router: this._router,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Auth.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
            },
            include: [
                (0, linkable_1.env)({
                    OPENAUTH_ISSUER: this.url,
                }),
            ],
        };
    };
    Auth.v1 = auth_v1_1.Auth;
    return Auth;
}(component_1.Component));
exports.Auth = Auth;
var __pulumiType = "sst:aws:Auth";
// @ts-expect-error
Auth.__pulumiType = __pulumiType;
