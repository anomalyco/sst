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
exports.AppSyncDataSource = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var error_1 = require("../error");
var arn_1 = require("./helpers/arn");
var aws_1 = require("@pulumi/aws");
var function_builder_1 = require("./helpers/function-builder");
/**
 * The `AppSyncDataSource` component is internally used by the `AppSync` component to add
 * data sources to [AWS AppSync](https://docs.aws.amazon.com/appsync/latest/devguide/what-is-appsync.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addDataSource` method of the `AppSync` component.
 */
var AppSyncDataSource = /** @class */ (function (_super) {
    __extends(AppSyncDataSource, _super);
    function AppSyncDataSource(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var apiId = (0, pulumi_1.output)(args.apiId);
        validateSingleDataSource();
        var type = getType();
        var lambda = createFunction();
        var serviceRole = createServiceRole();
        var dataSource = createDataSource();
        _this.dataSource = dataSource;
        _this.lambda = lambda;
        _this.serviceRole = serviceRole;
        function validateSingleDataSource() {
            var sources = [
                args.lambda,
                args.dynamodb,
                args.elasticSearch,
                args.eventBridge,
                args.http,
                args.openSearch,
                args.rds,
            ].filter(function (source) { return source; });
            if (sources.length > 1) {
                throw new Error("Expected only one data source, but found ".concat(sources.length, "."));
            }
        }
        function getType() {
            if (args.lambda)
                return "AWS_LAMBDA";
            if (args.dynamodb)
                return "AMAZON_DYNAMODB";
            if (args.elasticSearch)
                return "AMAZON_ELASTICSEARCH";
            if (args.eventBridge)
                return "AMAZON_EVENTBRIDGE";
            if (args.http)
                return "HTTP";
            if (args.openSearch)
                return "AMAZON_OPENSEARCH_SERVICE";
            if (args.rds)
                return "RELATIONAL_DATABASE";
            return "NONE";
        }
        function createFunction() {
            if (!args.lambda)
                return;
            return (0, function_builder_1.functionBuilder)("".concat(name, "Function"), args.lambda, {
                description: "".concat(args.apiComponentName, " data source"),
            });
        }
        function createServiceRole() {
            var _a;
            var _b;
            if (!lambda &&
                !args.dynamodb &&
                !args.elasticSearch &&
                !args.eventBridge &&
                !args.openSearch)
                return;
            return new ((_a = aws_1.iam.Role).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.serviceRole, "".concat(name, "ServiceRole"), {
                assumeRolePolicy: aws_1.iam.getPolicyDocumentOutput({
                    statements: [
                        {
                            actions: ["sts:AssumeRole"],
                            principals: [
                                {
                                    type: "Service",
                                    identifiers: ["appsync.amazonaws.com"],
                                },
                            ],
                        },
                    ],
                }).json,
                inlinePolicies: [
                    {
                        name: "inline",
                        policy: aws_1.iam.getPolicyDocumentOutput({
                            statements: __spreadArray(__spreadArray(__spreadArray(__spreadArray(__spreadArray([], (lambda
                                ? [{ actions: ["lambda:*"], resources: [lambda.arn] }]
                                : []), true), (args.dynamodb
                                ? [
                                    {
                                        actions: ["dynamodb:*"],
                                        resources: [args.dynamodb],
                                    },
                                ]
                                : []), true), (args.elasticSearch
                                ? [
                                    {
                                        actions: ["es:*"],
                                        resources: [args.elasticSearch],
                                    },
                                ]
                                : []), true), (args.eventBridge
                                ? [
                                    {
                                        actions: ["events:*"],
                                        resources: [args.eventBridge],
                                    },
                                ]
                                : []), true), (args.openSearch
                                ? [
                                    {
                                        actions: ["opensearch:*"],
                                        resources: [args.openSearch],
                                    },
                                ]
                                : []), true),
                        }).json,
                    },
                ],
            }, { parent: self }), false)))();
        }
        function createDataSource() {
            var _a;
            var _b;
            return new ((_a = aws_1.appsync.DataSource).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.dataSource, "".concat(name, "DataSource"), {
                apiId: apiId,
                type: type,
                name: args.name,
                serviceRoleArn: serviceRole === null || serviceRole === void 0 ? void 0 : serviceRole.arn,
                lambdaConfig: lambda ? { functionArn: lambda.arn } : undefined,
                dynamodbConfig: args.dynamodb
                    ? {
                        tableName: (0, pulumi_1.output)(args.dynamodb).apply(function (arn) { return (0, arn_1.parseDynamoArn)(arn).tableName; }),
                    }
                    : undefined,
                elasticsearchConfig: args.elasticSearch
                    ? { endpoint: args.elasticSearch }
                    : undefined,
                eventBridgeConfig: args.eventBridge
                    ? { eventBusArn: args.eventBridge }
                    : undefined,
                httpConfig: args.http ? { endpoint: args.http } : undefined,
                opensearchserviceConfig: args.openSearch
                    ? { endpoint: args.openSearch }
                    : undefined,
                relationalDatabaseConfig: args.rds
                    ? {
                        httpEndpointConfig: {
                            dbClusterIdentifier: (0, pulumi_1.output)(args.rds).cluster,
                            awsSecretStoreArn: (0, pulumi_1.output)(args.rds).credentials,
                        },
                    }
                    : undefined,
            }, { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(AppSyncDataSource.prototype, "name", {
        /**
         * The name of the data source.
         */
        get: function () {
            return this.dataSource.name;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(AppSyncDataSource.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var self = this;
            return {
                /**
                 * The Amazon AppSync DataSource.
                 */
                dataSource: this.dataSource,
                /**
                 * The Lambda function used by the data source.
                 */
                get function() {
                    if (!self.lambda)
                        throw new error_1.VisibleError("Cannot access `nodes.function` because the data source does not use a Lambda function.");
                    return self.lambda.apply(function (fn) { return fn.getFunction(); });
                },
                /**
                 * The DataSource service's IAM role.
                 */
                get serviceRole() {
                    if (!self.serviceRole)
                        throw new error_1.VisibleError("Cannot access `nodes.serviceRole` because the data source does not have a service role.");
                    return self.serviceRole;
                },
            };
        },
        enumerable: false,
        configurable: true
    });
    return AppSyncDataSource;
}(component_1.Component));
exports.AppSyncDataSource = AppSyncDataSource;
var __pulumiType = "sst:aws:AppSyncDataSource";
// @ts-expect-error
AppSyncDataSource.__pulumiType = __pulumiType;
