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
exports.CognitoUserPool = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var cognito_identity_provider_1 = require("./cognito-identity-provider");
var cognito_user_pool_client_1 = require("./cognito-user-pool-client");
var error_1 = require("../error");
var aws_1 = require("@pulumi/aws");
var permission_1 = require("./permission");
var function_builder_1 = require("./helpers/function-builder");
/**
 * The `CognitoUserPool` component lets you add a [Amazon Cognito User Pool](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-identity-pools.html) to your app.
 *
 * #### Create the user pool
 *
 * ```ts title="sst.config.ts"
 * const userPool = new sst.aws.CognitoUserPool("MyUserPool");
 * ```
 *
 * #### Login using email
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.CognitoUserPool("MyUserPool", {
 *   usernames: ["email"]
 * });
 * ```
 *
 * #### Configure triggers
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.CognitoUserPool("MyUserPool", {
 *   triggers: {
 *     preAuthentication: "src/preAuthentication.handler",
 *     postAuthentication: "src/postAuthentication.handler",
 *   },
 * });
 * ```
 *
 * #### Add Google identity provider
 *
 * ```ts title="sst.config.ts"
 * const GoogleClientId = new sst.Secret("GOOGLE_CLIENT_ID");
 * const GoogleClientSecret = new sst.Secret("GOOGLE_CLIENT_SECRET");
 *
 * userPool.addIdentityProvider({
 *   type: "google",
 *   details: {
 *     authorize_scopes: "email profile",
 *     client_id: GoogleClientId.value,
 *     client_secret: GoogleClientSecret.value,
 *   },
 *   attributes: {
 *     email: "email",
 *     name: "name",
 *     username: "sub",
 *   },
 * });
 * ```
 *
 * #### Add a client
 *
 * ```ts title="sst.config.ts"
 * userPool.addClient("Web");
 * ```
 */
