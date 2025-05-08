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
exports.Bucket = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var naming_1 = require("../naming");
var component_1 = require("../component");
var duration_1 = require("../duration");
var error_1 = require("../error");
var arn_1 = require("./helpers/arn");
var bucket_lambda_subscriber_1 = require("./bucket-lambda-subscriber");
var aws_1 = require("@pulumi/aws");
var permission_1 = require("./permission");
var bucket_queue_subscriber_1 = require("./bucket-queue-subscriber");
var bucket_topic_subscriber_1 = require("./bucket-topic-subscriber");
var bucket_notification_1 = require("./bucket-notification");
/**
 * The `Bucket` component lets you add an [AWS S3 Bucket](https://aws.amazon.com/s3/) to
 * your app.
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 * ```
 *
 * #### Public read access
 *
 * Enable `public` read access for all the files in the bucket. Useful for hosting public files.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Bucket("MyBucket", {
 *   access: "public"
 * });
 * ```
 *
 * #### Add a subscriber
 *
 * ```ts title="sst.config.ts"
 * bucket.notify({
 *   notifications: [
 *     {
 *       name: "MySubscriber",
 *       function: "src/subscriber.handler"
 *     }
 *   ]
 * });
 * ```
 *
 * #### Link the bucket to a resource
 *
 * You can link the bucket to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * Once linked, you can generate a pre-signed URL to upload files in your app.
 *
 * ```ts title="app/page.tsx" {1,7}
 * import { Resource } from "sst";
 * import { getSignedUrl } from "@aws-sdk/s3-request-presigner";
 * import { S3Client, PutObjectCommand } from "@aws-sdk/client-s3";
 *
 * const command = new PutObjectCommand({
 *    Key: "file.txt",
 *    Bucket: Resource.MyBucket.name
 *  });
 *  await getSignedUrl(new S3Client({}), command);
 * ```
 */
