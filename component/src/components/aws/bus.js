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
exports.Bus = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var arn_1 = require("./helpers/arn");
var bus_lambda_subscriber_1 = require("./bus-lambda-subscriber");
var aws_1 = require("@pulumi/aws");
var permission_1 = require("./permission");
var bus_queue_subscriber_1 = require("./bus-queue-subscriber");
/**
 * The `Bus` component lets you add an [Amazon EventBridge Event Bus](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-event-bus.html) to your app.
 *
 * @example
 *
 * #### Create a bus
 *
 * ```ts
 * const bus = new sst.aws.Bus("MyBus");
 * ```
 *
 * #### Add a subscriber
 *
 * ```ts
 * bus.subscribe("MySubscriber", "src/subscriber.handler");
 * ```
 *
 * #### Customize the subscriber
 *
 * ```ts
 * bus.subscribe("MySubscriber", {
 *   handler: "src/subscriber.handler",
 *   timeout: "60 seconds"
 * });
 * ```
 *
 * #### Link the bus to a resource
 *
 * You can link the bus to other resources, like a function or your Next.js app.
 *
 * ```ts
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [bus]
 * });
 * ```
 *
 * Once linked, you can publish messages to the bus from your app.
 *
 * ```ts title="app/page.tsx" {1,9}
 * import { Resource } from "sst";
 * import { EventBridgeClient, PutEventsCommand } from "@aws-sdk/client-eventbridge";
 *
 * const eb = new EventBridgeClient({});
 *
 * await eb.send(new PutEventsCommand({
 *   Entries: [
 *     {
 *       EventBusName: Resource.MyBus.name,
 *       Source: "my.source",
 *       Detail: JSON.stringify({ foo: "bar" })
 *     }
 *   ]
 * }));
 * ```
 */
