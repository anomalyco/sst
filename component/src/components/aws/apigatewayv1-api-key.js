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
Object.defineProperty(exports, "__esModule", { value: true });
exports.ApiGatewayV1ApiKey = void 0;
var aws_1 = require("@pulumi/aws");
var component_1 = require("../component");
/**
 * The `ApiGatewayV1ApiKey` component is internally used by the `ApiGatewayV1UsagePlan` component
 * to add API keys to [Amazon API Gateway REST API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addApiKey` method of the `ApiGatewayV1UsagePlan` component.
 */
var ApiGatewayV1ApiKey = /** @class */ (function (_super) {
    __extends(ApiGatewayV1ApiKey, _super);
    function ApiGatewayV1ApiKey(name, args, opts) {
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        _this.key = new aws_1.apigateway.ApiKey("".concat(name, "ApiKey"), {
            value: args.value,
        }, { parent: self });
        new aws_1.apigateway.UsagePlanKey("".concat(name, "UsagePlanKey"), {
            keyId: _this.key.id,
            keyType: "API_KEY",
            usagePlanId: args.usagePlanId,
        }, { parent: self });
        return _this;
    }
    Object.defineProperty(ApiGatewayV1ApiKey.prototype, "value", {
        /**
         * The API key value.
         */
        get: function () {
            return this.key.value;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(ApiGatewayV1ApiKey.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The API Gateway API Key.
                 */
                apiKey: this.key,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    ApiGatewayV1ApiKey.prototype.getSSTLink = function () {
        return {
            properties: {
                value: this.value,
            },
        };
    };
    return ApiGatewayV1ApiKey;
}(component_1.Component));
exports.ApiGatewayV1ApiKey = ApiGatewayV1ApiKey;
var __pulumiType = "sst:aws:ApiGatewayV1ApiKey";
// @ts-expect-error
ApiGatewayV1ApiKey.__pulumiType = __pulumiType;
