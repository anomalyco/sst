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
exports.ApiGatewayWebSocketRoute = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var function_builder_1 = require("./helpers/function-builder");
/**
 * The `ApiGatewayWebSocketRoute` component is internally used by the `ApiGatewayWebSocket`
 * component to add routes to your [API Gateway WebSocket API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-websocket-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `route` method of the `ApiGatewayWebSocket` component.
 */
var ApiGatewayWebSocketRoute = /** @class */ (function (_super) {
    __extends(ApiGatewayWebSocketRoute, _super);
    function ApiGatewayWebSocketRoute(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var api = (0, pulumi_1.output)(args.api);
        var route = (0, pulumi_1.output)(args.route);
        var fn = createFunction();
        var permission = createPermission();
        var integration = createIntegration();
        var apiRoute = createApiRoute();
        _this.fn = fn;
        _this.permission = permission;
        _this.apiRoute = apiRoute;
        _this.integration = integration;
        function createFunction() {
            return (0, function_builder_1.functionBuilder)("".concat(name, "Handler"), args.handler, {
                description: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["", " route ", ""], ["", " route ", ""])), api.name, route),
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
            return new ((_a = aws_1.apigatewayv2.Integration).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.integration, "".concat(name, "Integration"), {
                apiId: api.id,
                integrationType: "AWS_PROXY",
                integrationUri: fn.arn.apply(function (arn) {
                    var _a = arn.split(":"), partition = _a[1], region = _a[3];
                    return "arn:".concat(partition, ":apigateway:").concat(region, ":lambda:path/2015-03-31/functions/").concat(arn, "/invocations");
                }),
            }, { parent: self, dependsOn: [permission] }), false)))();
        }
        function createApiRoute() {
            var authArgs = (0, pulumi_1.all)([args.route, args.auth]).apply(function (_a) {
                var route = _a[0], auth = _a[1];
                if (route !== "$connect")
                    return { authorizationType: "NONE" };
                if (!auth)
                    return { authorizationType: "NONE" };
                if (auth.iam)
                    return { authorizationType: "AWS_IAM" };
                if (auth.lambda)
                    return {
                        authorizationType: "CUSTOM",
                        authorizerId: auth.lambda,
                    };
                if (auth.jwt)
                    return {
                        authorizationType: "JWT",
                        authorizationScopes: auth.jwt.scopes,
                        authorizerId: auth.jwt.authorizer,
                    };
                return { authorizationType: "NONE" };
            });
            return authArgs.apply(function (authArgs) {
                var _a;
                var _b;
                return new ((_a = aws_1.apigatewayv2.Route).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.route, "".concat(name, "Route"), __assign({ apiId: api.id, routeKey: route, target: (0, pulumi_1.interpolate)(templateObject_3 || (templateObject_3 = __makeTemplateObject(["integrations/", ""], ["integrations/", ""])), integration.id) }, authArgs), { parent: self }), false)))();
            });
        }
        return _this;
    }
    Object.defineProperty(ApiGatewayWebSocketRoute.prototype, "nodes", {
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
                 * The API Gateway HTTP API route.
                 */
                route: this.apiRoute,
                /**
                 * The API Gateway HTTP API integration.
                 */
                integration: this.integration,
            };
        },
        enumerable: false,
        configurable: true
    });
    return ApiGatewayWebSocketRoute;
}(component_1.Component));
exports.ApiGatewayWebSocketRoute = ApiGatewayWebSocketRoute;
var __pulumiType = "sst:aws:ApiGatewayWebSocketRoute";
// @ts-expect-error
ApiGatewayWebSocketRoute.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2, templateObject_3;
