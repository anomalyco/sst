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
exports.ApiGatewayV1IntegrationRoute = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var apigatewayv1_base_route_1 = require("./apigatewayv1-base-route");
/**
 * The `ApiGatewayV1IntegrationRoute` component is internally used by the `ApiGatewayV1` component
 * to add routes to your [API Gateway REST API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `routeIntegration` method of the `ApiGatewayV1` component.
 */
var ApiGatewayV1IntegrationRoute = /** @class */ (function (_super) {
    __extends(ApiGatewayV1IntegrationRoute, _super);
    function ApiGatewayV1IntegrationRoute(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var api = (0, pulumi_1.output)(args.api);
        var method = (0, apigatewayv1_base_route_1.createMethod)(name, args, self);
        var integration = createIntegration();
        _this.method = method;
        _this.integration = integration;
        function createIntegration() {
            var _a;
            var _b;
            return new ((_a = aws_1.apigateway.Integration).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.integration, "".concat(name, "Integration"), __assign(__assign({ restApi: api.id, resourceId: args.resourceId, httpMethod: method.httpMethod }, args.integration), { type: (0, pulumi_1.output)(args.integration.type).apply(function (v) {
                    return v.toUpperCase().replaceAll("-", "_");
                }), passthroughBehavior: args.integration.passthroughBehavior &&
                    (0, pulumi_1.output)(args.integration.passthroughBehavior).apply(function (v) {
                        return v.toUpperCase().replaceAll("-", "_");
                    }) }), { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(ApiGatewayV1IntegrationRoute.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
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
    return ApiGatewayV1IntegrationRoute;
}(component_1.Component));
exports.ApiGatewayV1IntegrationRoute = ApiGatewayV1IntegrationRoute;
var __pulumiType = "sst:aws:ApiGatewayV1IntegrationRoute";
// @ts-expect-error
ApiGatewayV1IntegrationRoute.__pulumiType = __pulumiType;
