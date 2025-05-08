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
exports.ApiGatewayV1LambdaRoute = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var apigatewayv1_base_route_1 = require("./apigatewayv1-base-route");
var function_builder_1 = require("./helpers/function-builder");
/**
 * The `ApiGatewayV1LambdaRoute` component is internally used by the `ApiGatewayV1` component
 * to add routes to your [API Gateway REST API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `route` method of the `ApiGatewayV1` component.
 */
var ApiGatewayV1LambdaRoute = /** @class */ (function (_super) {
    __extends(ApiGatewayV1LambdaRoute, _super);
    function ApiGatewayV1LambdaRoute(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var api = (0, pulumi_1.output)(args.api);
        var method = (0, apigatewayv1_base_route_1.createMethod)(name, args, self);
        var fn = createFunction();
        var permission = createPermission();
        var integration = createIntegration();
        _this.fn = fn;
        _this.permission = permission;
        _this.method = method;
        _this.integration = integration;
        function createFunction() {
            var method = args.method, path = args.path;
            return (0, function_builder_1.functionBuilder)("".concat(name, "Handler"), args.handler, {
                description: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["", " route ", " ", ""], ["", " route ", " ", ""])), api.name, method, path),
            }, args.handlerTransform, { parent: self });
        }
        function createPermission() {
            return new aws_1.lambda.Permission("".concat(name, "Permissions"), {
                action: "lambda:InvokeFunction",
                function: fn.arn,
                principal: "apigateway.amazonaws.com",
                sourceArn: (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["", "/*"], ["", "/*"])), api.executionArn),
            }, { parent: self });
        }
        function createIntegration() {
            var _a;
            var _b;
            return new ((_a = aws_1.apigateway.Integration).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.integration, "".concat(name, "Integration"), {
                restApi: api.id,
                resourceId: args.resourceId,
                httpMethod: method.httpMethod,
                integrationHttpMethod: "POST",
                type: "AWS_PROXY",
                uri: fn.invokeArn,
            }, { parent: self, dependsOn: [permission] }), false)))();
        }
        return _this;
    }
    Object.defineProperty(ApiGatewayV1LambdaRoute.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var self = this;
            return {
                /**
                 * The Lambda function.
                 */
                get function() {
                    return self.fn.apply(function (fn) { return fn.getFunction(); });
                },
                /**
                 * The Lambda permission.
                 */
                permission: this.permission,
                /**
                 * The API Gateway REST API integration.
                 */
                integration: this.integration,
                /**
                 * The API Gateway REST API method.
                 */
                method: this.method,
            };
        },
        enumerable: false,
        configurable: true
    });
    return ApiGatewayV1LambdaRoute;
}(component_1.Component));
exports.ApiGatewayV1LambdaRoute = ApiGatewayV1LambdaRoute;
var __pulumiType = "sst:aws:ApiGatewayV1LambdaRoute";
// @ts-expect-error
ApiGatewayV1LambdaRoute.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2;
