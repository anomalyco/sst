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
exports.SnsTopic = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var naming_1 = require("../naming");
var arn_1 = require("./helpers/arn");
var sns_topic_lambda_subscriber_1 = require("./sns-topic-lambda-subscriber");
var sns_topic_queue_subscriber_1 = require("./sns-topic-queue-subscriber");
var aws_1 = require("@pulumi/aws");
var permission_1 = require("./permission");
var subscriber_1 = require("./helpers/subscriber");
/**
 * The `SnsTopic` component lets you add an [Amazon SNS Topic](https://docs.aws.amazon.com/sns/latest/dg/sns-create-topic.html) to your app.
 *
 * :::note
 * The difference between an `SnsTopic` and a `Queue` is that with a topic you can deliver messages to multiple subscribers.
 * :::
 *
 * @example
 *
 * #### Create a topic
 *
 * ```ts title="sst.config.ts"
 * const topic = new sst.aws.SnsTopic("MyTopic");
 * ```
 *
 * #### Make it a FIFO topic
 *
 * You can optionally make it a FIFO topic.
 *
 * ```ts {2} title="sst.config.ts"
 * new sst.aws.SnsTopic("MyTopic", {
 *   fifo: true
 * });
 * ```
 *
 * #### Add a subscriber
 *
 * ```ts title="sst.config.ts"
 * topic.subscribe("MySubscriber", "src/subscriber.handler");
 * ```
 *
 * #### Link the topic to a resource
 *
 * You can link the topic to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [topic]
 * });
 * ```
 *
 * Once linked, you can publish messages to the topic from your function code.
 *
 * ```ts title="app/page.tsx" {1,7}
 * import { Resource } from "sst";
 * import { SNSClient, PublishCommand } from "@aws-sdk/client-sns";
 *
 * const sns = new SNSClient({});
 *
 * await sns.send(new PublishCommand({
 *   TopicArn: Resource.MyTopic.arn,
 *   Message: "Hello from Next.js!"
 * }));
 * ```
 */
