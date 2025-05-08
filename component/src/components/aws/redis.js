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
exports.Redis = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var random_1 = require("@pulumi/random");
var component_js_1 = require("../component.js");
var aws_1 = require("@pulumi/aws");
var vpc_js_1 = require("./vpc.js");
var error_js_1 = require("../error.js");
var dev_command_js_1 = require("../experimental/dev-command.js");
var redis_v1_1 = require("./redis-v1");
/**
 * The `Redis` component lets you add a Redis cluster to your app using
 * [Amazon ElastiCache](https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/WhatIs.html).
 *
 * @example
 *
 * #### Create the cluster
 *
 * ```js title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const redis = new sst.aws.Redis("MyRedis", { vpc });
 * ```
 *
 * #### Link to a resource
 *
 * You can link your cluster to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [redis],
 *   vpc
 * });
 * ```
 *
 * Once linked, you can connect to it from your function code.
 *
 * ```ts title="app/page.tsx" {1,6,7,12,13}
 * import { Resource } from "sst";
 * import { Cluster } from "ioredis";
 *
 * const client = new Cluster(
 *   [{
 *     host: Resource.MyRedis.host,
 *     port: Resource.MyRedis.port
 *   }],
 *   {
 *     redisOptions: {
 *       tls: { checkServerIdentity: () => undefined },
 *       username: Resource.MyRedis.username,
 *       password: Resource.MyRedis.password
 *     }
 *   }
 * );
 * ```
 *
 * #### Running locally
 *
 * By default, your Redis cluster is deployed in `sst dev`. But let's say you are running Redis
 * locally.
 *
 * ```bash
 * docker run \
 *   --rm \
 *   -p 6379:6379 \
 *   -v $(pwd)/.sst/storage/redis:/data \
 *   redis:latest
 * ```
 *
 * You can connect to it in `sst dev` by configuring the `dev` prop.
 *
 * ```ts title="sst.config.ts" {3-6}
 * const redis = new sst.aws.Redis("MyRedis", {
 *   vpc,
 *   dev: {
 *     host: "localhost",
 *     port: 6379
 *   }
 * });
 * ```
 *
 * This will skip deploying a Redis ElastiCache cluster and link to the locally running Redis
 * server instead. [Check out the full example](/docs/examples/#aws-redis-local).
 *
 * ---
 *
 * ### Cost
 *
 * By default this component uses _On-demand nodes_ with a single `cache.t4g.micro` instance.
 *
 * The default `redis` engine costs $0.016 per hour. That works out to $0.016 x 24 x 30 or **$12 per month**.
 *
 * If the `valkey` engine is used, the cost is $0.0128 per hour. That works out to $0.0128 x 24 x 30 or **$9 per month**.
 *
 * Adjust this for the `instance` type and number of `nodes` you are using.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [ElastiCache pricing](https://aws.amazon.com/elasticache/pricing/) for more details.
 */
