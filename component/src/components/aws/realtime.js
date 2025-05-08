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
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Realtime = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var function_1 = require("./function");
var naming_1 = require("../naming");
var realtime_lambda_subscriber_1 = require("./realtime-lambda-subscriber");
var aws_1 = require("@pulumi/aws");
/**
 * The `Realtime` component lets you publish and subscribe to messages in realtime.
 *
 * It offers a **topic-based** messaging network using [AWS IoT](https://docs.aws.amazon.com/iot/latest/developerguide/what-is-aws-iot.html). Letting you publish and subscribe to messages using
 * a WebSocket in the browser and your server.
 *
 * Also, provides an [SDK](#sdk) to authorize clients, grant permissions to subscribe, and
 * publish to topics.
 *
 * :::note
 * IoT is shared across all apps and stages in your AWS account. So you need to prefix the
 * topics by the app and stage name.
 * :::
 *
 * There is **only 1 IoT endpoint** per region per AWS account. Messages from all apps and
 * stages are published to the same IoT endpoint. Make sure to prefix the topics by the
 * app and stage name.
 *
 * @example
 *
 * #### Create a realtime endpoint
 *
 * ```ts title="sst.config.ts"
 * const server = new sst.aws.Realtime("MyServer", {
 *   authorizer: "src/authorizer.handler"
 * });
 * ```
 *
 * #### Authorize the client
 *
 * ```ts title="src/authorizer.ts" "realtime.authorizer"
 * import { Resource } from "sst/aws";
 * import { realtime } from "sst/aws/realtime";
 *
 * export const handler = realtime.authorizer(async (token) => {
 *   // Validate the token
 *
 *   // Return the topics to subscribe and publish
 *   return {
 *     subscribe: [`${Resource.App.name}/${Resource.App.stage}/chat/room1`],
 *     publish: [`${Resource.App.name}/${Resource.App.stage}/chat/room1`],
 *   };
 * });
 * ```
 *
 * #### Publish and receive messages in your frontend
 *
 * ```ts title="app/page.tsx"
 * import { Resource } from "sst/aws";
 *
 * const client = new mqtt.MqttClient();
 * // Configure with
 * // - Resource.Realtime.endpoint
 * // - Resource.Realtime.authorizer
 * const connection = client.new_connection(config);
 *
 * // Subscribe messages
 * connection.on("message", (topic, payload) => {
 *   // Handle the message
 * });
 *
 * // Publish messages
 * connection.publish(topic, payload, mqtt.QoS.AtLeastOnce);
 * ```
 *
 * #### Subscribe messages in your backend
 *
 * ```ts title="sst.config.ts"
 * server.subscribe("src/subscriber.handler", {
 *   filter: `${$app.name}/${$app.stage}/chat/room1`
 * });
 * ```
 *
 * #### Publish message from your backend
 *
 * ```ts title="src/lambda.ts"
 * import { IoTDataPlaneClient, PublishCommand } from "@aws-sdk/client-iot-data-plane";
 * const data = new IoTDataPlaneClient();
 * await data.send(
 *   new PublishCommand({
 *     payload: Buffer.from(
 *       JSON.stringify({ message: "Hello world" })
 *     ),
 *     topic: `${Resource.App.name}/${Resource.App.stage}/chat/room1`,
 *   })
 * );
 * ```
 */
var Realtime = /** @class */ (function (_super) {
    __extends(Realtime, _super);
    function Realtime(name, args, opts) {
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var authHadler = createAuthorizerFunction();
        var iotAuthorizer = createAuthorizer();
        createPermission();
        _this.constructorOpts = opts;
        _this.iotEndpoint = aws_1.iot.getEndpointOutput({
            endpointType: "iot:Data-ATS",
        }, { parent: parent }).endpointAddress;
        _this.constructorName = name;
        _this.authHadler = authHadler;
        _this.iotAuthorizer = iotAuthorizer;
        function createAuthorizerFunction() {
            return function_1.Function.fromDefinition("".concat(name, "AuthorizerHandler"), args.authorizer, {
                description: "Authorizer for ".concat(name),
                permissions: [
                    {
                        actions: ["iot:*"],
                        resources: ["*"],
                    },
                ],
            }, undefined, { parent: parent });
        }
        function createAuthorizer() {
            var _a;
            var _b;
            return new ((_a = aws_1.iot.Authorizer).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.authorizer, "".concat(name, "Authorizer"), {
                signingDisabled: true,
                authorizerFunctionArn: authHadler.arn,
            }, { parent: parent }), false)))();
        }
        function createPermission() {
            return new aws_1.lambda.Permission("".concat(name, "Permission"), {
                action: "lambda:InvokeFunction",
                function: authHadler.arn,
                principal: "iot.amazonaws.com",
                sourceArn: iotAuthorizer.arn,
            }, { parent: parent });
        }
        return _this;
    }
    Object.defineProperty(Realtime.prototype, "endpoint", {
        /**
         * The IoT endpoint.
         */
        get: function () {
            return this.iotEndpoint;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Realtime.prototype, "authorizer", {
        /**
         * The name of the IoT authorizer.
         */
        get: function () {
            return this.iotAuthorizer.name;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Realtime.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The IoT authorizer resource.
                 */
                authorizer: this.iotAuthorizer,
                /**
                 * The IoT authorizer function resource.
                 */
                authHandler: this.authHadler,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Subscribe to this Realtime server.
     *
     * @param subscriber The function that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * ```js title="sst.config.ts"
     * server.subscribe("src/subscriber.handler", {
     *   filter: `${$app.name}/${$app.stage}/chat/room1`
     * });
     * ```
     *
     * Customize the subscriber function.
     *
     * ```js title="sst.config.ts"
     * server.subscribe(
     *   {
     *     handler: "src/subscriber.handler",
     *     timeout: "60 seconds"
     *   },
     *   {
     *     filter: `${$app.name}/${$app.stage}/chat/room1`
     *   }
     * );
     * ```
     *
     * Or pass in the ARN of an existing Lambda function.
     *
     * ```js title="sst.config.ts"
     * server.subscribe("arn:aws:lambda:us-east-1:123456789012:function:my-function", {
     *   filter: `${$app.name}/${$app.stage}/chat/room1`
     * });
     * ```
     */
    Realtime.prototype.subscribe = function (subscriber, args) {
        var _this = this;
        return (0, pulumi_1.all)([subscriber, args.filter]).apply(function (_a) {
            var subscriber = _a[0], filter = _a[1];
            var suffix = (0, naming_1.logicalName)((0, naming_1.hashStringToPrettyString)([
                filter,
                typeof subscriber === "string" ? subscriber : subscriber.handler,
            ].join(""), 6));
            return new realtime_lambda_subscriber_1.RealtimeLambdaSubscriber("".concat(_this.constructorName, "Subscriber").concat(suffix), __assign({ iot: { name: _this.constructorName }, subscriber: subscriber }, args), { provider: _this.constructorOpts.provider });
        });
    };
    /** @internal */
    Realtime.prototype.getSSTLink = function () {
        return {
            properties: {
                endpoint: this.endpoint,
                authorizer: this.authorizer,
            },
        };
    };
    return Realtime;
}(component_1.Component));
exports.Realtime = Realtime;
var __pulumiType = "sst:aws:Realtime";
// @ts-expect-error
Realtime.__pulumiType = __pulumiType;
