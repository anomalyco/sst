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
exports.ApiGatewayV2UrlRoute = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var apigatewayv2_base_route_1 = require("./apigatewayv2-base-route");
/**
 * The `ApiGatewayV2UrlRoute` component is internally used by the `ApiGatewayV2` component
 * to add routes to [Amazon API Gateway HTTP API](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `routeUrl` method of the `ApiGatewayV2` component.
 */
var ApiGatewayV2UrlRoute = /** @class */ (function (_super) {
    __extends(ApiGatewayV2UrlRoute, _super);
    function ApiGatewayV2UrlRoute(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var api = (0, pulumi_1.output)(args.api);
        var integration = createIntegration();
        var apiRoute = (0, apigatewayv2_base_route_1.createApiRoute)(name, args, integration.id, self);
        _this.apiRoute = apiRoute;
        _this.integration = integration;
        function createIntegration() {
            var _a;
            var _b;
            return new ((_a = aws_1.apigatewayv2.Integration).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.integration, "".concat(name, "Integration"), {
                apiId: api.id,
                integrationType: "HTTP_PROXY",
                integrationUri: args.url,
                integrationMethod: "ANY",
            }, { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(ApiGatewayV2UrlRoute.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
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
    return ApiGatewayV2UrlRoute;
}(component_1.Component));
exports.ApiGatewayV2UrlRoute = ApiGatewayV2UrlRoute;
var __pulumiType = "sst:aws:ApiGatewayV2UrlRoute";
// @ts-expect-error
ApiGatewayV2UrlRoute.__pulumiType = __pulumiType;
