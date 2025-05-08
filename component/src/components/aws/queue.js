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
exports.Queue = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var error_1 = require("../error");
var naming_1 = require("../naming");
var arn_1 = require("./helpers/arn");
var queue_lambda_subscriber_1 = require("./queue-lambda-subscriber");
var aws_1 = require("@pulumi/aws");
var duration_1 = require("../duration");
var permission_js_1 = require("./permission.js");
/**
 * The `Queue` component lets you add a serverless queue to your app. It uses [Amazon SQS](https://aws.amazon.com/sqs/).
 *
 * @example
 *
 * #### Create a queue
 *
 * ```ts title="sst.config.ts"
 * const queue = new sst.aws.Queue("MyQueue");
 * ```
 *
 * #### Make it a FIFO queue
 *
 * You can optionally make it a FIFO queue.
 *
 * ```ts {2} title="sst.config.ts"
 * new sst.aws.Queue("MyQueue", {
 *   fifo: true
 * });
 * ```
 *
 * #### Add a subscriber
 *
 * ```ts title="sst.config.ts"
 * queue.subscribe("src/subscriber.handler");
 * ```
 *
 * #### Link the queue to a resource
 *
 * You can link the queue to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [queue]
 * });
 * ```
 *
 * Once linked, you can send messages to the queue from your function code.
 *
 * ```ts title="app/page.tsx" {1,7}
 * import { Resource } from "sst";
 * import { SQSClient, SendMessageCommand } from "@aws-sdk/client-sqs";
 *
 * const sqs = new SQSClient({});
 *
 * await sqs.send(new SendMessageCommand({
 *   QueueUrl: Resource.MyQueue.url,
 *   MessageBody: "Hello from Next.js!"
 * }));
 * ```
 */
