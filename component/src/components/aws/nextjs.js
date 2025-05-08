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
exports.Nextjs = void 0;
var fs_1 = require("fs");
var path_1 = require("path");
var pulumi_1 = require("@pulumi/pulumi");
var function_js_1 = require("./function.js");
var error_js_1 = require("../error.js");
var queue_js_1 = require("./queue.js");
var aws_1 = require("@pulumi/aws");
var compare_semver_js_1 = require("../../util/compare-semver.js");
var ssr_site_js_1 = require("./ssr-site.js");
var DEFAULT_OPEN_NEXT_VERSION = "3.5.5";
/**
 * The `Nextjs` component lets you deploy [Next.js](https://nextjs.org) apps on AWS. It uses
 * [OpenNext](https://open-next.js.org) to build your Next.js app, and transforms the build
 * output to a format that can be deployed to AWS.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy the Next.js app that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys a Next.js app in the `my-next-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   path: "my-next-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your Next.js app.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your Next.js app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your Next.js app.
 *
 * ```ts title="app/page.tsx"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 */
var Nextjs = /** @class */ (function (_super) {
    __extends(Nextjs, _super);
    function Nextjs(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        return _super.call(this, __pulumiType, name, args, opts) || this;
    }
    Nextjs.prototype.normalizeBuildCommand = function (args) {
        return (0, pulumi_1.all)([args === null || args === void 0 ? void 0 : args.buildCommand, args === null || args === void 0 ? void 0 : args.openNextVersion]).apply(function (_a) {
            var buildCommand = _a[0], openNextVersion = _a[1];
            if (buildCommand)
                return buildCommand;
            var version = openNextVersion !== null && openNextVersion !== void 0 ? openNextVersion : DEFAULT_OPEN_NEXT_VERSION;
            var packageName = (0, compare_semver_js_1.isALteB)(version, "3.1.3")
                ? "open-next"
                : "@opennextjs/aws";
            return "npx --yes ".concat(packageName, "@").concat(version, " build");
        });
    };
    Nextjs.prototype.buildPlan = function (outputPath, name, args, _a) {
        var bucket = _a.bucket;
        var parent = this;
        var ret = (0, pulumi_1.all)([outputPath, args === null || args === void 0 ? void 0 : args.imageOptimization]).apply(function (_a) {
            var outputPath = _a[0], imageOptimization = _a[1];
            var _b = loadBuildOutput(), openNextOutput = _b.openNextOutput, buildId = _b.buildId, prerenderManifest = _b.prerenderManifest, base = _b.base;
            if (Object.entries(openNextOutput.edgeFunctions).length) {
                throw new error_js_1.VisibleError("Lambda@Edge runtime is deprecated. Update your OpenNext configuration to use the standard Lambda runtime and deploy to multiple regions using the \"regions\" option in your Nextjs component.");
            }
            var _c = createRevalidationQueue(), revalidationQueue = _c.revalidationQueue, revalidationFunction = _c.revalidationFunction;
            var revalidationTable = createRevalidationTable();
            createRevalidationTableSeeder();
            var serverOrigin = openNextOutput.origins["default"];
            var imageOptimizerOrigin = openNextOutput.origins["imageOptimizer"];
            var s3Origin = openNextOutput.origins["s3"];
            var plan = (0, pulumi_1.all)([
                revalidationTable === null || revalidationTable === void 0 ? void 0 : revalidationTable.arn,
                revalidationTable === null || revalidationTable === void 0 ? void 0 : revalidationTable.name,
                bucket.arn,
                bucket.name,
                (0, aws_1.getRegionOutput)(undefined, { parent: bucket }).name,
                revalidationQueue === null || revalidationQueue === void 0 ? void 0 : revalidationQueue.arn,
                revalidationQueue === null || revalidationQueue === void 0 ? void 0 : revalidationQueue.url,
                (0, aws_1.getRegionOutput)(undefined, { parent: revalidationQueue }).name,
            ]).apply(function (_a) {
                var _b;
                var tableArn = _a[0], tableName = _a[1], bucketArn = _a[2], bucketName = _a[3], bucketRegion = _a[4], queueArn = _a[5], queueUrl = _a[6], queueRegion = _a[7];
                return ({
                    base: base,
                    server: {
                        description: "".concat(name, " server"),
                        bundle: path_1.default.join(outputPath, serverOrigin.bundle),
                        handler: serverOrigin.handler,
                        streaming: serverOrigin.streaming,
                        runtime: "nodejs20.x",
                        environment: __assign(__assign({ CACHE_BUCKET_NAME: bucketName, CACHE_BUCKET_KEY_PREFIX: "_cache", CACHE_BUCKET_REGION: bucketRegion }, (queueUrl && {
                            REVALIDATION_QUEUE_URL: queueUrl,
                            REVALIDATION_QUEUE_REGION: queueRegion,
                        })), (tableName && {
                            CACHE_DYNAMO_TABLE: tableName,
                        })),
                        permissions: __spreadArray(__spreadArray([
                            // access to the cache data
                            {
                                actions: ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
                                resources: ["".concat(bucketArn, "/*")],
                            },
                            {
                                actions: ["s3:ListBucket"],
                                resources: [bucketArn],
                            }
                        ], (queueArn
                            ? [
                                {
                                    actions: [
                                        "sqs:SendMessage",
                                        "sqs:GetQueueAttributes",
                                        "sqs:GetQueueUrl",
                                    ],
                                    resources: [queueArn],
                                },
                            ]
                            : []), true), (tableArn
                            ? [
                                {
                                    actions: [
                                        "dynamodb:BatchGetItem",
                                        "dynamodb:GetRecords",
                                        "dynamodb:GetShardIterator",
                                        "dynamodb:Query",
                                        "dynamodb:GetItem",
                                        "dynamodb:Scan",
                                        "dynamodb:ConditionCheckItem",
                                        "dynamodb:BatchWriteItem",
                                        "dynamodb:PutItem",
                                        "dynamodb:UpdateItem",
                                        "dynamodb:DeleteItem",
                                        "dynamodb:DescribeTable",
                                    ],
                                    resources: [tableArn, "".concat(tableArn, "/*")],
                                },
                            ]
                            : []), true),
                        injections: [
                            [
                                "outer:if (process.env.SST_KEY_FILE) {",
                                "  const { readFileSync } = await import(\"fs\")",
                                "  const { createDecipheriv } = await import(\"crypto\")",
                                "  const key = Buffer.from(process.env.SST_KEY, \"base64\");",
                                "  const encryptedData = readFileSync(process.env.SST_KEY_FILE);",
                                "  const nonce = Buffer.alloc(12, 0);",
                                "  const decipher = createDecipheriv(\"aes-256-gcm\", key, nonce);",
                                "  const authTag = encryptedData.slice(-16);",
                                "  const actualCiphertext = encryptedData.slice(0, -16);",
                                "  decipher.setAuthTag(authTag);",
                                "  let decrypted = decipher.update(actualCiphertext);",
                                "  decrypted = Buffer.concat([decrypted, decipher.final()]);",
                                "  const decryptedData = JSON.parse(decrypted.toString());",
                                "  globalThis.SST_KEY_FILE_DATA = decryptedData;",
                                "}",
                            ].join("\n"),
                        ],
                    },
                    imageOptimizer: {
                        prefix: "/_next/image",
                        function: {
                            description: "".concat(name, " image optimizer"),
                            handler: imageOptimizerOrigin.handler,
                            bundle: path_1.default.join(outputPath, imageOptimizerOrigin.bundle),
                            runtime: "nodejs20.x",
                            architecture: "arm64",
                            environment: __assign({ BUCKET_NAME: bucketName, BUCKET_KEY_PREFIX: "_assets" }, ((imageOptimization === null || imageOptimization === void 0 ? void 0 : imageOptimization.staticEtag)
                                ? { OPENNEXT_STATIC_ETAG: "true" }
                                : {})),
                            memory: (_b = imageOptimization === null || imageOptimization === void 0 ? void 0 : imageOptimization.memory) !== null && _b !== void 0 ? _b : "1536 MB",
                        },
                    },
                    assets: [
                        {
                            from: ".open-next/assets",
                            to: "_assets",
                            cached: true,
                            versionedSubDir: "_next",
                            deepRoute: true,
                        },
                    ],
                    isrCache: {
                        from: ".open-next/cache",
                        to: "_cache",
                    },
                    buildId: buildId,
                });
            });
            return {
                plan: plan,
                revalidationQueue: revalidationQueue,
                revalidationTable: revalidationTable,
                revalidationFunction: revalidationFunction,
            };
            function loadBuildOutput() {
                var _a;
                var openNextOutputPath = path_1.default.join(outputPath, ".open-next", "open-next.output.json");
                if (!fs_1.default.existsSync(openNextOutputPath)) {
                    throw new error_js_1.VisibleError("Could not load OpenNext output file at \"".concat(openNextOutputPath, "\". Make sure your Next.js app was built correctly with OpenNext."));
                }
                var content = fs_1.default.readFileSync(openNextOutputPath).toString();
                var json = JSON.parse(content);
                // Currently open-next.output.json's initializationFunction value
                // is wrong, it is set to ".open-next/initialization-function"
                if ((_a = json.additionalProps) === null || _a === void 0 ? void 0 : _a.initializationFunction) {
                    json.additionalProps.initializationFunction = {
                        handler: "index.handler",
                        bundle: ".open-next/dynamodb-provider",
                    };
                }
                return {
                    openNextOutput: json,
                    base: loadBasePath(),
                    buildId: loadBuildId(),
                    prerenderManifest: loadPrerenderManifest(),
                };
            }
            function loadBuildId() {
                try {
                    return fs_1.default
                        .readFileSync(path_1.default.join(outputPath, ".next/BUILD_ID"))
                        .toString();
                }
                catch (e) {
                    console.error(e);
                    throw new error_js_1.VisibleError("Build ID not found in \".next/BUILD_ID\" for site \"".concat(name, "\". Ensure your Next.js app was built successfully."));
                }
            }
            function loadBasePath() {
                try {
                    var content = fs_1.default.readFileSync(path_1.default.join(outputPath, ".next", "routes-manifest.json"), "utf-8");
                    var json = JSON.parse(content);
                    return json.basePath === "" ? undefined : json.basePath;
                }
                catch (e) {
                    console.error(e);
                    throw new error_js_1.VisibleError("Base path configuration not found in \".next/routes-manifest.json\" for site \"".concat(name, "\". Check your Next.js configuration."));
                }
            }
            function loadPrerenderManifest() {
                try {
                    var content = fs_1.default
                        .readFileSync(path_1.default.join(outputPath, ".next/prerender-manifest.json"))
                        .toString();
                    return JSON.parse(content);
                }
                catch (e) {
                    console.debug("Failed to load prerender-manifest.json", e);
                }
            }
            function createRevalidationQueue() {
                var _a, _b;
                if ((_a = openNextOutput.additionalProps) === null || _a === void 0 ? void 0 : _a.disableIncrementalCache)
                    return {};
                var revalidationFunction = (_b = openNextOutput.additionalProps) === null || _b === void 0 ? void 0 : _b.revalidationFunction;
                if (!revalidationFunction)
                    return {};
                var queue = new queue_js_1.Queue("".concat(name, "RevalidationEvents"), {
                    fifo: true,
                    transform: {
                        queue: function (args) {
                            args.receiveWaitTimeSeconds = 20;
                        },
                    },
                }, { parent: parent });
                var subscriber = queue.subscribe({
                    description: "".concat(name, " ISR revalidator"),
                    handler: revalidationFunction.handler,
                    bundle: path_1.default.join(outputPath, revalidationFunction.bundle),
                    runtime: "nodejs20.x",
                    timeout: "30 seconds",
                    permissions: [
                        {
                            actions: [
                                "sqs:ChangeMessageVisibility",
                                "sqs:DeleteMessage",
                                "sqs:GetQueueAttributes",
                                "sqs:GetQueueUrl",
                                "sqs:ReceiveMessage",
                            ],
                            resources: [queue.arn],
                        },
                    ],
                    dev: false,
                    _skipMetadata: true,
                }, {
                    transform: {
                        eventSourceMapping: function (args) {
                            args.batchSize = 5;
                        },
                    },
                }, { parent: parent });
                return {
                    revalidationQueue: queue,
                    revalidationFunction: subscriber.nodes.function,
                };
            }
            function createRevalidationTable() {
                var _a;
                if ((_a = openNextOutput.additionalProps) === null || _a === void 0 ? void 0 : _a.disableTagCache)
                    return;
                return new aws_1.dynamodb.Table("".concat(name, "RevalidationTable"), {
                    attributes: [
                        { name: "tag", type: "S" },
                        { name: "path", type: "S" },
                        { name: "revalidatedAt", type: "N" },
                    ],
                    hashKey: "tag",
                    rangeKey: "path",
                    pointInTimeRecovery: {
                        enabled: true,
                    },
                    billingMode: "PAY_PER_REQUEST",
                    globalSecondaryIndexes: [
                        {
                            name: "revalidate",
                            hashKey: "path",
                            rangeKey: "revalidatedAt",
                            projectionType: "ALL",
                        },
                    ],
                }, { parent: parent, retainOnDelete: false });
            }
            function createRevalidationTableSeeder() {
                var _a, _b, _c;
                if ((_a = openNextOutput.additionalProps) === null || _a === void 0 ? void 0 : _a.disableTagCache)
                    return;
                if (!((_b = openNextOutput.additionalProps) === null || _b === void 0 ? void 0 : _b.initializationFunction))
                    return;
                // Provision 128MB of memory for every 4,000 prerendered routes,
                // 1GB per 40,000, up to 10GB. This tends to use ~70% of the memory
                // provisioned when testing.
                var prerenderedRouteCount = Object.keys((_c = prerenderManifest === null || prerenderManifest === void 0 ? void 0 : prerenderManifest.routes) !== null && _c !== void 0 ? _c : {}).length;
                var seedFn = new function_js_1.Function("".concat(name, "RevalidationSeeder"), {
                    description: "".concat(name, " ISR revalidation data seeder"),
                    handler: openNextOutput.additionalProps.initializationFunction.handler,
                    bundle: path_1.default.join(outputPath, openNextOutput.additionalProps.initializationFunction.bundle),
                    runtime: "nodejs20.x",
                    timeout: "900 seconds",
                    memory: "".concat(Math.min(10240, Math.max(128, Math.ceil(prerenderedRouteCount / 4000) * 128)), " MB"),
                    permissions: [
                        {
                            actions: [
                                "dynamodb:BatchWriteItem",
                                "dynamodb:PutItem",
                                "dynamodb:DescribeTable",
                            ],
                            resources: [revalidationTable.arn],
                        },
                    ],
                    environment: {
                        CACHE_DYNAMO_TABLE: revalidationTable.name,
                    },
                    dev: false,
                    _skipMetadata: true,
                    _skipHint: true,
                }, { parent: parent });
                new aws_1.lambda.Invocation("".concat(name, "RevalidationSeed"), {
                    functionName: seedFn.nodes.function.name,
                    triggers: {
                        version: Date.now().toString(),
                    },
                    input: JSON.stringify({
                        RequestType: "Create",
                    }),
                }, { parent: parent });
            }
        });
        this.revalidationQueue = ret.revalidationQueue;
        this.revalidationTable = ret.revalidationTable;
        this.revalidationFunction = (0, pulumi_1.output)(ret.revalidationFunction);
        return ret.plan;
    };
    Object.defineProperty(Nextjs.prototype, "url", {
        /**
         * The URL of the Next.js app.
         *
         * If the `domain` is set, this is the URL with the custom domain.
         * Otherwise, it's the auto-generated CloudFront URL.
         */
        get: function () {
            return _super.prototype.url;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Nextjs.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return __assign(__assign({}, _super.prototype.nodes), { 
                /**
                 * The Amazon SQS queue that triggers the ISR revalidator.
                 */
                revalidationQueue: this.revalidationQueue, 
                /**
                 * The Amazon DynamoDB table that stores the ISR revalidation data.
                 */
                revalidationTable: this.revalidationTable, 
                /**
                 * The Lambda function that processes the ISR revalidation.
                 */
                revalidationFunction: this.revalidationFunction });
        },
        enumerable: false,
        configurable: true
    });
    return Nextjs;
}(ssr_site_js_1.SsrSite));
exports.Nextjs = Nextjs;
var __pulumiType = "sst:aws:Nextjs";
// @ts-expect-error
Nextjs.__pulumiType = __pulumiType;
