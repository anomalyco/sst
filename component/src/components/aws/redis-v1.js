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
exports.Redis = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var random_1 = require("@pulumi/random");
var component_js_1 = require("../component.js");
var aws_1 = require("@pulumi/aws");
var vpc_js_1 = require("./vpc.js");
var error_js_1 = require("../error.js");
var dev_command_js_1 = require("../experimental/dev-command.js");
/**
 * The `Redis` component lets you add a Redis cluster to your app using
 * [Amazon ElastiCache](https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/WhatIs.html).
 *
 * For existing usage, rename `sst.aws.Redis` to `sst.aws.Redis.v1`. For new Redis, use
 * the latest [`Redis`](/docs/component/aws/redis) component instead.
 *
 * :::caution
 * This component has been deprecated.
 * :::
 *
 * What changed:
 * - In this version, the Redis/Valkey cluster uses the default parameter group, which
 * cannot be customized.
 * - In the new version, the cluster now creates a custom parameter group. This allows
 * you to customize the parameters via the `transform` prop.
 *
 * @example
 *
 * #### Create the cluster
 *
 * ```js title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const redis = new sst.aws.Redis.v1("MyRedis", { vpc });
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
 * const redis = new sst.aws.Redis.v1("MyRedis", {
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
        if (args && "ref" in args) {
            var ref = args;
            _this_1.cluster = ref.cluster;
            _this_1._authToken = ref.authToken;
            return _this_1;
        }
        var parent = _this_1;
        var engine = (0, pulumi_1.output)(args.engine).apply(function (v) { return v !== null && v !== void 0 ? v : "redis"; });
        var version = (0, pulumi_1.all)([engine, args.version]).apply(function (_a) {
            var engine = _a[0], v = _a[1];
            return v !== null && v !== void 0 ? v : (engine === "redis" ? "7.1" : "7.2");
        });
        var instance = (0, pulumi_1.output)(args.instance).apply(function (v) { return v !== null && v !== void 0 ? v : "t4g.micro"; });
        var nodes = (0, pulumi_1.output)(args.nodes).apply(function (v) { return v !== null && v !== void 0 ? v : 1; });
        var vpc = normalizeVpc();
        var dev = registerDev();
        if (dev === null || dev === void 0 ? void 0 : dev.enabled) {
            _this_1.dev = dev;
            return _this_1;
        }
        var _a = createAuthToken(), authToken = _a.authToken, secret = _a.secret;
        var subnetGroup = createSubnetGroup();
        var cluster = createCluster();
        _this_1.cluster = cluster;
        _this_1._authToken = authToken;
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
        function createAuthToken() {
            var authToken = new random_1.RandomPassword("".concat(name, "AuthToken"), {
                length: 32,
                special: true,
                overrideSpecial: "!&#$^<>-",
            }, { parent: parent }).result;
            var secret = new aws_1.secretsmanager.Secret("".concat(name, "ProxySecret"), {
                recoveryWindowInDays: 0,
            }, { parent: parent });
            new aws_1.secretsmanager.SecretVersion("".concat(name, "ProxySecretVersion"), {
                secretId: secret.id,
                secretString: (0, pulumi_1.jsonStringify)({ authToken: authToken }),
            }, { parent: parent });
            return { secret: secret, authToken: authToken };
        }
        function createSubnetGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.elasticache.SubnetGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.subnetGroup, "".concat(name, "SubnetGroup"), {
                description: "Managed by SST",
                subnetIds: vpc.subnets,
            }, { parent: parent }), false)))();
        }
        function createCluster() {
            var _a;
            var _b;
            return new ((_a = aws_1.elasticache.ReplicationGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.cluster, "".concat(name, "Cluster"), {
                description: "Managed by SST",
                engine: engine,
                engineVersion: version,
                nodeType: (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["cache.", ""], ["cache.", ""])), instance),
                dataTieringEnabled: instance.apply(function (v) { return v.startsWith("r6gd."); }),
                port: 6379,
                automaticFailoverEnabled: true,
                clusterMode: "enabled",
                numNodeGroups: nodes,
                replicasPerNodeGroup: 0,
                multiAzEnabled: false,
                atRestEncryptionEnabled: true,
                transitEncryptionEnabled: true,
                transitEncryptionMode: "required",
                authToken: authToken,
                subnetGroupName: subnetGroup.name,
                securityGroupIds: vpc.securityGroups,
                tags: {
                    "sst:auth-token-ref": secret.id,
                },
            }, { parent: parent }), false)))();
        }
        return _this_1;
    }
    Object.defineProperty(Redis.prototype, "clusterID", {
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
            return this.dev
                ? this.dev.host
                : this.cluster.configurationEndpointAddress;
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
     * @param clusterID The id of the existing Redis cluster.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a cluster in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new cluster, you want to share the same cluster from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const redis = $app.stage === "frank"
     *   ? sst.aws.Redis.v1.get("MyRedis", "app-dev-myredis")
     *   : new sst.aws.Redis.v1("MyRedis");
     * ```
     *
     * Here `app-dev-myredis` is the ID of the cluster created in the `dev` stage.
     * You can find this by outputting the cluster ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   cluster: redis.clusterID
     * };
     * ```
     */
    Redis.get = function (name, clusterID, opts) {
        var cluster = aws_1.elasticache.ReplicationGroup.get("".concat(name, "Cluster"), clusterID, undefined, opts);
        var secret = cluster.tags.apply(function (tags) {
            return (tags === null || tags === void 0 ? void 0 : tags["sst:auth-token-ref"])
                ? aws_1.secretsmanager.getSecretVersionOutput({
                    secretId: tags["sst:auth-token-ref"],
                }, opts)
                : (0, pulumi_1.output)(undefined);
        });
        var authToken = secret.apply(function (v) {
            if (!v)
                throw new error_js_1.VisibleError("Failed to get auth token for Redis ".concat(name, "."));
            return JSON.parse(v.secretString).authToken;
        });
        return new Redis(name, {
            ref: true,
            cluster: cluster,
            authToken: authToken,
        });
    };
    return Redis;
}(component_js_1.Component));
exports.Redis = Redis;
var __pulumiType = "sst:aws:Redis";
// @ts-expect-error
Redis.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2;
