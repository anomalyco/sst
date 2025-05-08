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
exports.ApiGatewayV2Authorizer = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var error_1 = require("../error");
var duration_1 = require("../duration");
var function_builder_1 = require("./helpers/function-builder");
/**
 * The `ApiGatewayV2Authorizer` component is internally used by the `ApiGatewayV2` component
 * to add authorizers to [Amazon API Gateway HTTP API](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addAuthorizer` method of the `ApiGatewayV2` component.
 */
var ApiGatewayV2Authorizer = /** @class */ (function (_super) {
    __extends(ApiGatewayV2Authorizer, _super);
    function ApiGatewayV2Authorizer(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var api = (0, pulumi_1.output)(args.api);
        var lamb = args.lambda && (0, pulumi_1.output)(args.lambda);
        var jwt = args.jwt && (0, pulumi_1.output)(args.jwt);
        validateSingleAuthorizer();
        var fn = createFunction();
        var authorizer = createAuthorizer();
        createPermission();
        _this.authorizer = authorizer;
        function validateSingleAuthorizer() {
            var authorizers = [lamb, jwt].filter(function (e) { return e; });
            if (authorizers.length === 0)
                throw new error_1.VisibleError("Please provide one of \"lambda\" or \"jwt\" for the ".concat(args.name, " authorizer."));
            if (authorizers.length > 1)
                throw new error_1.VisibleError("Please provide only one of \"lambda\" or \"jwt\" for the ".concat(args.name, " authorizer."));
        }
        function createFunction() {
            if (!lamb)
                return;
            return (0, function_builder_1.functionBuilder)("".concat(name, "Handler"), lamb.function, {
                description: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["", " authorizer"], ["", " authorizer"])), api.name),
            }, undefined, { parent: self });
        }
        function createAuthorizer() {
            var _a;
            var _b;
            var defaultIdentitySource = args.type === "http"
                ? "$request.header.Authorization"
                : "route.request.header.Authorization";
            return new ((_a = aws_1.apigatewayv2.Authorizer).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.authorizer, "".concat(name, "Authorizer"), __assign({ apiId: api.id }, (lamb
                ? __assign({ authorizerType: "REQUEST", identitySources: lamb.apply(function (lamb) { var _a; return (_a = lamb.identitySources) !== null && _a !== void 0 ? _a : [defaultIdentitySource]; }), authorizerUri: fn.invokeArn }, (args.type === "http"
                    ? {
                        authorizerResultTtlInSeconds: lamb.apply(function (lamb) { var _a; return (0, duration_1.toSeconds)((_a = lamb.ttl) !== null && _a !== void 0 ? _a : "0 seconds"); }),
                        authorizerPayloadFormatVersion: lamb.apply(function (lamb) { var _a; return (_a = lamb.payload) !== null && _a !== void 0 ? _a : "2.0"; }),
                        enableSimpleResponses: lamb.apply(function (lamb) { var _a; return ((_a = lamb.response) !== null && _a !== void 0 ? _a : "simple") === "simple"; }),
                    }
                    : {})) : {
                authorizerType: "JWT",
                identitySources: [
                    jwt.apply(function (jwt) { var _a; return (_a = jwt.identitySource) !== null && _a !== void 0 ? _a : defaultIdentitySource; }),
                ],
                jwtConfiguration: jwt.apply(function (jwt) { return ({
                    audiences: jwt.audiences,
                    issuer: jwt.issuer,
                }); }),
            })), { parent: self }), false)))();
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
        return _this;
    }
    Object.defineProperty(ApiGatewayV2Authorizer.prototype, "id", {
        /**
         * The ID of the authorizer.
         */
        get: function () {
            return this.authorizer.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(ApiGatewayV2Authorizer.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The API Gateway V2 authorizer.
                 */
                authorizer: this.authorizer,
            };
        },
        enumerable: false,
        configurable: true
    });
    return ApiGatewayV2Authorizer;
}(component_1.Component));
exports.ApiGatewayV2Authorizer = ApiGatewayV2Authorizer;
var __pulumiType = "sst:aws:ApiGatewayV2Authorizer";
// @ts-expect-error
ApiGatewayV2Authorizer.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2;
