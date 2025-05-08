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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype);
    return g.next = verb(0), g["throw"] = verb(1), g["return"] = verb(2), typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Task = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_js_1 = require("../component.js");
var permission_js_1 = require("./permission.js");
var vpc_js_1 = require("./vpc.js");
var function_js_1 = require("./function.js");
var fargate_js_1 = require("./fargate.js");
/**
 * The `Task` component lets you create containers that are used for long running asynchronous
 * work, like data processing. It uses [Amazon ECS](https://aws.amazon.com/ecs/) on
 * [AWS Fargate](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/AWS_Fargate.html).
 *
 * @example
 *
 * #### Create a Task
 *
 * Tasks are run inside an ECS Cluster. If you haven't already, create one.
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const cluster = new sst.aws.Cluster("MyCluster", { vpc });
 * ```
 *
 * Add the task to it.
 *
 * ```ts title="sst.config.ts"
 * const task = new sst.aws.Task("MyTask", { cluster });
 * ```
 *
 * #### Configure the container image
 *
 * By default, the task will look for a Dockerfile in the root directory. Optionally,
 * configure the image context and dockerfile.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Task("MyTask", {
 *   cluster,
 *   image: {
 *     context: "./app",
 *     dockerfile: "Dockerfile"
 *   }
 * });
 * ```
 *
 * To add multiple containers in the task, pass in an array of containers args.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Task("MyTask", {
 *   cluster,
 *   containers: [
 *     {
 *       name: "app",
 *       image: "nginxdemos/hello:plain-text"
 *     },
 *     {
 *       name: "admin",
 *       image: {
 *         context: "./admin",
 *         dockerfile: "Dockerfile"
 *       }
 *     }
 *   ]
 * });
 * ```
 *
 * This is useful for running sidecar containers.
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your task. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {5} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.Task("MyTask", {
 *   cluster,
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources in your task.
 *
 * ```ts title="app.ts"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 *
 * #### Task SDK
 *
 * With the [Task JS SDK](/docs/component/aws/task#sdk), you can run your tasks, stop your
 * tasks, and get the status of your tasks.
 *
 * For example, you can link the task to a function in your app.
 *
 * ```ts title="sst.config.ts" {3}
 * new sst.aws.Function("MyFunction", {
 *   handler: "src/lambda.handler",
 *   link: [task]
 * });
 * ```
 *
 * Then from your function run the task.
 *
 * ```ts title="src/lambda.ts"
 * import { Resource } from "sst";
 * import { task } from "sst/aws/task";
 *
 * const runRet = await task.run(Resource.MyTask);
 * const taskArn = runRet.arn;
 * ```
 *
 * If you are not using Node.js, you can use the AWS SDK instead. Here's
 * [how to run a task](https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_RunTask.html).
 *
 * ---
 *
 * ### Cost
 *
 * By default, this uses a _Linux/X86_ _Fargate_ container with 0.25 vCPUs at $0.04048 per
 * vCPU per hour and 0.5 GB of memory at $0.004445 per GB per hour. It includes 20GB of
 * _Ephemeral Storage_ for free with additional storage at $0.000111 per GB per hour. Each
 * container also gets a public IPv4 address at $0.005 per hour.
 *
 * It works out to $0.04048 x 0.25 + $0.004445 x 0.5 + $0.005. Or **$0.02 per hour**
 * your task runs for.
 *
 * Adjust this for the `cpu`, `memory` and `storage` you are using. And
 * check the prices for _Linux/ARM_ if you are using `arm64` as your `architecture`.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [Fargate pricing](https://aws.amazon.com/fargate/pricing/) and the
 * [Public IPv4 Address pricing](https://aws.amazon.com/vpc/pricing/) for more details.
 */