var CognitoUserPool = /** @class */ (function (_super) {
    __extends(CognitoUserPool, _super);
    function CognitoUserPool(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        if (args && "ref" in args) {
            var ref = args;
            _this.constructorOpts = opts;
            _this.userPool = (0, pulumi_1.output)(ref.userPool);
            return _this;
        }
        var parent = _this;
        normalizeAliasesAndUsernames();
        var triggers = normalizeTriggers();
        var verify = normalizeVerify();
        var userPool = createUserPool();
        _this.constructorOpts = opts;
        _this.userPool = userPool;
        function normalizeAliasesAndUsernames() {
            (0, pulumi_1.all)([args.aliases, args.usernames]).apply(function (_a) {
                var aliases = _a[0], usernames = _a[1];
                if (aliases && usernames)
                    throw new error_1.VisibleError("You cannot set both aliases and usernames. Learn more about customizing sign-in attributes at https://docs.aws.amazon.com/cognito/latest/developerguide/user-pool-settings-attributes.html#user-pool-settings-aliases");
            });
        }
        function normalizeTriggers() {
            if (!args.triggers)
                return;
            return (0, pulumi_1.output)(args.triggers).apply(function (triggers) {
                if ((triggers.customEmailSender || triggers.customSmsSender) &&
                    !triggers.kmsKey)
                    throw new error_1.VisibleError("You must provide a KMS key via `kmsKey` when configuring `customEmailSender` or `customSmsSender`.");
                return __assign(__assign({}, triggers), { preTokenGenerationVersion: triggers.preTokenGenerationVersion === "v2" ? "V2_0" : "V1_0" });
            });
        }
        function normalizeVerify() {
            if (!args.verify)
                return;
            return (0, pulumi_1.output)(args.verify).apply(function (verify) {
                var _a, _b, _c;
                return {
                    defaultEmailOption: "CONFIRM_WITH_CODE",
                    emailMessage: (_a = verify.emailMessage) !== null && _a !== void 0 ? _a : "The verification code to your new account is {####}",
                    emailSubject: (_b = verify.emailSubject) !== null && _b !== void 0 ? _b : "Verify your new account",
                    smsMessage: (_c = verify.smsMessage) !== null && _c !== void 0 ? _c : "The verification code to your new account is {####}",
                };
            });
        }
        function createUserPool() {
            return (0, pulumi_1.output)(args.softwareToken).apply(function (softwareToken) {
                var _a;
                var _b;
                return new ((_a = aws_1.cognito.UserPool).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.userPool, "".concat(name, "UserPool"), {
                    aliasAttributes: args.aliases &&
                        (0, pulumi_1.output)(args.aliases).apply(function (aliases) { return __spreadArray(__spreadArray(__spreadArray([], (aliases.includes("email") ? ["email"] : []), true), (aliases.includes("phone") ? ["phone_number"] : []), true), (aliases.includes("preferred_username")
                            ? ["preferred_username"]
                            : []), true); }),
                    usernameAttributes: args.usernames &&
                        (0, pulumi_1.output)(args.usernames).apply(function (usernames) { return __spreadArray(__spreadArray([], (usernames.includes("email") ? ["email"] : []), true), (usernames.includes("phone") ? ["phone_number"] : []), true); }),
                    accountRecoverySetting: {
                        recoveryMechanisms: [
                            {
                                name: "verified_phone_number",
                                priority: 1,
                            },
                            {
                                name: "verified_email",
                                priority: 2,
                            },
                        ],
                    },
                    adminCreateUserConfig: {
                        allowAdminCreateUserOnly: false,
                    },
                    usernameConfiguration: {
                        caseSensitive: false,
                    },
                    autoVerifiedAttributes: (0, pulumi_1.all)([
                        args.aliases || [],
                        args.usernames || [],
                    ]).apply(function (_a) {
                        var aliases = _a[0], usernames = _a[1];
                        var attributes = __spreadArray(__spreadArray([], aliases, true), usernames, true);
                        return __spreadArray(__spreadArray([], (attributes.includes("email") ? ["email"] : []), true), (attributes.includes("phone") ? ["phone_number"] : []), true);
                    }),
                    emailConfiguration: {
                        emailSendingAccount: "COGNITO_DEFAULT",
                    },
                    verificationMessageTemplate: verify,
                    userPoolAddOns: {
                        advancedSecurityMode: (0, pulumi_1.output)(args.advancedSecurity).apply(function (v) { return (v !== null && v !== void 0 ? v : "off").toUpperCase(); }),
                    },
                    mfaConfiguration: (0, pulumi_1.output)(args.mfa).apply(function (v) {
                        return (v !== null && v !== void 0 ? v : "off").toUpperCase();
                    }),
                    smsAuthenticationMessage: args.smsAuthenticationMessage,
                    smsConfiguration: args.sms,
                    softwareTokenMfaConfiguration: softwareToken
                        ? { enabled: true }
                        : undefined,
                    lambdaConfig: triggers &&
                        triggers.apply(function (triggers) {
                            return {
                                kmsKeyId: triggers.kmsKey,
                                createAuthChallenge: createTrigger("createAuthChallenge"),
                                customEmailSender: triggers.customEmailSender === undefined
                                    ? undefined
                                    : {
                                        lambdaArn: createTrigger("customEmailSender"),
                                        lambdaVersion: "V1_0",
                                    },
                                customMessage: createTrigger("customMessage"),
                                customSmsSender: triggers.customSmsSender === undefined
                                    ? undefined
                                    : {
                                        lambdaArn: createTrigger("customSmsSender"),
                                        lambdaVersion: "V1_0",
                                    },
                                defineAuthChallenge: createTrigger("defineAuthChallenge"),
                                postAuthentication: createTrigger("postAuthentication"),
                                postConfirmation: createTrigger("postConfirmation"),
                                preAuthentication: createTrigger("preAuthentication"),
                                preSignUp: createTrigger("preSignUp"),
                                preTokenGenerationConfig: triggers.preTokenGeneration === undefined
                                    ? undefined
                                    : {
                                        lambdaArn: createTrigger("preTokenGeneration"),
                                        lambdaVersion: triggers.preTokenGenerationVersion,
                                    },
                                userMigration: createTrigger("userMigration"),
                                verifyAuthChallengeResponse: createTrigger("verifyAuthChallengeResponse"),
                            };
                            function createTrigger(key) {
                                if (!triggers[key])
                                    return;
                                var fn = (0, function_builder_1.functionBuilder)("".concat(name, "Trigger").concat(key), triggers[key], {
                                    description: "Subscribed to ".concat(key, " from ").concat(name),
                                }, undefined, { parent: parent });
                                new aws_1.lambda.Permission("".concat(name, "Permission").concat(key), {
                                    action: "lambda:InvokeFunction",
                                    function: fn.arn,
                                    principal: "cognito-idp.amazonaws.com",
                                    sourceArn: userPool.arn,
                                }, { parent: parent });
                                return fn.arn;
                            }
                        }),
                }, { parent: parent }), false)))();
            });
        }
        return _this;
    }
    Object.defineProperty(CognitoUserPool.prototype, "id", {
        /**
         * The Cognito User Pool ID.
         */
        get: function () {
            return this.userPool.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(CognitoUserPool.prototype, "arn", {
        /**
         * The Cognito User Pool ARN.
         */
        get: function () {
            return this.userPool.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(CognitoUserPool.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon Cognito User Pool.
                 */
                userPool: this.userPool,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Add a client to the User Pool.
     *
     * @param name Name of the client.
     * @param args Configure the client.
     * @param opts? Resource options.
     *
     * @example
     *
     * ```ts
     * userPool.addClient("Web");
     * ```
     */
    CognitoUserPool.prototype.addClient = function (name, args) {
        // Note: Referencing an existing client will be implemented in the future:
        // sst.aws.UserPool.getClient("pool", { userPooldID, clientID });
        return new cognito_user_pool_client_1.CognitoUserPoolClient(name, __assign({ userPool: this.id }, args), { provider: this.constructorOpts.provider });
    };
    /**
     * Add a federated identity provider to the User Pool.
     *
     * @param name Name of the identity provider.
     * @param args Configure the identity provider.
     *
     * @example
     *
     * For example, add a GitHub (OIDC) identity provider.
     *
     * ```ts title="sst.config.ts"
     * const GithubClientId = new sst.Secret("GITHUB_CLIENT_ID");
     * const GithubClientSecret = new sst.Secret("GITHUB_CLIENT_SECRET");
     *
     * userPool.addIdentityProvider("GitHub", {
     *   type: "oidc",
     *   details: {
     *      authorize_scopes: "read:user user:email",
     *      client_id: GithubClientId.value,
     *      client_secret: GithubClientSecret.value,
     *      oidc_issuer: "https://github.com/",
     *   },
     *   attributes: {
     *     email: "email",
     *     username: "sub",
     *   },
     * });
     * ```
     *
     * Or add a Google identity provider.
     *
     * ```ts title="sst.config.ts"
     * const GoogleClientId = new sst.Secret("GOOGLE_CLIENT_ID");
     * const GoogleClientSecret = new sst.Secret("GOOGLE_CLIENT_SECRET");
     *
     * userPool.addIdentityProvider("Google", {
     *   type: "google",
     *   details: {
     *     authorize_scopes: "email profile",
     *     client_id: GoogleClientId.value,
     *     client_secret: GoogleClientSecret.value,
     *   },
     *   attributes: {
     *     email: "email",
     *     name: "name",
     *     username: "sub",
     *   },
     * });
     * ```
     */
    CognitoUserPool.prototype.addIdentityProvider = function (name, args) {
        return new cognito_identity_provider_1.CognitoIdentityProvider(name, __assign({ userPool: this.id }, args), { provider: this.constructorOpts.provider });
    };
    /** @internal */
    CognitoUserPool.prototype.getSSTLink = function () {
        return {
            properties: {
                id: this.id,
            },
            include: [
                (0, permission_1.permission)({
                    actions: ["cognito-idp:*"],
                    resources: [this.userPool.arn],
                }),
            ],
        };
    };
    /**
     * Reference an existing User Pool with the given ID. This is useful when you
     * create a User Pool in one stage and want to share it in another. It avoids having to
     * create a new User Pool in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share User Pools across stages.
     * :::
     *
     * @param name The name of the component.
     * @param userPoolID The ID of the existing User Pool.
     *
     * @example
     * Imagine you create a User Pool in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new pool, you want to share the same pool from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const userPool = $app.stage === "frank"
     *   ? sst.aws.CognitoUserPool.get("MyUserPool", "us-east-1_gcF5PjhQK")
     *   : new sst.aws.CognitoUserPool("MyUserPool");
     * ```
     *
     * Here `us-east-1_gcF5PjhQK` is the ID of the User Pool created in the `dev` stage.
     * You can find this by outputting the User Pool ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   userPool: userPool.id
     * };
     * ```
     */
    CognitoUserPool.get = function (name, userPoolID, opts) {
        var userPool = aws_1.cognito.UserPool.get("".concat(name, "UserPool"), userPoolID, undefined, opts);
        return new CognitoUserPool(name, {
            ref: true,
            userPool: userPool,
        });
    };
    return CognitoUserPool;
}(component_1.Component));
exports.CognitoUserPool = CognitoUserPool;
var __pulumiType = "sst:aws:CognitoUserPool";
// @ts-expect-error
CognitoUserPool.__pulumiType = __pulumiType;
