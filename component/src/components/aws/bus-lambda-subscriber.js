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
var __makeTemplateObject = (this && this.__makeTemplateObject) || function (cooked, raw) {
    if (Object.defineProperty) { Object.defineProperty(cooked, "raw", { value: raw }); } else { cooked.raw = raw; }
    return cooked;
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
exports.BusLambdaSubscriber = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var bus_base_subscriber_1 = require("./bus-base-subscriber");
var aws_1 = require("@pulumi/aws");
var function_builder_1 = require("./helpers/function-builder");
/**
 * The `BusLambdaSubscriber` component is internally used by the `Bus` component
 * to add subscriptions to [Amazon EventBridge Event Bus](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-event-bus.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribe` method of the `Bus` component.
 */
var BusLambdaSubscriber = /** @class */ (function (_super) {
    __extends(BusLambdaSubscriber, _super);
    function BusLambdaSubscriber(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var bus = (0, pulumi_1.output)(args.bus);
        var rule = (0, bus_base_subscriber_1.createRule)(name, bus.name, args, self);
        var fn = createFunction();
        var permission = createPermission();
        var target = createTarget();
        _this.fn = fn;
        _this.permission = permission;
        _this.rule = rule;
        _this.target = target;
        function createFunction() {
            return (0, function_builder_1.functionBuilder)("".concat(name, "Function"), args.subscriber, {
                description: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["Subscribed to ", ""], ["Subscribed to ", ""])), bus.name),
            }, undefined, { parent: self });
        }
        function createPermission() {
            return new aws_1.lambda.Permission("".concat(name, "Permission"), {
                action: "lambda:InvokeFunction",
                function: fn.arn,
                principal: "events.amazonaws.com",
                sourceArn: rule.arn,
            }, { parent: self });
        }
        function createTarget() {
            var _a;
            var _b;
            return new ((_a = aws_1.cloudwatch.EventTarget).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.target, "".concat(name, "Target"), {
                arn: fn.arn,
                rule: rule.name,
                eventBusName: bus.name,
            }, { parent: self, dependsOn: [permission] }), false)))();
        }
        return _this;
    }
    Object.defineProperty(BusLambdaSubscriber.prototype, "nodes", {
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
                 * The Lambda permission.
                 */
                permission: this.permission,
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
    return BusLambdaSubscriber;
}(component_1.Component));
exports.BusLambdaSubscriber = BusLambdaSubscriber;
var __pulumiType = "sst:aws:BusLambdaSubscriber";
// @ts-expect-error
BusLambdaSubscriber.__pulumiType = __pulumiType;
var templateObject_1;