var Redis = /** @class */ (function (_super) {
    __extends(Redis, _super);
    function Redis(name, args, opts) {
        var _this_1 = _super.call(this, __pulumiType, name, args, opts) || this;
        var _version = 2;
        var self = _this_1;
        if (args && "ref" in args) {
            var ref = reference();
            _this_1.cluster = (0, pulumi_1.output)(ref.cluster);
            _this_1._authToken = ref.authToken;
            return _this_1;
        }
        registerVersion();
        var engine = (0, pulumi_1.output)(args.engine).apply(function (v) { return v !== null && v !== void 0 ? v : "redis"; });
        var version = (0, pulumi_1.all)([engine, args.version]).apply(function (_a) {
            var engine = _a[0], v = _a[1];
            return v !== null && v !== void 0 ? v : (engine === "redis" ? "7.1" : "7.2");
        });
        var instance = (0, pulumi_1.output)(args.instance).apply(function (v) { return v !== null && v !== void 0 ? v : "t4g.micro"; });
        var argsCluster = normalizeCluster();
        var vpc = normalizeVpc();
        var dev = registerDev();
        if (dev === null || dev === void 0 ? void 0 : dev.enabled) {
            _this_1.dev = dev;
            return _this_1;
        }
        var _a = createAuthToken(), authToken = _a.authToken, secret = _a.secret;
        var subnetGroup = createSubnetGroup();
        var parameterGroup = createParameterGroup();
        var cluster = createCluster();
        _this_1.cluster = cluster;
        _this_1._authToken = authToken;
        function reference() {
            var ref = args;
            var cluster = aws_1.elasticache.ReplicationGroup.get("".concat(name, "Cluster"), ref.clusterId, undefined, { parent: self });
            var input = cluster.tags.apply(function (tags) {
                registerVersion((tags === null || tags === void 0 ? void 0 : tags["sst:component-version"])
                    ? parseInt(tags["sst:component-version"])
                    : undefined);
                if (!(tags === null || tags === void 0 ? void 0 : tags["sst:ref:secret"]))
                    throw new error_js_1.VisibleError("Failed to lookup secret for Redis cluster \"".concat(name, "\"."));
                return {
                    secretRef: tags === null || tags === void 0 ? void 0 : tags["sst:ref:secret"],
                };
            });
            var secret = aws_1.secretsmanager.getSecretVersionOutput({ secretId: input.secretRef }, { parent: self });
            var authToken = secret.secretString.apply(function (v) {
                return JSON.parse(v).authToken;
            });
            return { cluster: cluster, authToken: authToken };
        }
        function registerVersion(overrideVersion) {
            var oldVersion = overrideVersion !== null && overrideVersion !== void 0 ? overrideVersion : $cli.state.version[name];
            self.registerVersion({
                new: _version,
                old: oldVersion,
                message: [
                    "There is a new version of \"Redis\" that has breaking changes.",
                    "",
                    "To continue using the previous version, rename \"Redis\" to \"Redis.v".concat(oldVersion, "\"."),
                    "Or recreate this component to update - https://sst.dev/docs/components/#versioning",
                ].join("\n"),
            });
        }
        function registerDev() {
            var _a, _b, _c;
            if (!args.dev)
                return undefined;
            var dev = {
                enabled: $dev,
                host: (0, pulumi_1.output)((_a = args.dev.host) !== null && _a !== void 0 ? _a : "localhost"),
                port: (0, pulumi_1.output)((_b = args.dev.port) !== null && _b !== void 0 ? _b : 6379),
                username: (0, pulumi_1.output)((_c = args.dev.username) !== null && _c !== void 0 ? _c : "default"),
                password: args.dev.password ? (0, pulumi_1.output)(args.dev.password) : undefined,
            };
            new dev_command_js_1.DevCommand("".concat(name, "Dev"), {
                dev: {
                    title: name,
                    autostart: true,
                    command: "sst print-and-not-quit",
                },
                environment: {
                    SST_DEV_COMMAND_MESSAGE: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["Make sure your local Redis server is using:\n\n  username: \"", "\"\n  password: ", "\n\nListening on \"", ":", "\"..."], ["Make sure your local Redis server is using:\n\n  username: \"", "\"\n  password: ", "\n\nListening on \"", ":", "\"..."])), dev.username, dev.password ? "\"".concat(dev.password, "\"") : "\x1b[38;5;8m[no password]\x1b[0m", dev.host, dev.port),
                },
            });
            return dev;
        }
        function normalizeVpc() {
            // "vpc" is a Vpc component
            if (args.vpc instanceof vpc_js_1.Vpc) {
                return (0, pulumi_1.output)({
                    subnets: args.vpc.privateSubnets,
                    securityGroups: args.vpc.securityGroups,
                });
            }
            // "vpc" is object
            return (0, pulumi_1.output)(args.vpc);
        }
        function normalizeCluster() {
            return (0, pulumi_1.all)([args.cluster, args.nodes]).apply(function (_a) {
                var v = _a[0], nodes = _a[1];
                if (v === false)
                    return undefined;
                if (v === true)
                    return { nodes: 1 };
                if (v === undefined) {
                    if (nodes)
                        return { nodes: nodes };
                    return { nodes: 1 };
                }
                return v;
            });
        }
        function createAuthToken() {
            var authToken = new random_1.RandomPassword("".concat(name, "AuthToken"), {
                length: 32,
                special: true,
                overrideSpecial: "!&#$^<>-",
            }, { parent: self }).result;
            var secret = new aws_1.secretsmanager.Secret("".concat(name, "ProxySecret"), {
                recoveryWindowInDays: 0,
            }, { parent: self });
            new aws_1.secretsmanager.SecretVersion("".concat(name, "ProxySecretVersion"), {
                secretId: secret.id,
                secretString: (0, pulumi_1.jsonStringify)({ authToken: authToken }),
            }, { parent: self });
            return { secret: secret, authToken: authToken };
        }
        function createSubnetGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.elasticache.SubnetGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.subnetGroup, "".concat(name, "SubnetGroup"), {
                description: "Managed by SST",
                subnetIds: vpc.subnets,
            }, { parent: self }), false)))();
        }
        function createParameterGroup() {
            var _a;
            var _b, _c;
            return new ((_a = aws_1.elasticache.ParameterGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.parameterGroup, "".concat(name, "ParameterGroup"), {
                description: "Managed by SST",
                family: (0, pulumi_1.all)([engine, version]).apply(function (_a) {
                    var _b;
                    var engine = _a[0], version = _a[1];
                    var majorVersion = version.split(".")[0];
                    var defaultFamily = "".concat(engine).concat(majorVersion);
                    return ((_b = {
                        redis4: "redis4.0",
                        redis5: "redis5.0",
                        redis6: "redis6.x",
                    }[defaultFamily]) !== null && _b !== void 0 ? _b : defaultFamily);
                }),
                parameters: (0, pulumi_1.all)([(_c = args.parameters) !== null && _c !== void 0 ? _c : {}, argsCluster]).apply(function (_a) {
                    var parameters = _a[0], argsCluster = _a[1];
                    return __spreadArray([
                        {
                            name: "cluster-enabled",
                            value: argsCluster ? "yes" : "no",
                        }
                    ], Object.entries(parameters).map(function (_a) {
                        var name = _a[0], value = _a[1];
                        return ({
                            name: name,
                            value: value,
                        });
                    }), true);
                }),
            }, { parent: self }), false)))();
        }
        function createCluster() {
            return argsCluster.apply(function (argsCluster) {
                var _a;
                var _b;
                return new ((_a = aws_1.elasticache.ReplicationGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.cluster, "".concat(name, "Cluster"), __assign(__assign({ description: "Managed by SST", engine: engine, engineVersion: version, nodeType: (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["cache.", ""], ["cache.", ""])), instance), dataTieringEnabled: instance.apply(function (v) {
                        return v.startsWith("r6gd.");
                    }), port: 6379 }, (argsCluster
                    ? {
                        clusterMode: "enabled",
                        numNodeGroups: argsCluster.nodes,
                        replicasPerNodeGroup: 0,
                        automaticFailoverEnabled: true,
                    }
                    : {
                        clusterMode: "disabled",
                    })), { multiAzEnabled: false, atRestEncryptionEnabled: true, transitEncryptionEnabled: true, transitEncryptionMode: "required", authToken: authToken, subnetGroupName: subnetGroup.name, parameterGroupName: parameterGroup.name, securityGroupIds: vpc.securityGroups, tags: {
                        "sst:component-version": _version.toString(),
                        "sst:ref:secret": secret.id,
                    } }), { parent: self }), false)))();
            });
        }
        return _this_1;
    }
    Object.defineProperty(Redis.prototype, "clusterId", {
        /**
         * The ID of the Redis cluster.
         */
        get: function () {
            return this.dev ? (0, pulumi_1.output)("placeholder") : this.cluster.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Redis.prototype, "username", {
        /**
         * The username to connect to the Redis cluster.
         */
        get: function () {
            return this.dev ? this.dev.username : (0, pulumi_1.output)("default");
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Redis.prototype, "password", {
        /**
         * The password to connect to the Redis cluster.
         */
        get: function () {
            var _a;
            return this.dev ? (_a = this.dev.password) !== null && _a !== void 0 ? _a : (0, pulumi_1.output)("") : this._authToken;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Redis.prototype, "host", {
        /**
         * The host to connect to the Redis cluster.
         */
        get: function () {
            var _this_1 = this;
            return this.dev
                ? this.dev.host
                : this.cluster.clusterEnabled.apply(function (enabled) {
                    return enabled
                        ? _this_1.cluster.configurationEndpointAddress
                        : _this_1.cluster.primaryEndpointAddress;
                });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Redis.prototype, "port", {
        /**
         * The port to connect to the Redis cluster.
         */
        get: function () {
            return this.dev ? this.dev.port : this.cluster.port.apply(function (v) { return v; });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Redis.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var _this = this;
            return {
                /**
                 * The ElastiCache Redis cluster.
                 */
                get cluster() {
                    if (_this.dev)
                        throw new error_js_1.VisibleError("Cannot access `nodes.cluster` in dev mode.");
                    return _this.cluster;
                },
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Redis.prototype.getSSTLink = function () {
        return {
            properties: {
                host: this.host,
                port: this.port,
                username: this.username,
                password: this.password,
            },
        };
    };
    /**
     * Reference an existing Redis cluster with the given cluster name. This is useful when you
     * create a Redis cluster in one stage and want to share it in another. It avoids having to
     * create a new Redis cluster in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share Redis clusters across stages.
     * :::
     *
     * @param name The name of the component.
     * @param clusterId The id of the existing Redis cluster.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a cluster in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new cluster, you want to share the same cluster from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const redis = $app.stage === "frank"
     *   ? sst.aws.Redis.get("MyRedis", "app-dev-myredis")
     *   : new sst.aws.Redis("MyRedis");
     * ```
     *
     * Here `app-dev-myredis` is the ID of the cluster created in the `dev` stage.
     * You can find this by outputting the cluster ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   cluster: redis.clusterId
     * };
     * ```
     */
    Redis.get = function (name, clusterId, opts) {
        return new Redis(name, {
            ref: true,
            clusterId: clusterId,
        }, opts);
    };
    Redis.v1 = redis_v1_1.Redis;
    return Redis;
}(component_js_1.Component));
exports.Redis = Redis;
var __pulumiType = "sst:aws:Redis";
// @ts-expect-error
Redis.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2;