var SnsTopic = /** @class */ (function (_super) {
    __extends(SnsTopic, _super);
    function SnsTopic(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        _this.constructorName = name;
        _this.constructorOpts = opts;
        if (args && "ref" in args) {
            var ref = reference();
            _this.topic = ref.topic;
            return _this;
        }
        var fifo = normalizeFifo();
        _this.topic = createTopic();
        function reference() {
            var ref = args;
            var topic = aws_1.sns.Topic.get("".concat(name, "Topic"), ref.topicArn, undefined, {
                parent: self,
            });
            return { topic: topic };
        }
        function normalizeFifo() {
            return (0, pulumi_1.output)(args.fifo).apply(function (v) { return v !== null && v !== void 0 ? v : false; });
        }
        function createTopic() {
            var _a;
            var _b;
            return new ((_a = aws_1.sns.Topic).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.topic, "".concat(name, "Topic"), {
                fifoTopic: fifo,
            }, { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(SnsTopic.prototype, "arn", {
        /**
         * The ARN of the SNS Topic.
         */
        get: function () {
            return this.topic.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(SnsTopic.prototype, "name", {
        /**
         * The name of the SNS Topic.
         */
        get: function () {
            return this.topic.name;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(SnsTopic.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon SNS Topic.
                 */
                topic: this.topic,
            };
        },
        enumerable: false,
        configurable: true
    });
    SnsTopic.prototype.subscribe = function (nameOrSubscriber, subscriberOrArgs, args) {
        var _this = this;
        return (0, subscriber_1.isFunctionSubscriber)(subscriberOrArgs).apply(function (v) {
            return v
                ? SnsTopic._subscribeFunction(nameOrSubscriber, // name
                _this.constructorName, _this.arn, subscriberOrArgs, // subscriber
                args, { provider: _this.constructorOpts.provider })
                : SnsTopic._subscribeFunctionV1(_this.constructorName, _this.arn, nameOrSubscriber, // subscriber
                subscriberOrArgs, // args
                { provider: _this.constructorOpts.provider });
        });
    };
    SnsTopic.subscribe = function (nameOrTopicArn, topicArnOrSubscriber, subscriberOrArgs, args) {
        var _this = this;
        return (0, subscriber_1.isFunctionSubscriber)(subscriberOrArgs).apply(function (v) {
            return v
                ? (0, pulumi_1.output)(topicArnOrSubscriber).apply(function (topicArn) {
                    return _this._subscribeFunction(nameOrTopicArn, // name
                    (0, naming_1.logicalName)((0, arn_1.parseTopicArn)(topicArn).topicName), topicArn, subscriberOrArgs, // subscriber
                    args);
                })
                : (0, pulumi_1.output)(nameOrTopicArn).apply(function (topicArn) {
                    return _this._subscribeFunctionV1((0, naming_1.logicalName)((0, arn_1.parseTopicArn)(topicArn).topicName), topicArn, topicArnOrSubscriber, // subscriber
                    subscriberOrArgs);
                });
        });
    };
    SnsTopic._subscribeFunction = function (subscriberName, name, topicArn, subscriber, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return (0, pulumi_1.output)(args).apply(function (args) {
            return new sns_topic_lambda_subscriber_1.SnsTopicLambdaSubscriber("".concat(name, "Subscriber").concat(subscriberName), __assign({ topic: { arn: topicArn }, subscriber: subscriber }, args), opts);
        });
    };
    SnsTopic._subscribeFunctionV1 = function (name, topicArn, subscriber, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return (0, pulumi_1.all)([subscriber, args]).apply(function (_a) {
            var _b;
            var subscriber = _a[0], args = _a[1];
            var suffix = (0, naming_1.logicalName)((0, naming_1.hashStringToPrettyString)([
                typeof topicArn === "string" ? topicArn : component_1.outputId,
                JSON.stringify((_b = args.filter) !== null && _b !== void 0 ? _b : {}),
                typeof subscriber === "string" ? subscriber : subscriber.handler,
            ].join(""), 6));
            return new sns_topic_lambda_subscriber_1.SnsTopicLambdaSubscriber("".concat(name, "Subscriber").concat(suffix), __assign({ topic: { arn: topicArn }, subscriber: subscriber }, args), opts);
        });
    };
    SnsTopic.prototype.subscribeQueue = function (nameOrQueue, queueOrArgs, args) {
        var _this = this;
        return (0, subscriber_1.isQueueSubscriber)(queueOrArgs).apply(function (v) {
            return v
                ? SnsTopic._subscribeQueue(nameOrQueue, // name
                _this.constructorName, _this.arn, queueOrArgs, // queue
                args)
                : SnsTopic._subscribeQueueV1(_this.constructorName, _this.arn, nameOrQueue, // queue
                queueOrArgs);
        });
    };
    SnsTopic.subscribeQueue = function (nameOrTopicArn, topicArnOrQueue, queueOrArgs, args) {
        var _this = this;
        return (0, subscriber_1.isQueueSubscriber)(queueOrArgs).apply(function (v) {
            return v
                ? (0, pulumi_1.output)(topicArnOrQueue).apply(function (topicArn) {
                    return _this._subscribeQueue(nameOrTopicArn, // name
                    (0, naming_1.logicalName)((0, arn_1.parseTopicArn)(topicArn).topicName), topicArn, queueOrArgs, // queue
                    args);
                })
                : (0, pulumi_1.output)(nameOrTopicArn).apply(function (topicArn) {
                    return _this._subscribeQueueV1((0, naming_1.logicalName)((0, arn_1.parseTopicArn)(topicArn).topicName), topicArn, topicArnOrQueue, // queue
                    queueOrArgs);
                });
        });
    };
    SnsTopic._subscribeQueue = function (subscriberName, name, topicArn, queue, args) {
        if (args === void 0) { args = {}; }
        return (0, pulumi_1.output)(args).apply(function (args) {
            return new sns_topic_queue_subscriber_1.SnsTopicQueueSubscriber("".concat(name, "Subscriber").concat(subscriberName), __assign({ topic: { arn: topicArn }, queue: queue }, args));
        });
    };
    SnsTopic._subscribeQueueV1 = function (name, topicArn, queueArn, args) {
        if (args === void 0) { args = {}; }
        return (0, pulumi_1.all)([queueArn, args]).apply(function (_a) {
            var _b;
            var queueArn = _a[0], args = _a[1];
            var suffix = (0, naming_1.logicalName)((0, naming_1.hashStringToPrettyString)([
                typeof topicArn === "string" ? topicArn : component_1.outputId,
                JSON.stringify((_b = args.filter) !== null && _b !== void 0 ? _b : {}),
                queueArn,
            ].join(""), 6));
            return new sns_topic_queue_subscriber_1.SnsTopicQueueSubscriber("".concat(name, "Subscriber").concat(suffix), __assign({ topic: { arn: topicArn }, queue: queueArn, disableParent: true }, args));
        });
    };
    /**
     * Reference an existing SNS topic with its topic ARN. This is useful when you create a
     * topic in one stage and want to share it in another stage. It avoids having to create
     * a new topic in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share SNS topics across stages.
     * :::
     *
     * @param name The name of the component.
     * @param topicArn The ARN of the existing SNS Topic.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a topic in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new topic, you want to share the topic from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const topic = $app.stage === "frank"
     *   ? sst.aws.SnsTopic.get("MyTopic", "arn:aws:sns:us-east-1:123456789012:MyTopic")
     *   : new sst.aws.SnsTopic("MyTopic");
     * ```
     *
     * Here `arn:aws:sns:us-east-1:123456789012:MyTopic` is the ARN of the topic created in
     * the `dev` stage. You can find this by outputting the topic ARN in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return topic.arn;
     * ```
     */
    SnsTopic.get = function (name, topicArn, opts) {
        return new SnsTopic(name, {
            ref: true,
            topicArn: topicArn,
        }, opts);
    };
    /** @internal */
    SnsTopic.prototype.getSSTLink = function () {
        return {
            properties: {
                arn: this.arn,
            },
            include: [
                (0, permission_1.permission)({
                    actions: ["sns:*"],
                    resources: [this.arn],
                }),
            ],
        };
    };
    return SnsTopic;
}(component_1.Component));
exports.SnsTopic = SnsTopic;
var __pulumiType = "sst:aws:SnsTopic";
// @ts-expect-error
SnsTopic.__pulumiType = __pulumiType;