var Task = /** @class */ (function (_super) {
    __extends(Task, _super);
    function Task(name, args, opts) {
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var dev = normalizeDev();
        var architecture = (0, fargate_js_1.normalizeArchitecture)(args);
        var cpu = (0, fargate_js_1.normalizeCpu)(args);
        var memory = (0, fargate_js_1.normalizeMemory)(cpu, args);
        var storage = (0, fargate_js_1.normalizeStorage)(args);
        var containers = (0, fargate_js_1.normalizeContainers)("task", args, name, architecture);
        var vpc = normalizeVpc();
        var taskRole = (0, fargate_js_1.createTaskRole)(name, args, opts, self, dev, dev
            ? [
                {
                    actions: ["appsync:*"],
                    resources: ["*"],
                },
            ]
            : []);
        _this.dev = dev;
        _this.taskRole = taskRole;
        var executionRole = (0, fargate_js_1.createExecutionRole)(name, args, opts, self);
        var taskDefinition = (0, fargate_js_1.createTaskDefinition)(name, args, opts, self, dev
            ? containers.apply(function (v) { return __awaiter(_this, void 0, void 0, function () {
                var appsync;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0: return [4 /*yield*/, function_js_1.Function.appsync()];
                        case 1:
                            appsync = _a.sent();
                            return [2 /*return*/, [
                                    __assign(__assign({}, v[0]), { image: (0, pulumi_1.output)("ghcr.io/sst/sst/bridge-task:20241224005724"), environment: __assign(__assign({}, v[0].environment), { SST_TASK_ID: name, SST_REGION: process.env.SST_AWS_REGION, SST_APPSYNC_HTTP: appsync.http, SST_APPSYNC_REALTIME: appsync.realtime, SST_APP: $app.name, SST_STAGE: $app.stage }) }),
                                ]];
                    }
                });
            }); })
            : containers, architecture, cpu, memory, storage, taskRole, executionRole);
        _this._cluster = args.cluster;
        _this.vpc = vpc;
        _this.executionRole = executionRole;
        _this._taskDefinition = taskDefinition;
        _this.containerNames = containers.apply(function (v) { return v.map(function (v) { return (0, pulumi_1.output)(v.name); }); });
        _this.registerOutputs({
            _task: (0, pulumi_1.all)([args.dev, containers]).apply(function (_a) {
                var v = _a[0], containers = _a[1];
                return (__assign({ directory: (function () {
                        if (!containers[0].image)
                            return "";
                        if (typeof containers[0].image === "string")
                            return "";
                        if (containers[0].image.context)
                            return containers[0].image.context;
                        return "";
                    })() }, v));
            }),
        });
        function normalizeDev() {
            if (!$dev)
                return false;
            if (args.dev === false)
                return false;
            return true;
        }
        function normalizeVpc() {
            // "vpc" is a Vpc component
            if (args.cluster.vpc instanceof vpc_js_1.Vpc) {
                var vpc_1 = args.cluster.vpc;
                return {
                    isSstVpc: true,
                    containerSubnets: vpc_1.publicSubnets,
                    securityGroups: vpc_1.securityGroups,
                };
            }
            // "vpc" is object
            return {
                isSstVpc: false,
                containerSubnets: (0, pulumi_1.output)(args.cluster.vpc).apply(function (v) {
                    return v.containerSubnets.map(function (v) { return (0, pulumi_1.output)(v); });
                }),
                securityGroups: (0, pulumi_1.output)(args.cluster.vpc).apply(function (v) {
                    return v.securityGroups.map(function (v) { return (0, pulumi_1.output)(v); });
                }),
            };
        }
        return _this;
    }
    Object.defineProperty(Task.prototype, "taskDefinition", {
        /**
         * The ARN of the ECS Task Definition.
         */
        get: function () {
            return this._taskDefinition.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Task.prototype, "containers", {
        /**
         * The names of the containers in the task.
         * @internal
         */
        get: function () {
            return this.containerNames;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Task.prototype, "cluster", {
        /**
         * The ARN of the cluster this task is deployed to.
         * @internal
         */
        get: function () {
            return this._cluster.nodes.cluster.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Task.prototype, "securityGroups", {
        /**
         * The security groups for the task.
         * @internal
         */
        get: function () {
            return this.vpc.securityGroups;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Task.prototype, "subnets", {
        /**
         * The subnets for the task.
         * @internal
         */
        get: function () {
            return this.vpc.containerSubnets;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Task.prototype, "assignPublicIp", {
        /**
         * Whether to assign a public IP address to the task.
         * @internal
         */
        get: function () {
            return this.vpc.isSstVpc;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Task.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon ECS Execution Role.
                 */
                executionRole: this.executionRole,
                /**
                 * The Amazon ECS Task Role.
                 */
                taskRole: this.taskRole,
                /**
                 * The Amazon ECS Task Definition.
                 */
                taskDefinition: this._taskDefinition,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Task.prototype.getSSTLink = function () {
        return {
            properties: {
                cluster: this.cluster,
                containers: this.containers,
                taskDefinition: this.taskDefinition,
                subnets: this.subnets,
                securityGroups: this.securityGroups,
                assignPublicIp: this.assignPublicIp,
            },
            include: [
                (0, permission_js_1.permission)({
                    actions: ["ecs:*"],
                    resources: [
                        this._taskDefinition.arn,
                        // permissions to describe and stop the task
                        this.cluster.apply(function (v) { return v.split(":cluster/").join(":task/") + "/*"; }),
                    ],
                }),
                (0, permission_js_1.permission)({
                    actions: ["iam:PassRole"],
                    resources: [this.executionRole.arn, this.taskRole.arn],
                }),
            ],
        };
    };
    return Task;
}(component_js_1.Component));
exports.Task = Task;
var __pulumiType = "sst:aws:Task";
// @ts-expect-error
Task.__pulumiType = __pulumiType;
