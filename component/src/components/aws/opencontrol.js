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
exports.OpenControl = void 0;
var random_1 = require("@pulumi/random");
var component_1 = require("../component");
var function_builder_1 = require("./helpers/function-builder");
/**
 * The `OpenControl` component lets you deploy your
 * [OpenControl](https://opencontrol.ai) server to
 * [AWS Lambda](https://aws.amazon.com/lambda/).
 *
 * :::note
 * OpenControl is currently in beta.
 * :::
 *
 * @example
 *
 * #### Create an OpenControl server
 *
 * ```ts title="sst.config.ts"
 * const server = new sst.aws.OpenControl("MyServer", {
 *   server: "src/server.handler"
 * });
 * ```
 *
 * #### Link your AI API keys
 *
 * ```ts title="sst.config.ts" {6}
 * const anthropicKey = new sst.Secret("AnthropicKey");
 *
 * const server = new sst.aws.OpenControl("MyServer", {
 *   server: {
 *     handler: "src/server.handler",
 *     link: [anthropicKey]
 *   }
 * });
 * ```
 *
 * #### Link your resources
 *
 * If your tools are need access to specific resources, you can link them to the
 * OpenControl server.
 *
 * ```ts title="sst.config.ts" {6}
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.OpenControl("MyServer", {
 *   server: {
 *     handler: "src/server.handler",
 *     link: [bucket]
 *   }
 * });
 * ```
 *
 * #### Give AWS permissions
 *
 * If you are using the AWS tool within OpenControl, you will need to give
 * your OpenControl server permissions to access your AWS account.
 *
 * ```ts title="sst.config.ts" {4-6}
 * new sst.aws.OpenControl("OpenControl", {
 *   server: {
 *     handler: "src/server.handler",
 *     policies: $dev
 *       ? ["arn:aws:iam::aws:policy/AdministratorAccess"]
 *       : ["arn:aws:iam::aws:policy/ReadOnlyAccess"]
 *   }
 * });
 * ```
 *
 * Here we are giving it admin access in dev but read-only access in prod.
 *
 * #### Define your server
 *
 * Your `server` function might look like this.
 *
 * ```ts title="src/server.ts"
 * import { Resource } from "sst";
 * import { create } from "opencontrol";
 * import { tool } from "opencontrol/tool";
 * import { handle } from "hono/aws-lambda";
 * import { createAnthropic } from "@ai-sdk/anthropic";
 *
 * const myTool = tool({
 *   name: "my_tool",
 *   description: "Get the most popular greeting",
 *   async run() {
 *     return "Hello, world!";
 *   }
 * });
 *
 * const app = create({
 *   model: createAnthropic({
 *     apiKey: Resource.AnthropicKey.value,
 *   })("claude-3-7-sonnet-20250219"),
 *   tools: [myTool],
 * });
 *
 * export const handler = handle(app);
 * ```
 *
 * Learn more in the [OpenControl docs](https://opencontrol.ai) on how to configure
 * the `server` function.
 */
var OpenControl = /** @class */ (function (_super) {
    __extends(OpenControl, _super);
    function OpenControl(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var key = createKey();
        var server = createServer();
        _this._server = server;
        _this._key = key;
        registerOutputs();
        function registerOutputs() {
            self.registerOutputs({
                _hint: self.url,
            });
        }
        function createKey() {
            return new random_1.RandomPassword("".concat(name, "Key"), {
                length: 16,
                special: false,
            }, { parent: self }).result;
        }
        function createServer() {
            return (0, function_builder_1.functionBuilder)("".concat(name, "Server"), args.server, {
                link: [],
                environment: {
                    OPENCONTROL_KEY: key,
                },
                url: true,
                _skipHint: true,
            }, function (args) {
                args.url = {
                    cors: false,
                };
            }, { parent: self }).apply(function (v) { return v.getFunction(); });
        }
        return _this;
    }
    Object.defineProperty(OpenControl.prototype, "url", {
        /**
         * The URL of the OpenControl server.
         */
        get: function () {
            return this._server.url;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(OpenControl.prototype, "password", {
        /**
         * The password for the OpenControl server.
         */
        get: function () {
            return this._key;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(OpenControl.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Function component for the server.
                 */
                server: this._server,
            };
        },
        enumerable: false,
        configurable: true
    });
    return OpenControl;
}(component_1.Component));
exports.OpenControl = OpenControl;
var __pulumiType = "sst:aws:OpenControl";
// @ts-expect-error
OpenControl.__pulumiType = __pulumiType;