var Queue = /** @class */ (function (_super) {
    __extends(Queue, _super);
    function Queue(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _a, _b;
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        _this.isSubscribed = false;
        var self = _this;
        _this.constructorName = name;
        _this.constructorOpts = opts;
        if (args && "ref" in args) {
            var ref = reference();
            _this.queue = ref.queue;
            return _this;
        }
        var fifo = normalizeFifo();
        var dlq = normalizeDlq();
        var visibilityTimeout = (0, pulumi_1.output)((_a = args === null || args === void 0 ? void 0 : args.visibilityTimeout) !== null && _a !== void 0 ? _a : "30 seconds");
        var delay = (0, pulumi_1.output)((_b = args === null || args === void 0 ? void 0 : args.delay) !== null && _b !== void 0 ? _b : "0 seconds");
        _this.queue = createQueue();
        function reference() {
            var ref = args;
            var queue = aws_1.sqs.Queue.get("".concat(name, "Queue"), ref.queueUrl, undefined, {
                parent: self,
            });
            return { queue: queue };
        }
        function normalizeFifo() {
            return (0, pulumi_1.output)(args === null || args === void 0 ? void 0 : args.fifo).apply(function (v) {
                var _a;
                if (!v)
                    return false;
                if (v === true)
                    return {
                        contentBasedDeduplication: false,
                    };
                return {
                    contentBasedDeduplication: (_a = v.contentBasedDeduplication) !== null && _a !== void 0 ? _a : false,
                };
            });
        }
        function normalizeDlq() {
            if ((args === null || args === void 0 ? void 0 : args.dlq) === undefined)
                return;
            return (0, pulumi_1.output)(args === null || args === void 0 ? void 0 : args.dlq).apply(function (v) {
                return typeof v === "string" ? { queue: v, retry: 3 } : v;
            });
        }
        function createQueue() {
            var _a;
            var _b;
            return new ((_a = aws_1.sqs.Queue).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.queue, "".concat(name, "Queue"), {
                fifoQueue: fifo.apply(function (v) { return v !== false; }),
                contentBasedDeduplication: fifo.apply(function (v) {
                    return v === false ? false : v.contentBasedDeduplication;
                }),
                visibilityTimeoutSeconds: visibilityTimeout.apply(function (v) {
                    return (0, duration_1.toSeconds)(v);
                }),
                delaySeconds: delay.apply(function (v) { return (0, duration_1.toSeconds)(v); }),
                redrivePolicy: dlq &&
                    (0, pulumi_1.jsonStringify)({
                        deadLetterTargetArn: dlq.queue,
                        maxReceiveCount: dlq.retry,
                    }),
            }, { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(Queue.prototype, "arn", {
        /**
         * The ARN of the SQS Queue.
         */
        get: function () {
            return this.queue.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Queue.prototype, "url", {
        /**
         * The SQS Queue URL.
         */
        get: function () {
            return this.queue.url;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Queue.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon SQS Queue.
                 */
                queue: this.queue,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Subscribe to this queue.
     *
     * @param subscriber The function that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * ```js title="sst.config.ts"
     * queue.subscribe("src/subscriber.handler");
     * ```
     *
     * Add a filter to the subscription.
     *
     * ```js title="sst.config.ts"
     * queue.subscribe("src/subscriber.handler", {
     *   filters: [
     *     {
     *       body: {
     *         RequestCode: ["BBBB"]
     *       }
     *     }
     *   ]
     * });
     * ```
     *
     * Customize the subscriber function.
     *
     * ```js title="sst.config.ts"
     * queue.subscribe({
     *   handler: "src/subscriber.handler",
     *   timeout: "60 seconds"
     * });
     * ```
     *
     * Or pass in the ARN of an existing Lambda function.
     *
     * ```js title="sst.config.ts"
     * queue.subscribe("arn:aws:lambda:us-east-1:123456789012:function:my-function");
     * ```
     */
    Queue.prototype.subscribe = function (subscriber, args, opts) {
        if (this.isSubscribed)
            throw new error_1.VisibleError("Cannot subscribe to the \"".concat(this.constructorName, "\" queue multiple times. An SQS Queue can only have one subscriber."));
        this.isSubscribed = true;
        return Queue._subscribeFunction(this.constructorName, this.arn, subscriber, args, __assign(__assign({}, opts), { provider: this.constructorOpts.provider }));
    };
    /**
     * Subscribe to an SQS Queue that was not created in your app.
     *
     * @param queueArn The ARN of the SQS Queue to subscribe to.
     * @param subscriber The function that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * For example, let's say you have an existing SQS Queue with the following ARN.
     *
     * ```js title="sst.config.ts"
     * const queueArn = "arn:aws:sqs:us-east-1:123456789012:MyQueue";
     * ```
     *
     * You can subscribe to it by passing in the ARN.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Queue.subscribe(queueArn, "src/subscriber.handler");
     * ```
     *
     * Add a filter to the subscription.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Queue.subscribe(queueArn, "src/subscriber.handler", {
     *   filters: [
     *     {
     *       body: {
     *         RequestCode: ["BBBB"]
     *       }
     *     }
     *   ]
     * });
     * ```
     *
     * Customize the subscriber function.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Queue.subscribe(queueArn, {
     *   handler: "src/subscriber.handler",
     *   timeout: "60 seconds"
     * });
     * ```
     */
    Queue.subscribe = function (queueArn, subscriber, args, opts) {
        var _this = this;
        return (0, pulumi_1.output)(queueArn).apply(function (queueArn) {
            return _this._subscribeFunction((0, naming_1.logicalName)((0, arn_1.parseQueueArn)(queueArn).queueName), queueArn, subscriber, args, opts);
        });
    };
    Queue._subscribeFunction = function (name, queueArn, subscriber, args, opts) {
        if (args === void 0) { args = {}; }
        return (0, pulumi_1.output)(queueArn).apply(function (queueArn) {
            var suffix = (0, naming_1.logicalName)((0, naming_1.hashStringToPrettyString)(queueArn, 6));
            return new queue_lambda_subscriber_1.QueueLambdaSubscriber("".concat(name, "Subscriber").concat(suffix), __assign({ queue: { arn: queueArn }, subscriber: subscriber }, args), opts);
        });
    };
    /**
     * Reference an existing SQS Queue with its queue URL. This is useful when you create a
     * queue in one stage and want to share it in another stage. It avoids having to create
     * a new queue in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share SQS queues across stages.
     * :::
     *
     * @param name The name of the component.
     * @param queueUrl The URL of the existing SQS Queue.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a queue in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new queue, you want to share the queue from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const queue = $app.stage === "frank"
     *   ? sst.aws.Queue.get("MyQueue", "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue")
     *   : new sst.aws.Queue("MyQueue");
     * ```
     *
     * Here `https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue` is the URL of the queue
     * created in the `dev` stage. You can find this by outputting the queue URL in the `dev`
     * stage.
     *
     * ```ts title="sst.config.ts"
     * return queue.url;
     * ```
     */
    Queue.get = function (name, queueUrl, opts) {
        return new Queue(name, {
            ref: true,
            queueUrl: queueUrl,
        }, opts);
    };
    /** @internal */
    Queue.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
            },
            include: [
                (0, permission_js_1.permission)({
                    actions: ["sqs:*"],
                    resources: [this.arn],
                }),
            ],
        };
    };
    /** @internal */
    Queue.createPolicy = function (name, arn, opts) {
        return new aws_1.sqs.QueuePolicy(name, {
            queueUrl: arn.apply(function (arn) { return (0, arn_1.parseQueueArn)(arn).queueUrl; }),
            policy: aws_1.iam.getPolicyDocumentOutput({
                statements: [
                    {
                        actions: ["sqs:SendMessage"],
                        resources: [arn],
                        principals: [
                            {
                                type: "Service",
                                identifiers: [
                                    "sns.amazonaws.com",
                                    "s3.amazonaws.com",
                                    "events.amazonaws.com",
                                ],
                            },
                        ],
                    },
                ],
            }).json,
        }, __assign({ retainOnDelete: true }, opts));
    };
    return Queue;
}(component_1.Component));
exports.Queue = Queue;
var __pulumiType = "sst:aws:Queue";
// @ts-expect-error
Queue.__pulumiType = __pulumiType;
