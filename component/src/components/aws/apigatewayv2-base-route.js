"use strict";
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
exports.createApiRoute = createApiRoute;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
function createApiRoute(name, args, integrationId, parent) {
    var authArgs = (0, pulumi_1.output)(args.auth).apply(function (auth) {
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
        return new ((_a = aws_1.apigatewayv2.Route).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.route, "".concat(name, "Route"), __assign({ apiId: (0, pulumi_1.output)(args.api).id, routeKey: args.route, target: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["integrations/", ""], ["integrations/", ""])), integrationId) }, authArgs), { parent: parent }), false)))();
    });
}
var templateObject_1;