var Bus = /** @class */ (function (_super) {
    __extends(Bus, _super);
    function Bus(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        _this.constructorName = name;
        _this.constructorOpts = opts;
        if (args && "ref" in args) {
            var ref = reference();
            _this.bus = ref.bus;
            return _this;
        }
        var bus = createBus();
        _this.bus = bus;
        function reference() {
            var ref = args;
            var bus = aws_1.cloudwatch.EventBus.get("".concat(name, "Bus"), ref.busName, undefined, {
                parent: self,
            });
            return { bus: bus };
        }
        function createBus() {
            var _a;
            var _b;
            return new ((_a = aws_1.cloudwatch.EventBus).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.bus, "".concat(name, "Bus"), {}, { parent: self }), false)))();
        }
        return _this;
    }
    Object.defineProperty(Bus.prototype, "arn", {
        /**
         * The ARN of the EventBus.
         */
        get: function () {
            return this.bus.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Bus.prototype, "name", {
        /**
         * The name of the EventBus.
         */
        get: function () {
            return this.bus.name;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Bus.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon EventBus resource.
                 */
                bus: this.bus,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Subscribe to this EventBus with a function.
     *
     * @param name The name of the subscription.
     * @param subscriber The function that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * ```js title="sst.config.ts"
     * bus.subscribe("MySubscription", "src/subscriber.handler");
     * ```
     *
     * You can add a pattern to the subscription.
     *
     * ```js
     * bus.subscribe("MySubscription", "src/subscriber.handler", {
     *   pattern: {
     *     source: ["my.source", "my.source2"],
     *     price_usd: [{numeric: [">=", 100]}]
     *   }
     * });
     * ```
     *
     * To customize the subscriber function:
     *
     * ```js
     * bus.subscribe("MySubscription", {
     *   handler: "src/subscriber.handler",
     *   timeout: "60 seconds"
     * });
     * ```
     *
     * Or pass in the ARN of an existing Lambda function.
     *
     * ```js title="sst.config.ts"
     * bus.subscribe("MySubscription", "arn:aws:lambda:us-east-1:123456789012:function:my-function");
     * ```
     */
    Bus.prototype.subscribe = function (name, subscriber, args) {
        if (args === void 0) { args = {}; }
        return Bus._subscribeFunction(this.constructorName, name, this.nodes.bus.name, this.nodes.bus.arn, subscriber, args, { provider: this.constructorOpts.provider });
    };
    /**
     * Subscribe to an EventBus that was not created in your app with a function.
     *
     * @param name The name of the subscription.
     * @param busArn The ARN of the EventBus to subscribe to.
     * @param subscriber The function that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * For example, let's say you have an existing EventBus with the following ARN.
     *
     * ```js title="sst.config.ts"
     * const busArn = "arn:aws:events:us-east-1:123456789012:event-bus/my-bus";
     * ```
     *
     * You can subscribe to it by passing in the ARN.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Bus.subscribe("MySubscription", busArn, "src/subscriber.handler");
     * ```
     *
     * To add a pattern to the subscription.
     *
     * ```js
     * sst.aws.Bus.subscribe("MySubscription", busArn, "src/subscriber.handler", {
     *   pattern: {
     *     price_usd: [{numeric: [">=", 100]}]
     *   }
     * });
     * ```
     *
     * Or customize the subscriber function.
     *
     * ```js
     * sst.aws.Bus.subscribe("MySubscription", busArn, {
     *   handler: "src/subscriber.handler",
     *   timeout: "60 seconds"
     * });
     * ```
     */
    Bus.subscribe = function (name, busArn, subscriber, args) {
        var _this = this;
        return (0, pulumi_1.output)(busArn).apply(function (busArn) {
            var busName = (0, arn_1.parseEventBusArn)(busArn).busName;
            return _this._subscribeFunction(busName, name, busName, busArn, subscriber, args);
        });
    };
    Bus._subscribeFunction = function (name, subscriberName, busName, busArn, subscriber, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return (0, pulumi_1.output)(args).apply(function (args) {
            return new bus_lambda_subscriber_1.BusLambdaSubscriber("".concat(name, "Subscriber").concat(subscriberName), __assign({ bus: { name: busName, arn: busArn }, subscriber: subscriber }, args), opts);
        });
    };
    /**
     * Subscribe to this EventBus with an SQS Queue.
     *
     * @param name The name of the subscription.
     * @param queue The queue that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * For example, let's say you have a queue.
     *
     * ```js title="sst.config.ts"
     * const queue = new sst.aws.Queue("MyQueue");
     * ```
     *
     * You can subscribe to this bus with it.
     *
     * ```js title="sst.config.ts"
     * bus.subscribeQueue("MySubscription", queue);
     * ```
     *
     * You can also add a filter to the subscription.
     *
     * ```js
     * bus.subscribeQueue("MySubscription", queue, {
     *   filter: {
     *     price_usd: [{numeric: [">=", 100]}]
     *   }
     * });
     * ```
     *
     * Or pass in the ARN of an existing SQS queue.
     *
     * ```js
     * bus.subscribeQueue("MySubscription", "arn:aws:sqs:us-east-1:123456789012:my-queue");
     * ```
     */
    Bus.prototype.subscribeQueue = function (name, queue, args) {
        if (args === void 0) { args = {}; }
        return Bus._subscribeQueue(this.constructorName, name, this.nodes.bus.arn, this.nodes.bus.name, queue, args);
    };
    /**
     * Subscribe to an existing EventBus with an SQS Queue.
     *
     * @param name The name of the subscription.
     * @param busArn The ARN of the EventBus to subscribe to.
     * @param queue The queue that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * For example, let's say you have an existing EventBus and an SQS Queue.
     *
     * ```js title="sst.config.ts"
     * const busArn = "arn:aws:events:us-east-1:123456789012:event-bus/MyBus";
     * const queue = new sst.aws.Queue("MyQueue");
     * ```
     *
     * You can subscribe to the bus with the queue.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Bus.subscribeQueue("MySubscription", busArn, queue);
     * ```
     *
     * Add a filter to the subscription.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Bus.subscribeQueue(MySubscription, busArn, queue, {
     *   filter: {
     *     price_usd: [{numeric: [">=", 100]}]
     *   }
     * });
     * ```
     *
     * Or pass in the ARN of an existing SQS queue.
     *
     * ```js
     * sst.aws.Bus.subscribeQueue("MySubscription", busArn, "arn:aws:sqs:us-east-1:123456789012:my-queue");
     * ```
     */
    Bus.subscribeQueue = function (name, busArn, queue, args) {
        var _this = this;
        return (0, pulumi_1.output)(busArn).apply(function (busArn) {
            var busName = (0, arn_1.parseEventBusArn)(busArn).busName;
            return _this._subscribeQueue(busName, name, busArn, busName, queue, args);
        });
    };
    Bus._subscribeQueue = function (name, subscriberName, busArn, busName, queue, args) {
        if (args === void 0) { args = {}; }
        return (0, pulumi_1.output)(args).apply(function (args) {
            return new bus_queue_subscriber_1.BusQueueSubscriber("".concat(name, "Subscriber").concat(subscriberName), __assign({ bus: { name: busName, arn: busArn }, queue: queue }, args));
        });
    };
    /** @internal */
    Bus.prototype.getSSTLink = function () {
        return {
            properties: {
                name: this.name,
                arn: this.arn,
            },
            include: [
                (0, permission_1.permission)({
                    actions: ["events:*"],
                    resources: [this.nodes.bus.arn],
                }),
            ],
        };
    };
    /**
     * Reference an existing EventBus with its ARN. This is useful when you create a
     * bus in one stage and want to share it in another stage. It avoids having to create
     * a new bus in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share EventBus across stages.
     * :::
     *
     * @param name The name of the component.
     * @param busName The name of the existing EventBus.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a bus in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new bus, you want to share the bus from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const bus = $app.stage === "frank"
     *   ? sst.aws.Bus.get("MyBus", "app-dev-MyBus")
     *   : new sst.aws.Bus("MyBus");
     * ```
     *
     * Here `app-dev-MyBus` is the name of the bus created in the `dev` stage. You can find
     * this by outputting the bus name in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return bus.name;
     * ```
     */
    Bus.get = function (name, busName, opts) {
        return new Bus(name, {
            ref: true,
            busName: busName,
        }, opts);
    };
    return Bus;
}(component_1.Component));
exports.Bus = Bus;
var __pulumiType = "sst:aws:Bus";
// @ts-expect-error
Bus.__pulumiType = __pulumiType;
