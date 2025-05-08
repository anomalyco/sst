"use strict";
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
exports.createMethod = createMethod;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
function createMethod(name, args, parent) {
    var api = args.api, method = args.method, resourceId = args.resourceId, auth = args.auth;
    var authArgs = (0, pulumi_1.output)(auth).apply(function (auth) {
        if (!auth)
            return { authorization: "NONE" };
        if (auth.iam)
            return { authorization: "AWS_IAM" };
        if (auth.custom)
            return { authorization: "CUSTOM", authorizerId: auth.custom };
        if (auth.cognito)
            return {
                authorization: "COGNITO_USER_POOLS",
                authorizerId: auth.cognito.authorizer,
                authorizationScopes: auth.cognito.scopes,
            };
        return { authorization: "NONE" };
    });
    return authArgs.apply(function (authArgs) {
        var _a;
        var _b;
        return new ((_a = aws_1.apigateway.Method).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.method, "".concat(name, "Method"), {
            restApi: (0, pulumi_1.output)(api).id,
            resourceId: resourceId,
            httpMethod: method,
            authorization: authArgs.authorization,
            authorizerId: authArgs.authorizerId,
            authorizationScopes: authArgs.authorizationScopes,
        }, { parent: parent }), false)))();
    });
}
