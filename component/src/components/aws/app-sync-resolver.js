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
exports.AppSyncResolver = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var error_1 = require("../error");
var aws_1 = require("@pulumi/aws");
/**
 * The `AppSyncResolver` component is internally used by the `AppSync` component to add
 * resolvers to [AWS AppSync](https://docs.aws.amazon.com/appsync/latest/devguide/what-is-appsync.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addResolver` method of the `AppSync` component.
 */
var AppSyncResolver = /** @class */ (function (_super) {
    __extends(AppSyncResolver, _super);
    function AppSyncResolver(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var kind = normalizeKind();
        var resolver = createResolver();
        _this.resolver = resolver;
        function normalizeKind() {
            var _a;
            return (0, pulumi_1.output)((_a = args.kind) !== null && _a !== void 0 ? _a : "unit").apply(function (kind) {
                if (kind === "unit" && args.functions)
                    throw new error_1.VisibleError("The `functions` property is not supported for `unit` resolvers.");
                if (kind === "pipeline" && args.dataSource)
                    throw new error_1.VisibleError("The `dataSource` property is not supported for `pipeline` resolvers.");
                return kind;
            });
        }
        function createResolver() {
            var _a;
            var _b;
            return new ((_a = aws_1.appsync.Resolver).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.resolver, "".concat(name, "Resolver"), {
                apiId: args.apiId,
                kind: kind.apply(function (kind) { return kind.toUpperCase(); }),
                type: args.type,
                field: args.field,
                dataSource: args.dataSource,
                requestTemplate: args.requestTemplate,
                responseTemplate: args.responseTemplate,
                code: args.code,
                runtime: args.code
                    ? {
                        name: "APPSYNC_JS",
                        runtimeVersion: "1.0.0",
                    }
                    : undefined,
                pipelineConfig: args.functions
                    ? { functions: args.functions }
                    : undefined,
            }, { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(AppSyncResolver.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon AppSync Resolver.
                 */
                resolver: this.resolver,
            };
        },
        enumerable: false,
        configurable: true
    });
    return AppSyncResolver;
}(component_1.Component));
exports.AppSyncResolver = AppSyncResolver;
var __pulumiType = "sst:aws:AppSyncResolver";
// @ts-expect-error
AppSyncResolver.__pulumiType = __pulumiType;
