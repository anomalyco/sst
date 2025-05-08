"use strict";
/**
 * The AWS Permission Linkable helper is used to define the AWS permissions included with the
 * [`sst.Linkable`](/docs/component/linkable/) component.
 *
 * @example
 *
 * ```ts
 * sst.aws.permission({
 *   actions: ["lambda:InvokeFunction"],
 *   resources: ["*"]
 * })
 * ```
 *
 * @packageDocumentation
 */
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.permission = permission;
function permission(input) {
    return __assign({ type: "aws.permission" }, input);
}