var Bucket = /** @class */ (function (_super) {
    __extends(Bucket, _super);
    function Bucket(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _a;
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        _this.isSubscribed = false;
        _this.constructorName = name;
        _this.constructorOpts = opts;
        if (args && "ref" in args) {
            var ref = args;
            _this.bucket = (0, pulumi_1.output)(ref.bucket);
            return _this;
        }
        var parent = _this;
        var access = normalizeAccess();
        var enforceHttps = (0, pulumi_1.output)((_a = args.enforceHttps) !== null && _a !== void 0 ? _a : true);
        var policyArgs = normalizePolicy();
        var bucket = createBucket();
        createVersioning();
        var publicAccessBlock = createPublicAccess();
        var policy = createBucketPolicy();
        createCorsRule();
        // Ensure the policy is created when the bucket is used in another component
        // (ie. bucket.name). Also, a bucket can only have one policy. We want to ensure
        // the policy created here is created first. And SST will throw an error if
        // another policy is created after this one.
        _this.bucket = policy.apply(function () { return bucket; });
        function normalizeAccess() {
            return (0, pulumi_1.all)([args.public, args.access]).apply(function (_a) {
                var pub = _a[0], access = _a[1];
                return pub === true ? "public" : access;
            });
        }
        function normalizePolicy() {
            var _a;
            return (0, pulumi_1.output)((_a = args.policy) !== null && _a !== void 0 ? _a : []).apply(function (policy) {
                return policy.map(function (p) { return (__assign(__assign({}, p), { effect: p.effect && p.effect.charAt(0).toUpperCase() + p.effect.slice(1), principals: p.principals === "*"
                        ? [{ type: "*", identifiers: ["*"] }]
                        : p.principals.map(function (i) { return (__assign(__assign({}, i), { type: {
                                aws: "AWS",
                                service: "Service",
                                federated: "Federated",
                                canonical: "Canonical",
                            }[i.type] })); }) })); });
            });
        }
        function createBucket() {
            var _a;
            var _b;
            return new ((_a = aws_1.s3.BucketV2).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.bucket, "".concat(name, "Bucket"), {
                forceDestroy: true,
            }, { parent: parent }), false)))();
        }
        function createVersioning() {
            return (0, pulumi_1.output)(args.versioning).apply(function (versioning) {
                var _a;
                var _b;
                if (!versioning)
                    return;
                return new ((_a = aws_1.s3.BucketVersioningV2).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.versioning, "".concat(name, "Versioning"), {
                    bucket: bucket.bucket,
                    versioningConfiguration: {
                        status: "Enabled",
                    },
                }, { parent: parent }), false)))();
            });
        }
        function createPublicAccess() {
            var _a;
            var _b, _c;
            if (((_b = args.transform) === null || _b === void 0 ? void 0 : _b.publicAccessBlock) === false)
                return;
            return new ((_a = aws_1.s3.BucketPublicAccessBlock).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.publicAccessBlock, "".concat(name, "PublicAccessBlock"), {
                bucket: bucket.bucket,
                blockPublicAcls: true,
                blockPublicPolicy: access.apply(function (v) { return v !== "public"; }),
                ignorePublicAcls: true,
                restrictPublicBuckets: access.apply(function (v) { return v !== "public"; }),
            }, { parent: parent }), false)))();
        }
        function createBucketPolicy() {
            return (0, pulumi_1.all)([access, enforceHttps, policyArgs]).apply(function (_a) {
                var _b;
                var _c;
                var access = _a[0], enforceHttps = _a[1], policyArgs = _a[2];
                var statements = [];
                if (access) {
                    statements.push({
                        principals: [
                            access === "public"
                                ? { type: "*", identifiers: ["*"] }
                                : {
                                    type: "Service",
                                    identifiers: ["cloudfront.amazonaws.com"],
                                },
                        ],
                        actions: ["s3:GetObject"],
                        resources: [(0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["", "/*"], ["", "/*"])), bucket.arn)],
                    });
                }
                if (enforceHttps) {
                    statements.push({
                        effect: "Deny",
                        principals: [{ type: "*", identifiers: ["*"] }],
                        actions: ["s3:*"],
                        resources: [bucket.arn, (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["", "/*"], ["", "/*"])), bucket.arn)],
                        conditions: [
                            {
                                test: "Bool",
                                variable: "aws:SecureTransport",
                                values: ["false"],
                            },
                        ],
                    });
                }
                statements.push.apply(statements, policyArgs.map(function (p) { return (__assign(__assign({}, p), { resources: [bucket.arn, (0, pulumi_1.interpolate)(templateObject_3 || (templateObject_3 = __makeTemplateObject(["", "/*"], ["", "/*"])), bucket.arn)] })); }));
                return new ((_b = aws_1.s3.BucketPolicy).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.policy, "".concat(name, "Policy"), {
                    bucket: bucket.bucket,
                    policy: aws_1.iam.getPolicyDocumentOutput({ statements: statements }).json,
                }, {
                    parent: parent,
                    dependsOn: publicAccessBlock,
                }), false)))();
            });
        }
        function createCorsRule() {
            return (0, pulumi_1.output)(args.cors).apply(function (cors) {
                var _a;
                var _b, _c, _d, _e, _f;
                if (cors === false)
                    return;
                return new ((_a = aws_1.s3.BucketCorsConfigurationV2).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.cors, "".concat(name, "Cors"), {
                    bucket: bucket.bucket,
                    corsRules: [
                        {
                            allowedHeaders: (_c = cors === null || cors === void 0 ? void 0 : cors.allowHeaders) !== null && _c !== void 0 ? _c : ["*"],
                            allowedMethods: (_d = cors === null || cors === void 0 ? void 0 : cors.allowMethods) !== null && _d !== void 0 ? _d : [
                                "DELETE",
                                "GET",
                                "HEAD",
                                "POST",
                                "PUT",
                            ],
                            allowedOrigins: (_e = cors === null || cors === void 0 ? void 0 : cors.allowOrigins) !== null && _e !== void 0 ? _e : ["*"],
                            exposeHeaders: cors === null || cors === void 0 ? void 0 : cors.exposeHeaders,
                            maxAgeSeconds: (0, duration_1.toSeconds)((_f = cors === null || cors === void 0 ? void 0 : cors.maxAge) !== null && _f !== void 0 ? _f : "0 seconds"),
                        },
                    ],
                }, { parent: parent }), false)))();
            });
        }
        return _this;
    }
    Object.defineProperty(Bucket.prototype, "name", {
        /**
         * The generated name of the S3 Bucket.
         */
        get: function () {
            return this.bucket.bucket;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Bucket.prototype, "domain", {
        /**
         * The domain name of the bucket. Has the format `${bucketName}.s3.amazonaws.com`.
         */
        get: function () {
            return this.bucket.bucketDomainName;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Bucket.prototype, "arn", {
        /**
         * The ARN of the S3 Bucket.
         */
        get: function () {
            return this.bucket.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Bucket.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon S3 bucket.
                 */
                bucket: this.bucket,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Reference an existing bucket with the given bucket name. This is useful when you
     * create a bucket in one stage and want to share it in another stage. It avoids having to
     * create a new bucket in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share buckets across stages.
     * :::
     *
     * @param name The name of the component.
     * @param bucketName The name of the existing S3 Bucket.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a bucket in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new bucket, you want to share the bucket from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const bucket = $app.stage === "frank"
     *   ? sst.aws.Bucket.get("MyBucket", "app-dev-mybucket-12345678")
     *   : new sst.aws.Bucket("MyBucket");
     * ```
     *
     * Here `app-dev-mybucket-12345678` is the auto-generated bucket name for the bucket created
     * in the `dev` stage. You can find this by outputting the bucket name in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   bucket: bucket.name
     * };
     * ```
     */
    Bucket.get = function (name, bucketName, opts) {
        return new Bucket(name, {
            ref: true,
            bucket: aws_1.s3.BucketV2.get("".concat(name, "Bucket"), bucketName, undefined, opts),
        });
    };
    /**
     * Subscribe to event notifications from this bucket. You can subscribe to these
     * notifications with a function, a queue, or a topic.
     *
     * @param args The config for the event notifications.
     *
     * @example
     *
     * For exmaple, to notify a function:
     *
     * ```js title="sst.config.ts" {5}
     * bucket.notify({
     *   notifications: [
     *     {
     *       name: "MySubscriber",
     *       function: "src/subscriber.handler"
     *     }
     *   ]
     * });
     * ```
     *
     * Or let's say you have a queue.
     *
     * ```js title="sst.config.ts"
     * const myQueue = new sst.aws.Queue("MyQueue");
     * ```
     *
     * You can notify it by passing in the queue.
     *
     * ```js title="sst.config.ts" {5}
     * bucket.notify({
     *   notifications: [
     *     {
     *       name: "MySubscriber",
     *       queue: myQueue
     *     }
     *   ]
     * });
     * ```
     *
     * Or let's say you have a topic.
     *
     * ```js title="sst.config.ts"
     * const myTopic = new sst.aws.SnsTopic("MyTopic");
     * ```
     *
     * You can notify it by passing in the topic.
     *
     * ```js title="sst.config.ts" {5}
     * bucket.notify({
     *   notifications: [
     *     {
     *       name: "MySubscriber",
     *       topic: myTopic
     *     }
     *   ]
     * });
     * ```
     *
     * You can also set it to only send notifications for specific S3 events.
     *
     * ```js {6}
     * bucket.notify({
     *   notifications: [
     *     {
     *       name: "MySubscriber",
     *       function: "src/subscriber.handler",
     *       events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     *     }
     *   ]
     * });
     * ```
     *
     * And you can add filters to be only notified from specific files in the bucket.
     *
     * ```js {6}
     * bucket.notify({
     *   notifications: [
     *     {
     *       name: "MySubscriber",
     *       function: "src/subscriber.handler",
     *       filterPrefix: "images/"
     *     }
     *   ]
     * });
     * ```
     */
    Bucket.prototype.notify = function (args) {
        if (this.isSubscribed) {
            throw new error_1.VisibleError("Cannot call \"notify\" on the \"".concat(this.constructorName, "\" bucket multiple times. Calling it again will override previous notifications."));
        }
        this.isSubscribed = true;
        var name = this.constructorName;
        var opts = this.constructorOpts;
        return new bucket_notification_1.BucketNotification("".concat(name, "Notifications"), __assign({ bucket: { name: this.bucket.bucket, arn: this.bucket.arn } }, args), opts);
    };
    /**
     * Subscribe to events from this bucket.
     *
     * @deprecated The `notify` function is now the recommended way to subscribe to events
     * from this bucket. It allows you to configure multiple subscribers at once. To migrate,
     * remove the current subscriber, deploy the changes, and then add the subscriber
     * back using the new `notify` function.
     *
     * @param subscriber The function that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * ```js title="sst.config.ts"
     * bucket.subscribe("src/subscriber.handler");
     * ```
     *
     * Subscribe to specific S3 events. The `link` ensures the subscriber can access the bucket.
     *
     * ```js title="sst.config.ts" "link: [bucket]"
     * bucket.subscribe({
     *   handler: "src/subscriber.handler",
     *   link: [bucket]
     * }, {
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     *
     * Subscribe to specific S3 events from a specific folder.
     *
     * ```js title="sst.config.ts" {2}
     * bucket.subscribe("src/subscriber.handler", {
     *   filterPrefix: "images/",
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     *
     * Customize the subscriber function.
     *
     * ```js title="sst.config.ts"
     * bucket.subscribe({
     *   handler: "src/subscriber.handler",
     *   timeout: "60 seconds",
     * });
     * ```
     *
     * Or pass in the ARN of an existing Lambda function.
     *
     * ```js title="sst.config.ts"
     * bucket.subscribe("arn:aws:lambda:us-east-1:123456789012:function:my-function");
     * ```
     */
    Bucket.prototype.subscribe = function (subscriber, args) {
        this.ensureNotSubscribed();
        return Bucket._subscribeFunction(this.constructorName, this.bucket.bucket, this.bucket.arn, subscriber, args, { provider: this.constructorOpts.provider });
    };
    /**
     * Subscribe to events of an S3 bucket that was not created in your app.
     *
     * @deprecated The `notify` function is now the recommended way to subscribe to events
     * from this bucket. It allows you to configure multiple subscribers at once. To migrate,
     * remove the current subscriber, deploy the changes, and then add the subscriber
     * back using the new `notify` function.
     *
     * @param bucketArn The ARN of the S3 bucket to subscribe to.
     * @param subscriber The function that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * For example, let's say you have an existing S3 bucket with the following ARN.
     *
     * ```js title="sst.config.ts"
     * const bucketArn = "arn:aws:s3:::my-bucket";
     * ```
     *
     * You can subscribe to it by passing in the ARN.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Bucket.subscribe(bucketArn, "src/subscriber.handler");
     * ```
     *
     * Subscribe to specific S3 events.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Bucket.subscribe(bucketArn, "src/subscriber.handler", {
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     *
     * Subscribe to specific S3 events from a specific folder.
     *
     * ```js title="sst.config.ts" {2}
     * sst.aws.Bucket.subscribe(bucketArn, "src/subscriber.handler", {
     *   filterPrefix: "images/",
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     *
     * Customize the subscriber function.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Bucket.subscribe(bucketArn, {
     *   handler: "src/subscriber.handler",
     *   timeout: "60 seconds",
     * });
     * ```
     */
    Bucket.subscribe = function (bucketArn, subscriber, args) {
        var _this = this;
        return (0, pulumi_1.output)(bucketArn).apply(function (bucketArn) {
            var bucketName = (0, arn_1.parseBucketArn)(bucketArn).bucketName;
            return _this._subscribeFunction(bucketName, bucketName, bucketArn, subscriber, args);
        });
    };
    Bucket._subscribeFunction = function (name, bucketName, bucketArn, subscriber, args, opts) {
        var _this = this;
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return (0, pulumi_1.all)([bucketArn, subscriber, args]).apply(function (_a) {
            var bucketArn = _a[0], subscriber = _a[1], args = _a[2];
            var subscriberId = _this.buildSubscriberId(bucketArn, typeof subscriber === "string" ? subscriber : subscriber.handler);
            return new bucket_lambda_subscriber_1.BucketLambdaSubscriber("".concat(name, "Subscriber").concat(subscriberId), __assign({ bucket: { name: bucketName, arn: bucketArn }, subscriber: subscriber, subscriberId: subscriberId }, args), opts);
        });
    };
    /**
     * Subscribe to events from this bucket with an SQS Queue.
     *
     * @deprecated The `notify` function is now the recommended way to subscribe to events
     * from this bucket. It allows you to configure multiple subscribers at once. To migrate,
     * remove the current subscriber, deploy the changes, and then add the subscriber
     * back using the new `notify` function.
     *
     * @param queueArn The ARN of the queue that'll be notified.
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
     * You can subscribe to this bucket with it.
     *
     * ```js title="sst.config.ts"
     * bucket.subscribe(queue.arn);
     * ```
     *
     * Subscribe to specific S3 events.
     *
     * ```js title="sst.config.ts"
     * bucket.subscribe(queue.arn, {
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     *
     * Subscribe to specific S3 events from a specific folder.
     *
     * ```js title="sst.config.ts" {2}
     * bucket.subscribe(queue.arn, {
     *   filterPrefix: "images/",
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     */
    Bucket.prototype.subscribeQueue = function (queueArn, args) {
        if (args === void 0) { args = {}; }
        this.ensureNotSubscribed();
        return Bucket._subscribeQueue(this.constructorName, this.bucket.bucket, this.arn, queueArn, args, { provider: this.constructorOpts.provider });
    };
    /**
     * Subscribe to events of an S3 bucket that was not created in your app with an SQS Queue.
     *
     * @deprecated The `notify` function is now the recommended way to subscribe to events
     * from this bucket. It allows you to configure multiple subscribers at once. To migrate,
     * remove the current subscriber, deploy the changes, and then add the subscriber
     * back using the new `notify` function.
     *
     * @param bucketArn The ARN of the S3 bucket to subscribe to.
     * @param queueArn The ARN of the queue that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * For example, let's say you have an existing S3 bucket and SQS queue with the following ARNs.
     *
     * ```js title="sst.config.ts"
     * const bucketArn = "arn:aws:s3:::my-bucket";
     * const queueArn = "arn:aws:sqs:us-east-1:123456789012:MyQueue";
     * ```
     *
     * You can subscribe to the bucket with the queue.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Bucket.subscribeQueue(bucketArn, queueArn);
     * ```
     *
     * Subscribe to specific S3 events.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Bucket.subscribeQueue(bucketArn, queueArn, {
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     *
     * Subscribe to specific S3 events from a specific folder.
     *
     * ```js title="sst.config.ts" {2}
     * sst.aws.Bucket.subscribeQueue(bucketArn, queueArn, {
     *   filterPrefix: "images/",
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     */
    Bucket.subscribeQueue = function (bucketArn, queueArn, args) {
        var _this = this;
        return (0, pulumi_1.output)(bucketArn).apply(function (bucketArn) {
            var bucketName = (0, arn_1.parseBucketArn)(bucketArn).bucketName;
            return _this._subscribeQueue(bucketName, bucketName, bucketArn, queueArn, args);
        });
    };
    Bucket._subscribeQueue = function (name, bucketName, bucketArn, queueArn, args, opts) {
        var _this = this;
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return (0, pulumi_1.all)([bucketArn, queueArn, args]).apply(function (_a) {
            var bucketArn = _a[0], queueArn = _a[1], args = _a[2];
            var subscriberId = _this.buildSubscriberId(bucketArn, queueArn);
            return new bucket_queue_subscriber_1.BucketQueueSubscriber("".concat(name, "Subscriber").concat(subscriberId), __assign({ bucket: { name: bucketName, arn: bucketArn }, queue: queueArn, subscriberId: subscriberId }, args), opts);
        });
    };
    /**
     * Subscribe to events from this bucket with an SNS Topic.
     *
     * @deprecated The `notify` function is now the recommended way to subscribe to events
     * from this bucket. It allows you to configure multiple subscribers at once. To migrate,
     * remove the current subscriber, deploy the changes, and then add the subscriber
     * back using the new `notify` function.
     *
     * @param topicArn The ARN of the topic that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * For example, let's say you have a topic.
     *
     * ```js title="sst.config.ts"
     * const topic = new sst.aws.SnsTopic("MyTopic");
     * ```
     *
     * You can subscribe to this bucket with it.
     *
     * ```js title="sst.config.ts"
     * bucket.subscribe(topic.arn);
     * ```
     *
     * Subscribe to specific S3 events.
     *
     * ```js title="sst.config.ts"
     * bucket.subscribe(topic.arn, {
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     *
     * Subscribe to specific S3 events from a specific folder.
     *
     * ```js title="sst.config.ts" {2}
     * bucket.subscribe(topic.arn, {
     *   filterPrefix: "images/",
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     */
    Bucket.prototype.subscribeTopic = function (topicArn, args) {
        if (args === void 0) { args = {}; }
        this.ensureNotSubscribed();
        return Bucket._subscribeTopic(this.constructorName, this.bucket.bucket, this.arn, topicArn, args, { provider: this.constructorOpts.provider });
    };
    /**
     * Subscribe to events of an S3 bucket that was not created in your app with an SNS Topic.
     *
     * @deprecated The `notify` function is now the recommended way to subscribe to events
     * from this bucket. It allows you to configure multiple subscribers at once. To migrate,
     * remove the current subscriber, deploy the changes, and then add the subscriber
     * back using the new `notify` function.
     *
     * @param bucketArn The ARN of the S3 bucket to subscribe to.
     * @param topicArn The ARN of the topic that'll be notified.
     * @param args Configure the subscription.
     *
     * @example
     *
     * For example, let's say you have an existing S3 bucket and SNS topic with the following ARNs.
     *
     * ```js title="sst.config.ts"
     * const bucketArn = "arn:aws:s3:::my-bucket";
     * const topicArn = "arn:aws:sns:us-east-1:123456789012:MyTopic";
     * ```
     *
     * You can subscribe to the bucket with the topic.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Bucket.subscribe(bucketArn, topicArn);
     * ```
     *
     * Subscribe to specific S3 events.
     *
     * ```js title="sst.config.ts"
     * sst.aws.Bucket.subscribe(bucketArn, topicArn, {
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     *
     * Subscribe to specific S3 events from a specific folder.
     *
     * ```js title="sst.config.ts" {2}
     * sst.aws.Bucket.subscribe(bucketArn, topicArn, {
     *   filterPrefix: "images/",
     *   events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
     * });
     * ```
     */
    Bucket.subscribeTopic = function (bucketArn, topicArn, args) {
        var _this = this;
        return (0, pulumi_1.output)(bucketArn).apply(function (bucketArn) {
            var bucketName = (0, arn_1.parseBucketArn)(bucketArn).bucketName;
            return _this._subscribeTopic(bucketName, bucketName, bucketArn, topicArn, args);
        });
    };
    Bucket._subscribeTopic = function (name, bucketName, bucketArn, topicArn, args, opts) {
        var _this = this;
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return (0, pulumi_1.all)([bucketArn, topicArn, args]).apply(function (_a) {
            var bucketArn = _a[0], topicArn = _a[1], args = _a[2];
            var subscriberId = _this.buildSubscriberId(bucketArn, topicArn);
            return new bucket_topic_subscriber_1.BucketTopicSubscriber("".concat(name, "Subscriber").concat(subscriberId), __assign({ bucket: { name: bucketName, arn: bucketArn }, topic: topicArn, subscriberId: subscriberId }, args), opts);
        });
    };
    Bucket.buildSubscriberId = function (bucketArn, _discriminator) {
        return (0, naming_1.logicalName)((0, naming_1.hashStringToPrettyString)([
            bucketArn,
            // Temporarily only allowing one subscriber per bucket because of the
            // AWS/Terraform issue that appending/removing a notification deletes
            // all existing notifications.
            //
            // A solution would be to implement a dynamic provider. On create,
            // get existing notifications then append. And on delete, get existing
            // notifications then remove from the list.
            //
            // https://github.com/hashicorp/terraform-provider-aws/issues/501
            //
            // Commenting out the lines below to ensure the id never changes.
            // Because on id change, the removal of notification happens after
            // the creation of notification. And the newly created notification
            // gets removed.
            //...events,
            //args.filterPrefix ?? "",
            //args.filterSuffix ?? "",
            //discriminator,
        ].join(""), 6));
    };
    Bucket.prototype.ensureNotSubscribed = function () {
        if (this.isSubscribed)
            throw new error_1.VisibleError("Cannot subscribe to the \"".concat(this.constructorName, "\" bucket multiple times. An S3 bucket can only have one subscriber."));
        this.isSubscribed = true;
    };
    /** @internal */
    Bucket.prototype.getSSTLink = function () {
        return {
            properties: {
                name: this.name,
            },
            include: [
                (0, permission_1.permission)({
                    actions: ["s3:*"],
                    resources: [this.arn, (0, pulumi_1.interpolate)(templateObject_4 || (templateObject_4 = __makeTemplateObject(["", "/*"], ["", "/*"])), this.arn)],
                }),
            ],
        };
    };
    return Bucket;
}(component_1.Component));
exports.Bucket = Bucket;
var __pulumiType = "sst:aws:Bucket";
// @ts-expect-error
Bucket.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2, templateObject_3, templateObject_4;
