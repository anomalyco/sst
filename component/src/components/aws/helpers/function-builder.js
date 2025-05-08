"use strict";
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
exports.functionBuilder = functionBuilder;
var pulumi_1 = require("@pulumi/pulumi");
var function_1 = require("../function");
var component_1 = require("../../component");
var error_1 = require("../../error");
function functionBuilder(name, definition, defaultArgs, argsTransform, opts) {
    return (0, pulumi_1.output)(definition).apply(function (definition) {
        if (typeof definition === "string") {
            // Case 1: The definition is an ARN
            if (definition.startsWith("arn:")) {
                var parts = definition.split(":");
                return {
                    getFunction: function () {
                        throw new error_1.VisibleError("Cannot access the created function because it is referenced as an ARN.");
                    },
                    arn: (0, pulumi_1.output)(definition),
                    invokeArn: (0, pulumi_1.output)("arn:".concat(parts[1], ":apigateway:").concat(parts[3], ":lambda:path/2015-03-31/functions/").concat(definition, "/invocations")),
                };
            }
            // Case 2: The definition is a handler
            var fn_1 = new (function_1.Function.bind.apply(function_1.Function, __spreadArray([void 0], (0, component_1.transform)(argsTransform, name, __assign({ handler: definition }, defaultArgs), opts || {}), false)))();
            return {
                getFunction: function () { return fn_1; },
                arn: fn_1.arn,
                invokeArn: fn_1.nodes.function.invokeArn,
            };
        }
        // Case 3: The definition is a FunctionArgs
        else if (definition.handler) {
            var fn_2 = new (function_1.Function.bind.apply(function_1.Function, __spreadArray([void 0], (0, component_1.transform)(argsTransform, name, __assign(__assign(__assign({}, defaultArgs), definition), { link: (0, pulumi_1.all)([defaultArgs === null || defaultArgs === void 0 ? void 0 : defaultArgs.link, definition.link]).apply(function (_a) {
                    var defaultLink = _a[0], link = _a[1];
                    return __spreadArray(__spreadArray([], (defaultLink !== null && defaultLink !== void 0 ? defaultLink : []), true), (link !== null && link !== void 0 ? link : []), true);
                }), environment: (0, pulumi_1.all)([
                    defaultArgs === null || defaultArgs === void 0 ? void 0 : defaultArgs.environment,
                    definition.environment,
                ]).apply(function (_a) {
                    var defaultEnvironment = _a[0], environment = _a[1];
                    return (__assign(__assign({}, (defaultEnvironment !== null && defaultEnvironment !== void 0 ? defaultEnvironment : {})), (environment !== null && environment !== void 0 ? environment : {})));
                }), permissions: (0, pulumi_1.all)([
                    defaultArgs === null || defaultArgs === void 0 ? void 0 : defaultArgs.permissions,
                    definition.permissions,
                ]).apply(function (_a) {
                    var defaultPermissions = _a[0], permissions = _a[1];
                    return __spreadArray(__spreadArray([], (defaultPermissions !== null && defaultPermissions !== void 0 ? defaultPermissions : []), true), (permissions !== null && permissions !== void 0 ? permissions : []), true);
                }) }), opts || {}), false)))();
            return {
                getFunction: function () { return fn_2; },
                arn: fn_2.arn,
                invokeArn: fn_2.nodes.function.invokeArn,
            };
        }
        throw new Error("Invalid function definition for the \"".concat(name, "\" Function"));
    });
}
