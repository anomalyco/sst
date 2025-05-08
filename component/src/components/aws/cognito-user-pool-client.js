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
exports.CognitoUserPoolClient = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
/**
 * The `CognitoUserPoolClient` component is internally used by the `CognitoUserPool`
 * component to add clients to your [Amazon Cognito user pool](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-identity-pools.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addClient` method of the `CognitoUserPool` component.
 */
var CognitoUserPoolClient = /** @class */ (function (_super) {
    __extends(CognitoUserPoolClient, _super);
    function CognitoUserPoolClient(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var providers = normalizeProviders();
        var client = createClient();
        _this.client = client;
        function normalizeProviders() {
            if (!args.providers)
                return ["COGNITO"];
            return (0, pulumi_1.output)(args.providers);
        }
        function createClient() {
            var _a;
            var _b;
            return new ((_a = aws_1.cognito.UserPoolClient).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.client, "".concat(name, "Client"), {
                name: name,
                userPoolId: args.userPool,
                allowedOauthFlows: ["implicit", "code"],
                allowedOauthFlowsUserPoolClient: true,
                allowedOauthScopes: [
                    "profile",
                    "phone",
                    "email",
                    "openid",
                    "aws.cognito.signin.user.admin",
                ],
                callbackUrls: ["https://example.com"],
                supportedIdentityProviders: providers,
            }, { parent: parent }), false)))();
        }
        return _this;
    }
    Object.defineProperty(CognitoUserPoolClient.prototype, "id", {
        /**
         * The Cognito User Pool client ID.
         */
        get: function () {
            return this.client.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(CognitoUserPoolClient.prototype, "secret", {
        /**
         * The Cognito User Pool client secret.
         */
        get: function () {
            return this.client.clientSecret;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(CognitoUserPoolClient.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Cognito User Pool client.
                 */
                client: this.client,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    CognitoUserPoolClient.prototype.getSSTLink = function () {
        return {
            properties: {
                id: this.id,
                secret: this.secret,
            },
        };
    };
    return CognitoUserPoolClient;
}(component_1.Component));
exports.CognitoUserPoolClient = CognitoUserPoolClient;
var __pulumiType = "sst:aws:CognitoUserPoolClient";
// @ts-expect-error
CognitoUserPoolClient.__pulumiType = __pulumiType;
