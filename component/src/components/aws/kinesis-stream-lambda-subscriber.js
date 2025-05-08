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
exports.KinesisStreamLambdaSubscriber = void 0;
var aws_1 = require("@pulumi/aws");
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var function_builder_1 = require("./helpers/function-builder");
var arn_1 = require("./helpers/arn");
/**
 * The `KinesisStreamLambdaSubscriber` component is internally used by the `KinesisStream` component to
 * add a consumer to [Amazon Kinesis Data Streams](https://docs.aws.amazon.com/streams/latest/dev/introduction.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribe` method of the `KinesisStream` component.
 */
var KinesisStreamLambdaSubscriber = /** @class */ (function (_super) {
    __extends(KinesisStreamLambdaSubscriber, _super);
    function KinesisStreamLambdaSubscriber(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var stream = (0, pulumi_1.output)(args.stream);
        var fn = createFunction();
        var eventSourceMapping = createEventSourceMapping();
        _this.fn = fn;
        _this.eventSourceMapping = eventSourceMapping;
        function createFunction() {
            return (0, pulumi_1.output)(args.subscriber).apply(function (subscriber) {
                return (0, function_builder_1.functionBuilder)("".concat(name, "Function"), subscriber, {
                    description: "Subscribed to ".concat(name),
                    permissions: [
                        {
                            actions: [
                                "kinesis:DescribeStream",
                                "kinesis:DescribeStreamSummary",
                                "kinesis:GetRecords",
                                "kinesis:GetShardIterator",
                                "kinesis:ListShards",
                                "kinesis:ListStreams",
                                "kinesis:SubscribeToShard",
                            ],
                            resources: [stream.arn],
                        },
                    ],
                }, undefined, { parent: self });
            });
        }
        function createEventSourceMapping() {
            var _a;
            var _b;
            return new ((_a = aws_1.lambda.EventSourceMapping).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.eventSourceMapping, "".concat(name, "EventSourceMapping"), {
                eventSourceArn: stream.arn,
                functionName: fn.arn.apply(function (arn) { return (0, arn_1.parseFunctionArn)(arn).functionName; }),
                startingPosition: "LATEST",
                filterCriteria: args.filters && {
                    filters: (0, pulumi_1.output)(args.filters).apply(function (filters) {
                        return filters.map(function (filter) { return ({
                            pattern: JSON.stringify(filter),
                        }); });
                    }),
                },
            }, { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(KinesisStreamLambdaSubscriber.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var self = this;
            return {
                /**
                 * The Lambda function that'll be notified.
                 */
                get function() {
                    return self.fn.apply(function (fn) { return fn.getFunction(); });
                },
                /**
                 * The Lambda event source mapping.
                 */
                eventSourceMapping: self.eventSourceMapping,
            };
        },
        enumerable: false,
        configurable: true
    });
    return KinesisStreamLambdaSubscriber;
}(component_1.Component));
exports.KinesisStreamLambdaSubscriber = KinesisStreamLambdaSubscriber;
var __pulumiType = "sst:aws:KinesisStreamLambdaSubscriber";
// @ts-expect-error
KinesisStreamLambdaSubscriber.__pulumiType = __pulumiType;
