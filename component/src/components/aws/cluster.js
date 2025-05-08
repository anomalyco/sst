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
exports.Cluster = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var service_1 = require("./service");
var aws_1 = require("@pulumi/aws");
var cluster_v1_1 = require("./cluster-v1");
var vpc_1 = require("./vpc");
var vpc_v1_js_1 = require("./vpc-v1.js");
var task_1 = require("./task");
var error_1 = require("../error");
/**
 * The `Cluster` component lets you create an [ECS cluster](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/clusters.html) for your app.
 * add `Service` and `Task` components to it.
 *
 * @example
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const cluster = new sst.aws.Cluster("MyCluster", { vpc });
 * ```
 *
 * Once created, you can add the following to it:
 *
 * 1. `Service`: These are containers that are always running, like web or
 *   application servers. They automatically restart if they fail.
 * 2. `Task`: These are containers that are used for long running asynchronous work,
 *   like data processing.
 */
var Cluster = /** @class */ (function (_super) {
    __extends(Cluster, _super);
    function Cluster(name, args, opts) {
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var _version = { major: 2, minor: 0 };
        var self = _this;
        _this.constructorOpts = opts;
        if (args && "ref" in args) {
            var ref = reference();
            var vpc_2 = normalizeVpc();
            _this.cluster = ref.cluster;
            _this._vpc = vpc_2;
            return _this;
        }
        registerVersion();
        var vpc = normalizeVpc();
        var cluster = createCluster();
        createCapacityProviders();
        _this.cluster = (0, pulumi_1.output)(cluster);
        _this._vpc = vpc;
        function reference() {
            var ref = args;
            var cluster = aws_1.ecs.Cluster.get("".concat(name, "Cluster"), ref.id, undefined, {
                parent: self,
            });
            var clusterValidated = cluster.tags.apply(function (tags) {
                var refVersion = (tags === null || tags === void 0 ? void 0 : tags["sst:ref:version"])
                    ? (0, component_1.parseComponentVersion)(tags["sst:ref:version"])
                    : undefined;
                if ((refVersion === null || refVersion === void 0 ? void 0 : refVersion.minor) !== _version.minor) {
                    throw new error_1.VisibleError([
                        "There have been some minor changes to the \"Cluster\" component that's being referenced by \"".concat(name, "\".\n"),
                        "To update, you'll need to redeploy the stage where the cluster was created. And then redeploy this stage.",
                    ].join("\n"));
                }
                registerVersion(refVersion);
                return cluster;
            });
            return { cluster: clusterValidated };
        }
        function normalizeVpc() {
            // "vpc" is a Vpc.v1 component
            if (args.vpc instanceof vpc_v1_js_1.Vpc) {
                throw new error_1.VisibleError("You are using the \"Vpc.v1\" component. Please migrate to the latest \"Vpc\" component.");
            }
            // "vpc" is a Vpc component
            if (args.vpc instanceof vpc_1.Vpc) {
                return args.vpc;
            }
            // "vpc" is object
            return (0, pulumi_1.output)(args.vpc).apply(function (vpc) {
                var _a;
                if (vpc.containerSubnets && vpc.serviceSubnets)
                    throw new error_1.VisibleError("You cannot provide both \"vpc.containerSubnets\" and \"vpc.serviceSubnets\" in the \"".concat(name, "\" Cluster component. The \"serviceSubnets\" property has been deprecated. Use \"containerSubnets\" instead."));
                if (!vpc.containerSubnets && !vpc.serviceSubnets)
                    throw new error_1.VisibleError("Missing \"vpc.containerSubnets\" for the \"".concat(name, "\" Cluster component."));
                if ((vpc.cloudmapNamespaceId && !vpc.cloudmapNamespaceName) ||
                    (!vpc.cloudmapNamespaceId && vpc.cloudmapNamespaceName))
                    throw new error_1.VisibleError("You must provide both \"vpc.cloudmapNamespaceId\" and \"vpc.cloudmapNamespaceName\" for the \"".concat(name, "\" Cluster component."));
                return __assign(__assign({}, vpc), { containerSubnets: ((_a = vpc.containerSubnets) !== null && _a !== void 0 ? _a : vpc.serviceSubnets), serviceSubnets: undefined });
            });
        }
        function createCluster() {
            var _a;
            var _b;
            return new ((_a = aws_1.ecs.Cluster).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.cluster, "".concat(name, "Cluster"), {
                tags: {
                    "sst:ref:version": "".concat(_version.major, ".").concat(_version.minor),
                },
            }, { parent: self }), false)))();
        }
        function registerVersion(overrideVersion) {
            var _a;
            var newMajorVersion = _version.major;
            var oldMajorVersion = (_a = overrideVersion === null || overrideVersion === void 0 ? void 0 : overrideVersion.major) !== null && _a !== void 0 ? _a : $cli.state.version[name];
            self.registerVersion({
                new: newMajorVersion,
                old: oldMajorVersion,
                message: [
                    "There is a new version of \"Cluster\" that has breaking changes.",
                    "",
                    "What changed:",
                    "  - In the old version, load balancers were deployed in public subnets, and services were deployed in private subnets. The VPC was required to have NAT gateways.",
                    "  - In the latest version, both the load balancer and the services are deployed in public subnets. The VPC is not required to have NAT gateways. So the new default makes this cheaper to run.",
                    "",
                    "To upgrade:",
                    "  - Set `forceUpgrade: \"v".concat(newMajorVersion, "\"` on the \"Cluster\" component. Learn more https://sst.dev/docs/component/aws/cluster#forceupgrade"),
                    "",
                    "To continue using v".concat($cli.state.version[name], ":"),
                    "  - Rename \"Cluster\" to \"Cluster.v".concat($cli.state.version[name], "\". Learn more about versioning - https://sst.dev/docs/components/#versioning"),
                ].join("\n"),
                forceUpgrade: args.forceUpgrade,
            });
        }
        function createCapacityProviders() {
            return new aws_1.ecs.ClusterCapacityProviders("".concat(name, "CapacityProviders"), {
                clusterName: cluster.name,
                capacityProviders: ["FARGATE", "FARGATE_SPOT"],
            }, { parent: self });
        }
        return _this;
    }
    Object.defineProperty(Cluster.prototype, "id", {
        /**
         * The cluster ID.
         */
        get: function () {
            return this.cluster.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Cluster.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon ECS Cluster.
                 */
                cluster: this.cluster,
            };
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Cluster.prototype, "vpc", {
        /**
         * The VPC configuration for the cluster.
         * @internal
         */
        get: function () {
            return this._vpc;
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Add a service to the cluster.
     *
     * @deprecated Use the `Service` component directly to create services. To migrate, change
     *
     * ```ts
     * cluster.addService("MyService", { ...args });
     * ```
     *
     * to
     *
     * ```ts
     * new sst.aws.Service("MyService", { cluster, ...args });
     * ```
     *
     * @param name Name of the service.
     * @param args? Configure the service.
     * @param opts? Resource options.
     *
     * @example
     *
     * ```ts title="sst.config.ts"
     * cluster.addService("MyService");
     * ```
     *
     * You can also configure the service. For example, set a custom domain.
     *
     * ```js {2} title="sst.config.ts"
     * cluster.addService("MyService", {
     *   domain: "example.com"
     * });
     * ```
     *
     * Enable auto-scaling.
     *
     * ```ts title="sst.config.ts"
     * cluster.addService("MyService", {
     *   scaling: {
     *     min: 4,
     *     max: 16,
     *     cpuUtilization: 50,
     *     memoryUtilization: 50,
     *   }
     * });
     * ```
     *
     * By default this starts a single container. To add multiple containers in the service, pass in an array of containers args.
     *
     * ```ts title="sst.config.ts"
     * cluster.addService("MyService", {
     *   architecture: "arm64",
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
     */
    Cluster.prototype.addService = function (name, args, opts) {
        // Do not prefix the service to allow `Resource.MyService` to work.
        return new service_1.Service(name, __assign({ cluster: this }, args), __assign({ provider: this.constructorOpts.provider }, opts));
    };
    /**
     * Add a task to the cluster.
     *
     * @deprecated Use the `Task` component directly to create tasks. To migrate, change
     *
     * ```ts
     * cluster.addTask("MyTask", { ...args });
     * ```
     *
     * to
     *
     * ```ts
     * new sst.aws.Task("MyTask", { cluster, ...args });
     * ```
     *
     * @param name Name of the task.
     * @param args? Configure the task.
     * @param opts? Resource options.
     *
     * @example
     *
     * ```ts title="sst.config.ts"
     * cluster.addTask("MyTask");
     * ```
     *
     * You can also configure the task. By default this starts a single container.
     * To add multiple containers in the task, pass in an array of containers args.
     *
     * ```ts title="sst.config.ts"
     * cluster.addTask("MyTask", {
     *   architecture: "arm64",
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
     */
    Cluster.prototype.addTask = function (name, args, opts) {
        // Do not prefix the task to allow `Resource.MyTask` to work.
        return new task_1.Task(name, __assign({ cluster: this }, args), __assign({ provider: this.constructorOpts.provider }, opts));
    };
    /**
     * Reference an existing ECS Cluster with the given ID. This is useful when you
     * create a cluster in one stage and want to share it in another. It avoids
     * having to create a new cluster in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share cluster across stages.
     * :::
     *
     * @param name The name of the component.
     * @param args The arguments to get the cluster.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a cluster in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new cluster, you want to share the same cluster from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const cluster = $app.stage === "frank"
     *   ? sst.aws.Cluster.get("MyCluster", {
     *       id: "arn:aws:ecs:us-east-1:123456789012:cluster/app-dev-MyCluster",
     *       vpc,
     *     })
     *   : new sst.aws.Cluster("MyCluster", { vpc });
     * ```
     *
     * Here `arn:aws:ecs:us-east-1:123456789012:cluster/app-dev-MyCluster` is the ID of the
     * cluster created in the `dev` stage. You can find these by outputting the cluster ID
     * in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   id: cluster.id,
     * };
     * ```
     */
    Cluster.get = function (name, args, opts) {
        return new Cluster(name, { ref: true, id: args.id, vpc: args.vpc }, opts);
    };
    Cluster.v1 = cluster_v1_1.Cluster;
    return Cluster;
}(component_1.Component));
exports.Cluster = Cluster;
var __pulumiType = "sst:aws:Cluster";
// @ts-expect-error
Cluster.__pulumiType = __pulumiType;
