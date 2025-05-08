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
exports.CognitoIdentityProvider = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var error_1 = require("../error");
/**
 * The `CognitoIdentityProvider` component is internally used by the `CognitoUserPool`
 * component to add identity providers to your [Amazon Cognito user pool](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-identity-pools.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addIdentityProvider` method of the `CognitoUserPool` component.
 */
var CognitoIdentityProvider = /** @class */ (function (_super) {
    __extends(CognitoIdentityProvider, _super);
    function CognitoIdentityProvider(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var providerType = normalizeProviderType();
        var identityProvider = createIdentityProvider();
        _this.identityProvider = identityProvider;
        function normalizeProviderType() {
            var type = (0, pulumi_1.output)(args.type).apply(function (type) {
                return ({
                    saml: "SAML",
                    oidc: "OIDC",
                    facebook: "Facebook",
                    google: "Google",
                    amazon: "LoginWithAmazon",
                    apple: "SignInWithApple",
                })[type];
            });
            if (!type)
                throw new error_1.VisibleError("Invalid provider type: ".concat(args.type));
            return type;
        }
        function createIdentityProvider() {
            var _a;
            var _b;
            return new ((_a = aws_1.cognito.IdentityProvider).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.identityProvider, "".concat(name, "IdentityProvider"), {
                userPoolId: args.userPool,
                providerName: name,
                providerType: providerType,
                providerDetails: args.details,
                attributeMapping: args.attributes,
            }, { parent: parent }), false)))();
        }
        return _this;
    }
    Object.defineProperty(CognitoIdentityProvider.prototype, "providerName", {
        /**
         * The Cognito identity provider name.
         */
        get: function () {
            return this.identityProvider.providerName;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(CognitoIdentityProvider.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Cognito identity provider.
                 */
                identityProvider: this.identityProvider,
            };
        },
        enumerable: false,
        configurable: true
    });
    return CognitoIdentityProvider;
}(component_1.Component));
exports.CognitoIdentityProvider = CognitoIdentityProvider;
var __pulumiType = "sst:aws:CognitoIdentityProvider";
// @ts-expect-error
CognitoIdentityProvider.__pulumiType = __pulumiType;
