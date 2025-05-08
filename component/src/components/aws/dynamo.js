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
exports.Dynamo = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var naming_1 = require("../naming");
var arn_1 = require("./helpers/arn");
var dynamo_lambda_subscriber_1 = require("./dynamo-lambda-subscriber");
var aws_1 = require("@pulumi/aws");
var permission_1 = require("./permission");
var subscriber_1 = require("./helpers/subscriber");
/**
 * The `Dynamo` component lets you add an [Amazon DynamoDB](https://aws.amazon.com/dynamodb/) table to your app.
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * const table = new sst.aws.Dynamo("MyTable", {
 *   fields: {
 *     userId: "string",
 *     noteId: "string"
 *   },
 *   primaryIndex: { hashKey: "userId", rangeKey: "noteId" }
 * });
 * ```
 *
 * #### Add a global index
 *
 * Optionally add a global index to the table.
 *
 * ```ts {8-10} title="sst.config.ts"
 * new sst.aws.Dynamo("MyTable", {
 *   fields: {
 *     userId: "string",
 *     noteId: "string",
 *     createdAt: "number",
 *   },
 *   primaryIndex: { hashKey: "userId", rangeKey: "noteId" },
 *   globalIndexes: {
 *     CreatedAtIndex: { hashKey: "userId", rangeKey: "createdAt" }
 *   }
 * });
 * ```
 *
 * #### Add a local index
 *
 * Optionally add a local index to the table.
 *
 * ```ts {8-10} title="sst.config.ts"
 * new sst.aws.Dynamo("MyTable", {
 *   fields: {
 *     userId: "string",
 *     noteId: "string",
 *     createdAt: "number",
 *   },
 *   primaryIndex: { hashKey: "userId", rangeKey: "noteId" },
 *   localIndexes: {
 *     CreatedAtIndex: { rangeKey: "createdAt" }
 *   }
 * });
 * ```
 *
 * #### Subscribe to a DynamoDB Stream
 *
 * To subscribe to a [DynamoDB Stream](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Streams.html), start by enabling it.
 *
 * ```ts {7} title="sst.config.ts"
 * const table = new sst.aws.Dynamo("MyTable", {
 *   fields: {
 *     userId: "string",
 *     noteId: "string"
 *   },
 *   primaryIndex: { hashKey: "userId", rangeKey: "noteId" },
 *   stream: "new-and-old-images"
 * });
 * ```
 *
 * Then, subscribing to it.
 *
 * ```ts title="sst.config.ts"
 * table.subscribe("MySubscriber", "src/subscriber.handler");
 * ```
 *
 * #### Link the table to a resource
 *
 * You can link the table to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [table]
 * });
 * ```
 *
 * Once linked, you can query the table through your app.
 *
 * ```ts title="app/page.tsx" {1,8}
 * import { Resource } from "sst";
 * import { DynamoDBClient, QueryCommand, ScanCommand } from "@aws-sdk/client-dynamodb";
 *
 * const client = new DynamoDBClient();
 *
 * await client.send(new QueryCommand({
 *   TableName: Resource.MyTable.name,
 *   KeyConditionExpression: "userId = :userId",
 *   ExpressionAttributeValues: {
 *     ":userId": "my-user-id"
 *   }
 * }));
 * ```
 */
