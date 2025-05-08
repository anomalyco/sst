"use strict";
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
exports.createRule = createRule;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
function createRule(name, eventBusName, args, parent) {
    var _a;
    var _b;
    return new ((_a = aws_1.cloudwatch.EventRule).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.rule, "".concat(name, "Rule"), {
        eventBusName: eventBusName,
        eventPattern: args.pattern
            ? (0, pulumi_1.output)(args.pattern).apply(function (pattern) {
                return JSON.stringify({
                    "detail-type": pattern.detailType,
                    source: pattern.source,
                    detail: pattern.detail,
                });
            })
            : JSON.stringify({
                source: [{ prefix: "" }],
            }),
    }, { parent: parent }), false)))();
}
