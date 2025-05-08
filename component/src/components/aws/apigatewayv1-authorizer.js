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
exports.ApiGatewayV1Authorizer = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var error_1 = require("../error");
var aws_1 = require("@pulumi/aws");
var function_builder_1 = require("./helpers/function-builder");
/**
 * The `ApiGatewayV1Authorizer` component is internally used by the `ApiGatewayV1` component
 * to add authorizers to [Amazon API Gateway REST API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addAuthorizer` method of the `ApiGatewayV1` component.
 */
var ApiGatewayV1Authorizer = /** @class */ (function (_super) {
    __extends(ApiGatewayV1Authorizer, _super);
    function ApiGatewayV1Authorizer(name, args, opts) {
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var api = (0, pulumi_1.output)(args.api);
        validateSingleAuthorizer();
        var type = getType();
        var fn = createFunction();
        var authorizer = createAuthorizer();
        createPermission();
        _this.fn = fn;
        _this.authorizer = authorizer;
        function validateSingleAuthorizer() {
            var authorizers = [
                args.requestFunction,
                args.tokenFunction,
                args.userPools,
            ].filter(function (e) { return e; });
            if (authorizers.length === 0)
                throw new error_1.VisibleError("Please provide one of \"requestFunction\", \"tokenFunction\", or \"userPools\" for the ".concat(args.name, " authorizer."));
            if (authorizers.length > 1) {
                throw new error_1.VisibleError("Please provide only one of \"requestFunction\", \"tokenFunction\", or \"userPools\" for the ".concat(args.name, " authorizer."));
            }
        }
        function getType() {
            if (args.tokenFunction)
                return "TOKEN";
            if (args.requestFunction)
                return "REQUEST";
            if (args.userPools)
                return "COGNITO_USER_POOLS";
        }
        function createFunction() {
            var _a;
            var fn = (_a = args.tokenFunction) !== null && _a !== void 0 ? _a : args.requestFunction;
            if (!fn)
                return;
            return (0, function_builder_1.functionBuilder)("".concat(name, "Handler"), fn, {
                description: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["", " authorizer"], ["", " authorizer"])), api.name),
            }, undefined, { parent: self });
        }
        function createPermission() {
            if (!fn)
                return;
            return new aws_1.lambda.Permission("".concat(name, "Permission"), {
                action: "lambda:InvokeFunction",
                function: fn.arn,
                principal: "apigateway.amazonaws.com",
                sourceArn: (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["", "/authorizers/", ""], ["", "/authorizers/", ""])), api.executionArn, authorizer.id),
            }, { parent: self });
        }
        function createAuthorizer() {
            var _a;
            var _b;
            return new ((_a = aws_1.apigateway.Authorizer).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.authorizer, "".concat(name, "Authorizer"), {
                restApi: api.id,
                type: type,
                name: args.name,
                providerArns: args.userPools,
                authorizerUri: fn === null || fn === void 0 ? void 0 : fn.invokeArn,
                authorizerResultTtlInSeconds: args.ttl,
                identitySource: args.identitySource,
            }, { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(ApiGatewayV1Authorizer.prototype, "id", {
        /**
         * The ID of the authorizer.
         */
        get: function () {
            return this.authorizer.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(ApiGatewayV1Authorizer.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var self = this;
            return {
                /**
                 * The API Gateway Authorizer.
                 */
                authorizer: this.authorizer,
                /**
                 * The Lambda function used by the authorizer.
                 */
                get function() {
                    if (!self.fn)
                        throw new error_1.VisibleError("Cannot access `nodes.function` because the data source does not use a Lambda function.");
                    return self.fn.apply(function (fn) { return fn.getFunction(); });
                },
            };
        },
        enumerable: false,
        configurable: true
    });
    return ApiGatewayV1Authorizer;
}(component_1.Component));
exports.ApiGatewayV1Authorizer = ApiGatewayV1Authorizer;
var __pulumiType = "sst:aws:ApiGatewayV1Authorizer";
// @ts-expect-error
ApiGatewayV1Authorizer.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2;