var Dynamo = /** @class */ (function (_super) {
    __extends(Dynamo, _super);
    function Dynamo(name, args, opts) {
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        _this.isStreamEnabled = false;
        _this.constructorName = name;
        _this.constructorOpts = opts;
        if (args && "ref" in args) {
            var ref = args;
            _this.table = (0, pulumi_1.output)(ref.table);
            return _this;
        }
        var parent = _this;
        var table = createTable();
        _this.table = table;
        _this.isStreamEnabled = Boolean(args.stream);
        function createTable() {
            return (0, pulumi_1.all)([
                args.fields,
                args.primaryIndex,
                args.globalIndexes,
                args.localIndexes,
                args.stream,
                args.deletionProtection,
            ]).apply(function (_a) {
                var _b;
                var _c;
                var fields = _a[0], primaryIndex = _a[1], globalIndexes = _a[2], localIndexes = _a[3], stream = _a[4], deletionProtection = _a[5];
                return new ((_b = aws_1.dynamodb.Table).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.table, "".concat(name, "Table"), {
                    attributes: Object.entries(fields).map(function (_a) {
                        var name = _a[0], type = _a[1];
                        return ({
                            name: name,
                            type: type === "string" ? "S" : type === "number" ? "N" : "B",
                        });
                    }),
                    billingMode: "PAY_PER_REQUEST",
                    hashKey: primaryIndex.hashKey,
                    rangeKey: primaryIndex.rangeKey,
                    streamEnabled: Boolean(stream),
                    streamViewType: stream
                        ? stream.toUpperCase().replaceAll("-", "_")
                        : undefined,
                    pointInTimeRecovery: {
                        enabled: true,
                    },
                    ttl: args.ttl === undefined
                        ? undefined
                        : {
                            attributeName: args.ttl,
                            enabled: true,
                        },
                    globalSecondaryIndexes: Object.entries(globalIndexes !== null && globalIndexes !== void 0 ? globalIndexes : {}).map(function (_a) {
                        var name = _a[0], index = _a[1];
                        return (__assign({ name: name, hashKey: index.hashKey, rangeKey: index.rangeKey }, (index.projection === "keys-only"
                            ? { projectionType: "KEYS_ONLY" }
                            : Array.isArray(index.projection)
                                ? {
                                    projectionType: "INCLUDE",
                                    nonKeyAttributes: index.projection,
                                }
                                : { projectionType: "ALL" })));
                    }),
                    localSecondaryIndexes: Object.entries(localIndexes !== null && localIndexes !== void 0 ? localIndexes : {}).map(function (_a) {
                        var name = _a[0], index = _a[1];
                        return (__assign({ name: name, rangeKey: index.rangeKey }, (index.projection === "keys-only"
                            ? { projectionType: "KEYS_ONLY" }
                            : Array.isArray(index.projection)
                                ? {
                                    projectionType: "INCLUDE",
                                    nonKeyAttributes: index.projection,
                                }
                                : { projectionType: "ALL" })));
                    }),
                    deletionProtectionEnabled: deletionProtection,
                }, { parent: parent }), false)))();
            });
        }
        return _this;
    }
    Object.defineProperty(Dynamo.prototype, "arn", {
        /**
         * The ARN of the DynamoDB Table.
         */
        get: function () {
            return this.table.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Dynamo.prototype, "name", {
        /**
         * The name of the DynamoDB Table.
         */
        get: function () {
            return this.table.name;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Dynamo.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon DynamoDB Table.
                 */
                table: this.table,
            };
        },
        enumerable: false,
        configurable: true
    });
    Dynamo.prototype.subscribe = function (nameOrSubscriber, subscriberOrArgs, args) {
        var _this = this;
        var sourceName = this.constructorName;
        // Validate stream is enabled
        if (!this.isStreamEnabled)
            throw new Error("Cannot subscribe to \"".concat(sourceName, "\" because stream is not enabled."));
        return (0, subscriber_1.isFunctionSubscriber)(subscriberOrArgs).apply(function (v) {
            return v
                ? Dynamo._subscribe(nameOrSubscriber, // name
                _this.constructorName, _this.nodes.table.streamArn, subscriberOrArgs, // subscriber
                args, { provider: _this.constructorOpts.provider })
                : Dynamo._subscribeV1(_this.constructorName, _this.nodes.table.streamArn, nameOrSubscriber, // subscriber
                subscriberOrArgs, // args
                { provider: _this.constructorOpts.provider });
        });
    };
    Dynamo.subscribe = function (nameOrStreamArn, streamArnOrSubscriber, subscriberOrArgs, args) {
        var _this = this;
        return (0, subscriber_1.isFunctionSubscriber)(subscriberOrArgs).apply(function (v) {
            return v
                ? (0, pulumi_1.output)(streamArnOrSubscriber).apply(function (streamArn) {
                    return _this._subscribe(nameOrStreamArn, // name
                    (0, naming_1.logicalName)((0, arn_1.parseDynamoStreamArn)(streamArn).tableName), streamArn, subscriberOrArgs, // subscriber
                    args);
                })
                : (0, pulumi_1.output)(nameOrStreamArn).apply(function (streamArn) {
                    return _this._subscribeV1((0, naming_1.logicalName)((0, arn_1.parseDynamoStreamArn)(streamArn).tableName), streamArn, streamArnOrSubscriber, // subscriber
                    subscriberOrArgs);
                });
        });
    };
    Dynamo._subscribe = function (subscriberName, name, streamArn, subscriber, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return (0, pulumi_1.output)(args).apply(function (args) {
            return new dynamo_lambda_subscriber_1.DynamoLambdaSubscriber("".concat(name, "Subscriber").concat(subscriberName), __assign({ dynamo: { streamArn: streamArn }, subscriber: subscriber }, args), opts);
        });
    };
    Dynamo._subscribeV1 = function (name, streamArn, subscriber, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return (0, pulumi_1.all)([name, subscriber, args]).apply(function (_a) {
            var _b;
            var name = _a[0], subscriber = _a[1], args = _a[2];
            var suffix = (0, naming_1.logicalName)((0, naming_1.hashStringToPrettyString)([
                typeof streamArn === "string" ? streamArn : component_1.outputId,
                JSON.stringify((_b = args.filters) !== null && _b !== void 0 ? _b : {}),
                typeof subscriber === "string" ? subscriber : subscriber.handler,
            ].join(""), 6));
            return new dynamo_lambda_subscriber_1.DynamoLambdaSubscriber("".concat(name, "Subscriber").concat(suffix), __assign({ dynamo: { streamArn: streamArn }, subscriber: subscriber, disableParent: true }, args), opts);
        });
    };
    /**
     * Reference an existing DynamoDB Table with the given table name. This is useful when you
     * create a table in one stage and want to share it in another stage. It avoid having to
     * create a new table in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share a table across stages.
     * :::
     *
     * @param name The name of the component.
     * @param tableName The name of the DynamoDB Table.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a table in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new table, you want to share the table from `dev`.
     *
     * ```ts title=sst.config.ts"
     * const table = $app.stage === "frank"
     *  ? sst.aws.Dynamo.get("MyTable", "app-dev-mytable")
     *  : new sst.aws.Dynamo("MyTable");
     * ```
     *
     * Here `app-dev-mytable` is the name of the DynamoDB Table created in the `dev` stage.
     * You can find this by outputting the table name in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   table: table.name
     * };
     * ```
     */
    Dynamo.get = function (name, tableName, opts) {
        return new Dynamo(name, {
            ref: true,
            table: aws_1.dynamodb.Table.get("".concat(name, "Table"), tableName, undefined, opts),
        });
    };
    /** @internal */
    Dynamo.prototype.getSSTLink = function () {
        return {
            properties: {
                name: this.name,
            },
            include: [
                (0, permission_1.permission)({
                    actions: ["dynamodb:*"],
                    resources: [this.arn, (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["", "/*"], ["", "/*"])), this.arn)],
                }),
            ],
        };
    };
    return Dynamo;
}(component_1.Component));
exports.Dynamo = Dynamo;
var __pulumiType = "sst:aws:Dynamo";
// @ts-expect-error
Dynamo.__pulumiType = __pulumiType;
var templateObject_1;
