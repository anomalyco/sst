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
exports.SnsTopicQueueSubscriber = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var queue_1 = require("./queue");
/**
 * The `SnsTopicQueueSubscriber` component is internally used by the `SnsTopic` component
 * to add subscriptions to your [Amazon SNS Topic](https://docs.aws.amazon.com/sns/latest/dg/sns-create-topic.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribeQueue` method of the `SnsTopic` component.
 */
var SnsTopicQueueSubscriber = /** @class */ (function (_super) {
    __extends(SnsTopicQueueSubscriber, _super);
    function SnsTopicQueueSubscriber(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var topic = (0, pulumi_1.output)(args.topic);
        var queueArn = (0, pulumi_1.output)(args.queue).apply(function (queue) {
            return queue instanceof queue_1.Queue ? queue.arn : (0, pulumi_1.output)(queue);
        });
        var policy = createPolicy();
        var subscription = createSubscription();
        _this.policy = policy;
        _this.subscription = subscription;
        function createPolicy() {
            return queue_1.Queue.createPolicy("".concat(name, "Policy"), queueArn, {
                parent: args.disableParent ? undefined : self,
            });
        }
        function createSubscription() {
            var _a;
            var _b;
            return new ((_a = aws_1.sns.TopicSubscription).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.subscription, "".concat(name, "Subscription"), {
                topic: topic.arn,
                protocol: "sqs",
                endpoint: queueArn,
                filterPolicy: args.filter && (0, pulumi_1.jsonStringify)(args.filter),
            }, { parent: args.disableParent ? undefined : self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(SnsTopicQueueSubscriber.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The SQS Queue policy.
                 */
                policy: this.policy,
                /**
                 * The SNS Topic subscription.
                 */
                subscription: this.subscription,
            };
        },
        enumerable: false,
        configurable: true
    });
    return SnsTopicQueueSubscriber;
}(component_1.Component));
exports.SnsTopicQueueSubscriber = SnsTopicQueueSubscriber;
var __pulumiType = "sst:aws:SnsTopicQueueSubscriber";
// @ts-expect-error
SnsTopicQueueSubscriber.__pulumiType = __pulumiType;
