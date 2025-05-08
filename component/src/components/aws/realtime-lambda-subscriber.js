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
exports.RealtimeLambdaSubscriber = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var aws_2 = require("@pulumi/aws");
var function_builder_1 = require("./helpers/function-builder");
var arn_1 = require("./helpers/arn");
/**
 * The `RealtimeLambdaSubscriber` component is internally used by the `Realtime` component
 * to add subscriptions to the [AWS IoT endpoint](https://docs.aws.amazon.com/iot/latest/developerguide/what-is-aws-iot.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribe` method of the `Realtime` component.
 */
var RealtimeLambdaSubscriber = /** @class */ (function (_super) {
    __extends(RealtimeLambdaSubscriber, _super);
    function RealtimeLambdaSubscriber(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var normalizedIot = (0, pulumi_1.output)(args.iot);
        var filter = (0, pulumi_1.output)(args.filter);
        var fn = createFunction();
        var rule = createRule();
        var permission = createPermission();
        _this.fn = fn;
        _this.permission = permission;
        _this.rule = rule;
        function createFunction() {
            return (0, function_builder_1.functionBuilder)("".concat(name, "Handler"), args.subscriber, {
                description: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["Subscribed to ", " on ", ""], ["Subscribed to ", " on ", ""])), normalizedIot.name, filter),
            }, undefined, { parent: self });
        }
        function createRule() {
            var _a;
            var _b;
            return new ((_a = aws_2.iot.TopicRule).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.topicRule, "".concat(name, "Rule"), {
                sqlVersion: "2016-03-23",
                sql: (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["SELECT * FROM '", "'"], ["SELECT * FROM '", "'"])), filter),
                enabled: true,
                lambdas: [{ functionArn: fn.arn }],
            }, { parent: self }), false)))();
        }
        function createPermission() {
            return new aws_1.lambda.Permission("".concat(name, "Permission"), {
                action: "lambda:InvokeFunction",
                function: fn.arn.apply(function (arn) { return (0, arn_1.parseFunctionArn)(arn).functionName; }),
                principal: "iot.amazonaws.com",
                sourceArn: rule.arn,
            }, { parent: self });
        }
        return _this;
    }
    Object.defineProperty(RealtimeLambdaSubscriber.prototype, "nodes", {
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
                 * The IoT Topic rule.
                 */
                rule: this.rule,
            };
        },
        enumerable: false,
        configurable: true
    });
    return RealtimeLambdaSubscriber;
}(component_1.Component));
exports.RealtimeLambdaSubscriber = RealtimeLambdaSubscriber;
var __pulumiType = "sst:aws:RealtimeLambdaSubscriber";
// @ts-expect-error
RealtimeLambdaSubscriber.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2;
