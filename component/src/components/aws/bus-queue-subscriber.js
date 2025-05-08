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
exports.BusQueueSubscriber = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var bus_base_subscriber_1 = require("./bus-base-subscriber");
var aws_1 = require("@pulumi/aws");
var queue_1 = require("./queue");
/**
 * The `BusQueueSubscriber` component is internally used by the `Bus` component
 * to add subscriptions to [Amazon EventBridge Event Bus](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-event-bus.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribeQueue` method of the `Bus` component.
 */
var BusQueueSubscriber = /** @class */ (function (_super) {
    __extends(BusQueueSubscriber, _super);
    function BusQueueSubscriber(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var bus = (0, pulumi_1.output)(args.bus);
        var queueArn = (0, pulumi_1.output)(args.queue).apply(function (queue) {
            return queue instanceof queue_1.Queue ? queue.arn : (0, pulumi_1.output)(queue);
        });
        var policy = createPolicy();
        var rule = (0, bus_base_subscriber_1.createRule)(name, bus.name, args, self);
        var target = createTarget();
        _this.policy = policy;
        _this.rule = rule;
        _this.target = target;
        function createPolicy() {
            return queue_1.Queue.createPolicy("".concat(name, "Policy"), queueArn, { parent: self });
        }
        function createTarget() {
            var _a;
            var _b;
            return new ((_a = aws_1.cloudwatch.EventTarget).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.target, "".concat(name, "Target"), {
                arn: queueArn,
                rule: rule.name,
                eventBusName: bus.name,
            }, { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(BusQueueSubscriber.prototype, "nodes", {
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
                 * The EventBus rule.
                 */
                rule: this.rule,
                /**
                 * The EventBus target.
                 */
                target: this.target,
            };
        },
        enumerable: false,
        configurable: true
    });
    return BusQueueSubscriber;
}(component_1.Component));
exports.BusQueueSubscriber = BusQueueSubscriber;
var __pulumiType = "sst:aws:BusQueueSubscriber";
// @ts-expect-error
BusQueueSubscriber.__pulumiType = __pulumiType;
