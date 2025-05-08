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
Object.defineProperty(exports, "__esModule", { value: true });
exports.ApiGatewayV1UsagePlan = void 0;
var aws_1 = require("@pulumi/aws");
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var apigatewayv1_api_key_1 = require("./apigatewayv1-api-key");
/**
 * The `ApiGatewayV1UsagePlan` component is internally used by the `ApiGatewayV1` component
 * to add usage plans to [Amazon API Gateway REST API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addUsagePlan` method of the `ApiGatewayV1` component.
 */
var ApiGatewayV1UsagePlan = /** @class */ (function (_super) {
    __extends(ApiGatewayV1UsagePlan, _super);
    function ApiGatewayV1UsagePlan(name, args, opts) {
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        _this.plan = new aws_1.apigateway.UsagePlan("".concat(name, "UsagePlan"), {
            apiStages: [{ apiId: args.apiId, stage: args.apiStage }],
            quotaSettings: args.quota &&
                (0, pulumi_1.output)(args.quota).apply(function (quota) { return ({
                    limit: quota.limit,
                    period: quota.period.toUpperCase(),
                    offset: quota.offset,
                }); }),
            throttleSettings: args.throttle &&
                (0, pulumi_1.output)(args.throttle).apply(function (throttle) { return ({
                    burstLimit: throttle.burst,
                    rateLimit: throttle.rate,
                }); }),
        }, { parent: self });
        _this.constructorArgs = args;
        _this.constructorOpts = opts;
        return _this;
    }
    Object.defineProperty(ApiGatewayV1UsagePlan.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The API Gateway Usage Plan.
                 */
                usagePlan: this.plan,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Add an API key to the API Gateway usage plan.
     *
     * @param name The name of the API key.
     * @param args Configure the API key.
     * @example
     * ```js title="sst.config.ts"
     * plan.addApiKey("MyKey", {
     *   value: "d41d8cd98f00b204e9800998ecf8427e",
     * });
     * ```
     */
    ApiGatewayV1UsagePlan.prototype.addApiKey = function (name, args) {
        return new apigatewayv1_api_key_1.ApiGatewayV1ApiKey(name, __assign({ apiId: this.constructorArgs.apiId, usagePlanId: this.plan.id }, args), { provider: this.constructorOpts.provider });
    };
    return ApiGatewayV1UsagePlan;
}(component_1.Component));
exports.ApiGatewayV1UsagePlan = ApiGatewayV1UsagePlan;
var __pulumiType = "sst:aws:ApiGatewayV1UsagePlan";
// @ts-expect-error
ApiGatewayV1UsagePlan.__pulumiType = __pulumiType;
