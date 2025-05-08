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
exports.AppSyncFunction = void 0;
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
/**
 * The `AppSyncFunction` component is internally used by the `AppSync` component to add
 * functions to [AWS AppSync](https://docs.aws.amazon.com/appsync/latest/devguide/what-is-appsync.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addFunction` method of the `AppSync` component.
 */
var AppSyncFunction = /** @class */ (function (_super) {
    __extends(AppSyncFunction, _super);
    function AppSyncFunction(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var fn = createFunction();
        _this.fn = fn;
        function createFunction() {
            var _a;
            var _b;
            return new ((_a = aws_1.appsync.Function).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.function, "".concat(name, "Function"), {
                apiId: args.apiId,
                name: args.name,
                dataSource: args.dataSource,
                requestMappingTemplate: args.requestMappingTemplate,
                responseMappingTemplate: args.responseMappingTemplate,
                code: args.code,
                runtime: args.code
                    ? {
                        name: "APPSYNC_JS",
                        runtimeVersion: "1.0.0",
                    }
                    : undefined,
            }, { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(AppSyncFunction.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon AppSync Function.
                 */
                function: this.fn,
            };
        },
        enumerable: false,
        configurable: true
    });
    return AppSyncFunction;
}(component_1.Component));
exports.AppSyncFunction = AppSyncFunction;
var __pulumiType = "sst:aws:AppSyncFunction";
// @ts-expect-error
AppSyncFunction.__pulumiType = __pulumiType;
