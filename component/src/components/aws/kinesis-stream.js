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
exports.KinesisStream = void 0;
var aws = require("@pulumi/aws");
var pulumi_1 = require("@pulumi/pulumi");
var component_js_1 = require("../component.js");
var naming_js_1 = require("../naming.js");
var kinesis_stream_lambda_subscriber_js_1 = require("./kinesis-stream-lambda-subscriber.js");
var arn_js_1 = require("./helpers/arn.js");
var permission_js_1 = require("./permission.js");
var subscriber_js_1 = require("./helpers/subscriber.js");
/**
 * The `KinesisStream` component lets you add an [Amazon Kinesis Data Streams](https://docs.aws.amazon.com/streams/latest/dev/introduction.html) to your app.
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * const stream = new sst.aws.KinesisStream("MyStream");
 * ```
 *
 * #### Subscribe to a stream
 *
 * ```ts title="sst.config.ts"
 * stream.subscribe("MySubscriber", "src/subscriber.handler");
 * ```
 *
 * #### Link the stream to a resource
 *
 * You can link the stream to other resources, like a function or your Next.js app.
 *
 * ```ts {2} title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [stream]
 * });
 * ```
 *
 * Once linked, you can write to the stream from your function code.
 *
 * ```ts title="app/page.tsx" {1,7}
 * import { Resource } from "sst";
 * import { KinesisClient, PutRecordCommand } from "@aws-sdk/client-kinesis";
 *
 * const client = new KinesisClient();
 *
 * await client.send(new PutRecordCommand({
 *   StreamName: Resource.MyStream.name,
 *   Data: JSON.stringify({ foo: "bar" }),
 *   PartitionKey: "myKey",
 * }));
 * ```
 */
var KinesisStream = /** @class */ (function (_super) {
    __extends(KinesisStream, _super);
    function KinesisStream(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var stream = createStream();
        _this.stream = stream;
        _this.constructorName = name;
        _this.constructorOpts = opts;
        function createStream() {
            var _a;
            var _b;
            return new ((_a = aws.kinesis.Stream).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.stream, "".concat(name, "Stream"), {
                streamModeDetails: {
                    streamMode: "ON_DEMAND",
                },
            }, { parent: parent }), false)))();
        }
        return _this;
    }
    KinesisStream.prototype.subscribe = function (nameOrSubscriber, subscriberOrArgs, args) {
        var _this = this;
        return (0, subscriber_js_1.isFunctionSubscriber)(subscriberOrArgs).apply(function (v) {
            return v
                ? KinesisStream._subscribe(nameOrSubscriber, // name
                _this.constructorName, _this.nodes.stream.arn, subscriberOrArgs, // subscriber
                args, { provider: _this.constructorOpts.provider })
                : KinesisStream._subscribeV1(_this.constructorName, _this.nodes.stream.arn, nameOrSubscriber, // subscriber
                subscriberOrArgs, // args
                { provider: _this.constructorOpts.provider });
        });
    };
    KinesisStream.subscribe = function (nameOrStreamArn, streamArnOrSubscriber, subscriberOrArgs, args) {
        var _this = this;
        return (0, subscriber_js_1.isFunctionSubscriber)(subscriberOrArgs).apply(function (v) {
            return v
                ? (0, pulumi_1.output)(streamArnOrSubscriber).apply(function (streamArn) {
                    return _this._subscribe(nameOrStreamArn, // name
                    (0, naming_js_1.logicalName)((0, arn_js_1.parseKinesisStreamArn)(streamArn).streamName), streamArn, subscriberOrArgs, // subscriber
                    args);
                })
                : (0, pulumi_1.output)(nameOrStreamArn).apply(function (streamArn) {
                    return _this._subscribeV1((0, naming_js_1.logicalName)((0, arn_js_1.parseKinesisStreamArn)(streamArn).streamName), streamArn, streamArnOrSubscriber, // subscriber
                    subscriberOrArgs);
                });
        });
    };
    KinesisStream._subscribe = function (subscriberName, name, streamArn, subscriber, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return (0, pulumi_1.output)(args).apply(function (args) {
            return new kinesis_stream_lambda_subscriber_js_1.KinesisStreamLambdaSubscriber("".concat(name, "Subscriber").concat(subscriberName), __assign({ stream: { arn: streamArn }, subscriber: subscriber }, args), opts);
        });
    };
    KinesisStream._subscribeV1 = function (name, streamArn, subscriber, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return (0, pulumi_1.all)([streamArn, subscriber, args]).apply(function (_a) {
            var _b;
            var streamArn = _a[0], subscriber = _a[1], args = _a[2];
            var suffix = (0, naming_js_1.logicalName)((0, naming_js_1.hashStringToPrettyString)([
                streamArn,
                JSON.stringify((_b = args.filters) !== null && _b !== void 0 ? _b : {}),
                typeof subscriber === "string" ? subscriber : subscriber.handler,
            ].join(""), 6));
            return new kinesis_stream_lambda_subscriber_js_1.KinesisStreamLambdaSubscriber("".concat(name, "Subscriber").concat(suffix), __assign({ stream: { arn: streamArn }, subscriber: subscriber }, args), opts);
        });
    };
    Object.defineProperty(KinesisStream.prototype, "name", {
        get: function () {
            return this.stream.name;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KinesisStream.prototype, "arn", {
        get: function () {
            return this.stream.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(KinesisStream.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon Kinesis Data Stream.
                 */
                stream: this.stream,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    KinesisStream.prototype.getSSTLink = function () {
        return {
            properties: {
                name: this.stream.name,
            },
            include: [
                (0, permission_js_1.permission)({
                    actions: ["kinesis:*"],
                    resources: [this.nodes.stream.arn],
                }),
            ],
        };
    };
    return KinesisStream;
}(component_js_1.Component));
exports.KinesisStream = KinesisStream;
var __pulumiType = "sst:aws:KinesisStream";
// @ts-expect-error
KinesisStream.__pulumiType = __pulumiType;
