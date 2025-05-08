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
exports.CognitoIdentityPool = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var permission_1 = require("./permission");
var arn_1 = require("./helpers/arn");
/**
 * The `CognitoIdentityPool` component lets you add a [Amazon Cognito identity pool](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-identity.html) to your app.
 *
 * #### Create the identity pool
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.CognitoIdentityPool("MyIdentityPool", {
 *   userPools: [
 *     {
 *       userPool: "us-east-1_QY6Ly46JH",
 *       client: "6va5jg3cgtrd170sgokikjm5m6"
 *     }
 *   ]
 * });
 * ```
 *
 * #### Configure permissions for authenticated users
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.CognitoIdentityPool("MyIdentityPool", {
 *   userPools: [
 *     {
 *       userPool: "us-east-1_QY6Ly46JH",
 *       client: "6va5jg3cgtrd170sgokikjm5m6"
 *     }
 *   ],
 *   permissions: {
 *     authenticated: [
 *       {
 *         actions: ["s3:GetObject", "s3:PutObject"],
 *         resources: ["arn:aws:s3:::my-bucket/*"]
 *       }
 *     ]
 *   }
 * });
 * ```
 */
var CognitoIdentityPool = /** @class */ (function (_super) {
    __extends(CognitoIdentityPool, _super);
    function CognitoIdentityPool(name, args, opts) {
        if (args === void 0) { args = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        if (args && "ref" in args) {
            var ref = args;
            _this.identityPool = ref.identityPool;
            _this.authRole = ref.authRole;
            _this.unauthRole = ref.unauthRole;
            return _this;
        }
        var parent = _this;
        var region = getRegion();
        var identityPool = createIdentityPool();
        var authRole = createAuthRole();
        var unauthRole = createUnauthRole();
        createRoleAttachment();
        _this.identityPool = identityPool;
        _this.authRole = authRole;
        _this.unauthRole = unauthRole;
        function getRegion() {
            return (0, aws_1.getRegionOutput)(undefined, { parent: parent }).name;
        }
        function createIdentityPool() {
            var _a;
            var _b;
            return new ((_a = aws_1.cognito.IdentityPool).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.identityPool, "".concat(name, "IdentityPool"), {
                identityPoolName: "",
                allowUnauthenticatedIdentities: true,
                cognitoIdentityProviders: args.userPools &&
                    (0, pulumi_1.output)(args.userPools).apply(function (userPools) {
                        return userPools.map(function (v) { return ({
                            clientId: v.client,
                            providerName: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["cognito-idp.", ".amazonaws.com/", ""], ["cognito-idp.", ".amazonaws.com/", ""])), region, v.userPool),
                        }); });
                    }),
                supportedLoginProviders: {},
            }, { parent: parent }), false)))();
        }
        function createAuthRole() {
            var _a;
            var _b;
            var policy = (0, pulumi_1.output)(args.permissions).apply(function (permissions) {
                return aws_1.iam.getPolicyDocumentOutput({
                    statements: __spreadArray([
                        {
                            effect: "Allow",
                            actions: [
                                "mobileanalytics:PutEvents",
                                "cognito-sync:*",
                                "cognito-identity:*",
                            ],
                            resources: ["*"],
                        }
                    ], ((permissions === null || permissions === void 0 ? void 0 : permissions.authenticated) || []), true),
                });
            });
            return new ((_a = aws_1.iam.Role).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.authenticatedRole, "".concat(name, "AuthRole"), {
                assumeRolePolicy: aws_1.iam.getPolicyDocumentOutput({
                    statements: [
                        {
                            effect: "Allow",
                            principals: [
                                {
                                    type: "Federated",
                                    identifiers: ["cognito-identity.amazonaws.com"],
                                },
                            ],
                            actions: ["sts:AssumeRoleWithWebIdentity"],
                            conditions: [
                                {
                                    test: "StringEquals",
                                    variable: "cognito-identity.amazonaws.com:aud",
                                    values: [identityPool.id],
                                },
                                {
                                    test: "ForAnyValue:StringLike",
                                    variable: "cognito-identity.amazonaws.com:amr",
                                    values: ["authenticated"],
                                },
                            ],
                        },
                    ],
                }).json,
                inlinePolicies: [{ name: "inline", policy: policy.json }],
            }, { parent: parent }), false)))();
        }
        function createUnauthRole() {
            var _a;
            var _b;
            var policy = (0, pulumi_1.output)(args.permissions).apply(function (permissions) {
                return aws_1.iam.getPolicyDocumentOutput({
                    statements: __spreadArray([
                        {
                            effect: "Allow",
                            actions: ["mobileanalytics:PutEvents", "cognito-sync:*"],
                            resources: ["*"],
                        }
                    ], ((permissions === null || permissions === void 0 ? void 0 : permissions.unauthenticated) || []), true),
                });
            });
            return new ((_a = aws_1.iam.Role).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.unauthenticatedRole, "".concat(name, "UnauthRole"), {
                assumeRolePolicy: aws_1.iam.getPolicyDocumentOutput({
                    statements: [
                        {
                            effect: "Allow",
                            principals: [
                                {
                                    type: "Federated",
                                    identifiers: ["cognito-identity.amazonaws.com"],
                                },
                            ],
                            actions: ["sts:AssumeRoleWithWebIdentity"],
                            conditions: [
                                {
                                    test: "StringEquals",
                                    variable: "cognito-identity.amazonaws.com:aud",
                                    values: [identityPool.id],
                                },
                                {
                                    test: "ForAnyValue:StringLike",
                                    variable: "cognito-identity.amazonaws.com:amr",
                                    values: ["unauthenticated"],
                                },
                            ],
                        },
                    ],
                }).json,
                inlinePolicies: [{ name: "inline", policy: policy.json }],
            }, { parent: parent }), false)))();
        }
        function createRoleAttachment() {
            return new aws_1.cognito.IdentityPoolRoleAttachment("".concat(name, "RoleAttachment"), {
                identityPoolId: identityPool.id,
                roles: {
                    authenticated: authRole.arn,
                    unauthenticated: unauthRole.arn,
                },
            }, { parent: parent });
        }
        return _this;
    }
    Object.defineProperty(CognitoIdentityPool.prototype, "id", {
        /**
         * The Cognito identity pool ID.
         */
        get: function () {
            return this.identityPool.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(CognitoIdentityPool.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon Cognito identity pool.
                 */
                identityPool: this.identityPool,
                /**
                 * The authenticated IAM role.
                 */
                authenticatedRole: this.authRole,
                /**
                 * The unauthenticated IAM role.
                 */
                unauthenticatedRole: this.unauthRole,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    CognitoIdentityPool.prototype.getSSTLink = function () {
        return {
            properties: {
                id: this.id,
            },
            include: [
                (0, permission_1.permission)({
                    actions: ["cognito-identity:*"],
                    resources: [this.identityPool.arn],
                }),
            ],
        };
    };
    /**
     * Reference an existing Identity Pool with the given ID. This is useful when you
     * create a Identity Pool in one stage and want to share it in another. It avoids having to
     * create a new Identity Pool in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share Identity Pools across stages.
     * :::
     *
     * @param name The name of the component.
     * @param identityPoolID The ID of the existing Identity Pool.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a Identity Pool in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new pool, you want to share the same pool from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const identityPool = $app.stage === "frank"
     *   ? sst.aws.CognitoIdentityPool.get("MyIdentityPool", "us-east-1:02facf30-e2f3-49ec-9e79-c55187415cf8")
     *   : new sst.aws.CognitoIdentityPool("MyIdentityPool");
     * ```
     *
     * Here `us-east-1:02facf30-e2f3-49ec-9e79-c55187415cf8` is the ID of the Identity Pool created in the `dev` stage.
     * You can find this by outputting the Identity Pool ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   identityPool: identityPool.id
     * };
     * ```
     */
    CognitoIdentityPool.get = function (name, identityPoolID, opts) {
        var identityPool = aws_1.cognito.IdentityPool.get("".concat(name, "IdentityPool"), identityPoolID, undefined, opts);
        var attachment = aws_1.cognito.IdentityPoolRoleAttachment.get("".concat(name, "RoleAttachment"), identityPoolID, undefined, opts);
        var authRole = aws_1.iam.Role.get("".concat(name, "AuthRole"), attachment.roles.authenticated.apply(function (arn) { return (0, arn_1.parseRoleArn)(arn).roleName; }), undefined, opts);
        var unauthRole = aws_1.iam.Role.get("".concat(name, "UnauthRole"), attachment.roles.unauthenticated.apply(function (arn) { return (0, arn_1.parseRoleArn)(arn).roleName; }), undefined, opts);
        return new CognitoIdentityPool(name, {
            ref: true,
            identityPool: identityPool,
            authRole: authRole,
            unauthRole: unauthRole,
        });
    };
    return CognitoIdentityPool;
}(component_1.Component));
exports.CognitoIdentityPool = CognitoIdentityPool;
var __pulumiType = "sst:aws:CognitoIdentityPool";
// @ts-expect-error
CognitoIdentityPool.__pulumiType = __pulumiType;
var templateObject_1;
