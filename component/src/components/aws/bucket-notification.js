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
exports.BucketNotification = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var function_builder_1 = require("./helpers/function-builder");
var error_1 = require("../error");
var sns_topic_1 = require("./sns-topic");
var queue_1 = require("./queue");
/**
 * The `BucketNotification` component is internally used by the `Bucket` component to
 * add bucket notifications to [AWS S3 Bucket](https://aws.amazon.com/s3/).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `notify` method of the `Bucket` component.
 */
var BucketNotification = /** @class */ (function (_super) {
    __extends(BucketNotification, _super);
    function BucketNotification(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var bucket = (0, pulumi_1.output)(args.bucket);
        var notifications = normalizeNotifications();
        var _a = createNotificationsConfig(), config = _a.config, functionBuilders = _a.functionBuilders;
        var notification = createNotification();
        _this.functionBuilders = functionBuilders;
        _this.notification = notification;
        function normalizeNotifications() {
            return (0, pulumi_1.output)(args.notifications).apply(function (notifications) {
                return notifications.map(function (n) {
                    var _a;
                    var count = (n.function ? 1 : 0) + (n.queue ? 1 : 0) + (n.topic ? 1 : 0);
                    if (count === 0)
                        throw new error_1.VisibleError("At least one of function, queue, or topic is required for the \"".concat(n.name, "\" bucket notification."));
                    if (count > 1)
                        throw new error_1.VisibleError("Only one of function, queue, or topic is allowed for the \"".concat(n.name, "\" bucket notification."));
                    return __assign(__assign({}, n), { events: (_a = n.events) !== null && _a !== void 0 ? _a : [
                            "s3:ObjectCreated:*",
                            "s3:ObjectRemoved:*",
                            "s3:ObjectRestore:*",
                            "s3:ReducedRedundancyLostObject",
                            "s3:Replication:*",
                            "s3:LifecycleExpiration:*",
                            "s3:LifecycleTransition",
                            "s3:IntelligentTiering",
                            "s3:ObjectTagging:*",
                            "s3:ObjectAcl:Put",
                        ] });
                });
            });
        }
        function createNotificationsConfig() {
            return notifications.apply(function (notifications) {
                var config = notifications.map(function (n) {
                    if (n.function) {
                        var fn = (0, function_builder_1.functionBuilder)("".concat(name, "Notification").concat(n.name), n.function, {
                            description: n.events.length < 5
                                ? "Notified by ".concat(name, " on ").concat(n.events.join(", "))
                                : "Notified by ".concat(name, " on ").concat(n.events
                                    .slice(0, 3)
                                    .join(", "), ", and ").concat(n.events.length - 3, " more events"),
                        }, undefined, { parent: self });
                        var permission = new aws_1.lambda.Permission("".concat(name, "Notification").concat(n.name, "Permission"), {
                            action: "lambda:InvokeFunction",
                            function: fn.arn,
                            principal: "s3.amazonaws.com",
                            sourceArn: bucket.arn,
                        }, { parent: self });
                        return { args: n, functionBuilder: fn, dependsOn: permission };
                    }
                    if (n.topic) {
                        var arn = n.topic instanceof sns_topic_1.SnsTopic ? n.topic.arn : (0, pulumi_1.output)(n.topic);
                        var policy = new aws_1.sns.TopicPolicy("".concat(name, "Notification").concat(n.name, "Policy"), {
                            arn: arn,
                            policy: aws_1.iam.getPolicyDocumentOutput({
                                statements: [
                                    {
                                        actions: ["sns:Publish"],
                                        resources: [arn],
                                        principals: [
                                            {
                                                type: "Service",
                                                identifiers: ["s3.amazonaws.com"],
                                            },
                                        ],
                                        conditions: [
                                            {
                                                test: "ArnEquals",
                                                variable: "aws:SourceArn",
                                                values: [bucket.arn],
                                            },
                                        ],
                                    },
                                ],
                            }).json,
                        }, { parent: self });
                        return { args: n, topicArn: arn, dependsOn: policy };
                    }
                    if (n.queue) {
                        var arn = n.queue instanceof queue_1.Queue ? n.queue.arn : (0, pulumi_1.output)(n.queue);
                        var policy = queue_1.Queue.createPolicy("".concat(name, "Notification").concat(n.name, "Policy"), arn, { parent: self });
                        return { args: n, queueArn: arn, dependsOn: policy };
                    }
                });
                return {
                    config: config,
                    functionBuilders: config
                        .filter(function (c) { return c.functionBuilder; })
                        .map(function (c) { return c.functionBuilder; }),
                };
            });
        }
        function createNotification() {
            var _a;
            var _b;
            return new ((_a = aws_1.s3.BucketNotification).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.notification, "".concat(name, "Notification"), {
                bucket: bucket.name,
                lambdaFunctions: config.apply(function (config) {
                    return config
                        .filter(function (c) { return c.functionBuilder; })
                        .map(function (c) { return ({
                        id: c.args.name,
                        lambdaFunctionArn: c.functionBuilder.arn,
                        events: c.args.events,
                        filterPrefix: c.args.filterPrefix,
                        filterSuffix: c.args.filterSuffix,
                    }); });
                }),
                queues: config.apply(function (config) {
                    return config
                        .filter(function (c) { return c.queueArn; })
                        .map(function (c) { return ({
                        id: c.args.name,
                        queueArn: c.queueArn,
                        events: c.args.events,
                        filterPrefix: c.args.filterPrefix,
                        filterSuffix: c.args.filterSuffix,
                    }); });
                }),
                topics: config.apply(function (config) {
                    return config
                        .filter(function (c) { return c.topicArn; })
                        .map(function (c) { return ({
                        id: c.args.name,
                        topicArn: c.topicArn,
                        events: c.args.events,
                        filterPrefix: c.args.filterPrefix,
                        filterSuffix: c.args.filterSuffix,
                    }); });
                }),
            }, {
                parent: self,
                dependsOn: config.apply(function (config) {
                    return config.map(function (c) { return c.dependsOn; });
                }),
            }), false)))();
        }
        return _this;
    }
    Object.defineProperty(BucketNotification.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var self = this;
            return {
                /**
                 * The functions that will be notified.
                 */
                get functions() {
                    return (0, pulumi_1.output)(self.functionBuilders).apply(function (functionBuilders) {
                        return functionBuilders.map(function (builder) { return builder.getFunction(); });
                    });
                },
                /**
                 * The notification resource that's created.
                 */
                notification: this.notification,
            };
        },
        enumerable: false,
        configurable: true
    });
    return BucketNotification;
}(component_1.Component));
exports.BucketNotification = BucketNotification;
var __pulumiType = "sst:aws:BucketNotification";
// @ts-expect-error
BucketNotification.__pulumiType = __pulumiType;
