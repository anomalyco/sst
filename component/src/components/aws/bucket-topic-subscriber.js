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
exports.BucketTopicSubscriber = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
/**
 * The `BucketTopicSubscriber` component is internally used by the `Bucket` component
 * to add subscriptions to your [AWS S3 Bucket](https://aws.amazon.com/s3/).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribeTopic` method of the `Bucket` component.
 */
var BucketTopicSubscriber = /** @class */ (function (_super) {
    __extends(BucketTopicSubscriber, _super);
    function BucketTopicSubscriber(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var topicArn = (0, pulumi_1.output)(args.topic);
        var bucket = (0, pulumi_1.output)(args.bucket);
        var events = args.events
            ? (0, pulumi_1.output)(args.events)
            : (0, pulumi_1.output)([
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
            ]);
        var policy = createPolicy();
        var notification = createNotification();
        _this.policy = policy;
        _this.notification = notification;
        function createPolicy() {
            return new aws_1.sns.TopicPolicy("".concat(name, "Policy"), {
                arn: topicArn,
                policy: aws_1.iam.getPolicyDocumentOutput({
                    statements: [
                        {
                            actions: ["sns:Publish"],
                            resources: [topicArn],
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
            });
        }
        function createNotification() {
            var _a;
            var _b;
            return new ((_a = aws_1.s3.BucketNotification).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.notification, "".concat(name, "Notification"), {
                bucket: bucket.name,
                topics: [
                    {
                        id: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["Notification", ""], ["Notification", ""])), args.subscriberId),
                        topicArn: topicArn,
                        events: events,
                        filterPrefix: args.filterPrefix,
                        filterSuffix: args.filterSuffix,
                    },
                ],
            }, { parent: self, dependsOn: [policy] }), false)))();
        }
        return _this;
    }
    Object.defineProperty(BucketTopicSubscriber.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The SNS Topic policy.
                 */
                policy: this.policy,
                /**
                 * The S3 Bucket notification.
                 */
                notification: this.notification,
            };
        },
        enumerable: false,
        configurable: true
    });
    return BucketTopicSubscriber;
}(component_1.Component));
exports.BucketTopicSubscriber = BucketTopicSubscriber;
var __pulumiType = "sst:aws:BucketTopicSubscriber";
// @ts-expect-error
BucketTopicSubscriber.__pulumiType = __pulumiType;
var templateObject_1;
