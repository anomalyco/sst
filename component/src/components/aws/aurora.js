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
exports.Aurora = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_js_1 = require("../component.js");
var aws_1 = require("@pulumi/aws");
var error_js_1 = require("../error.js");
var vpc_js_1 = require("./vpc.js");
var random_1 = require("@pulumi/random");
var dev_command_js_1 = require("../experimental/dev-command.js");
var rds_role_lookup_js_1 = require("./providers/rds-role-lookup.js");
var duration_js_1 = require("../duration.js");
var permission_js_1 = require("./permission.js");
function parseACU(acu) {
    var result = parseFloat(acu.split(" ")[0]);
    return result;
}
/**
 * The `Aurora` component lets you add a Aurora Postgres or MySQL cluster to your app
 * using [Amazon Aurora Serverless v2](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora-serverless-v2.html).
 *
 * @example
 *
 * #### Create an Aurora Postgres cluster
 *
 * ```js title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const database = new sst.aws.Aurora("MyDatabase", {
 *   engine: "postgres",
 *   vpc
 * });
 * ```
 *
 * #### Create an Aurora MySQL cluster
 *
 * ```js title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const database = new sst.aws.Aurora("MyDatabase", {
 *   engine: "mysql",
 *   vpc
 * });
 * ```
 *
 * #### Change the scaling config
 *
 * ```js title="sst.config.ts"
 * new sst.aws.Aurora("MyDatabase", {
 *   engine: "postgres",
 *   scaling: {
 *     min: "2 ACU",
 *     max: "128 ACU"
 *   },
 *   vpc
 * });
 * ```
 *
 * #### Link to a resource
 *
 * You can link your database to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [database],
 *   vpc
 * });
 * ```
 *
 * Once linked, you can connect to it from your function code.
 *
 * ```ts title="app/page.tsx" {1,5-9}
 * import { Resource } from "sst";
 * import postgres from "postgres";
 *
 * const sql = postgres({
 *   username: Resource.MyDatabase.username,
 *   password: Resource.MyDatabase.password,
 *   database: Resource.MyDatabase.database,
 *   host: Resource.MyDatabase.host,
 *   port: Resource.MyDatabase.port
 * });
 * ```
 *
 * #### Enable the RDS Data API
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Aurora("MyDatabase", {
 *   engine: "postgres",
 *   dataApi: true,
 *   vpc
 * });
 * ```
 *
 * When using the Data API, connecting to the database does not require a persistent
 * connection, and works over HTTP. You also don't need the `sst tunnel` or a VPN to connect
 * to it from your local machine.
 *
 * ```ts title="app/page.tsx" {1,6,7,8}
 * import { Resource } from "sst";
 * import { drizzle } from "drizzle-orm/aws-data-api/pg";
 * import { RDSDataClient } from "@aws-sdk/client-rds-data";
 *
 * drizzle(new RDSDataClient({}), {
 *   database: Resource.MyDatabase.database,
 *   secretArn: Resource.MyDatabase.secretArn,
 *   resourceArn: Resource.MyDatabase.clusterArn
 * });
 * ```
 *
 * #### Running locally
 *
 * By default, your Aurora database is deployed in `sst dev`. But let's say you are running
 * Postgres locally.
 *
 * ```bash
 * docker run \
 *   --rm \
 *   -p 5432:5432 \
 *   -v $(pwd)/.sst/storage/postgres:/var/lib/postgresql/data \
 *   -e POSTGRES_USER=postgres \
 *   -e POSTGRES_PASSWORD=password \
 *   -e POSTGRES_DB=local \
 *   postgres:16.4
 * ```
 *
 * You can connect to it in `sst dev` by configuring the `dev` prop.
 *
 * ```ts title="sst.config.ts" {4-9}
 * new sst.aws.Aurora("MyDatabase", {
 *   engine: "postgres",
 *   vpc,
 *   dev: {
 *     username: "postgres",
 *     password: "password",
 *     database: "local",
 *     port: 5432
 *   }
 * });
 * ```
 *
 * This will skip deploying the database and link to the locally running Postgres database
 * instead. [Check out the full example](/docs/examples/#aws-aurora-local).
 *
 * ---
 *
 * ### Cost
 *
 * This component has one DB instance that is used for both writes and reads. The
 * instance can scale from the minimum number of ACUs to the maximum number of ACUs. By default,
 * this uses a `min` of 0 ACUs and a `max` of 4 ACUs.
 *
 * When the database is paused, you are not charged for the ACUs.
 *
 * Each ACU costs $0.12 per hour for both `postgres` and `mysql` engine. The storage costs
 * $0.01 per GB per month for standard storage.
 *
 * So if your database is constantly using 1GB of memory or 0.5 ACUs, then you are charged
 * $0.12 x 0.5 x 24 x 30 or **$43 per month**. And add the storage costs to this as well.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [Amazon Aurora pricing](https://aws.amazon.com/rds/aurora/pricing) for more details.
 *
 * #### RDS Proxy
 *
 * If you enable the `proxy`, it uses _Aurora Capacity Units_ with a minumum of 8 ACUs at
 * $0.015 per ACU hour.
 *
 * That works out to an **additional** $0.015 x 8 x 24 x 30 or **$86 per month**. Adjust
 * this if you end up using more than 8 ACUs.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [RDS Proxy pricing](https://aws.amazon.com/rds/proxy/pricing/) for more details.
 *
 * #### RDS Data API
 *
 * If you enable `dataApi`, you get charged an **additional** $0.35 per million requests for
 * the first billion requests. After that, it's $0.20 per million requests.
 *
 * Check out the [RDS Data API pricing](https://aws.amazon.com/rds/aurora/pricing/#Data_API_costs)
 * for more details.
 */
var Aurora = /** @class */ (function (_super) {
    __extends(Aurora, _super);
    function Aurora(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        if (args && "ref" in args) {
            var ref = reference();
            _this.cluster = ref.cluster;
            _this.instance = ref.instance;
            _this._password = ref.password;
            _this.proxy = (0, pulumi_1.output)(ref.proxy);
            _this.secret = ref.secret;
            return _this;
        }
        var engine = (0, pulumi_1.output)(args.engine);
        var version = (0, pulumi_1.all)([args.version, engine]).apply(function (_a) {
            var version = _a[0], engine = _a[1];
            return version !== null && version !== void 0 ? version : { postgres: "16.4", mysql: "3.08.0" }[engine];
        });
        var username = (0, pulumi_1.all)([args.username, engine]).apply(function (_a) {
            var username = _a[0], engine = _a[1];
            return username !== null && username !== void 0 ? username : { postgres: "postgres", mysql: "root" }[engine];
        });
        var dbName = (0, pulumi_1.output)(args.database).apply(function (name) { return name !== null && name !== void 0 ? name : $app.name.replaceAll("-", "_"); });
        var dataApi = (0, pulumi_1.output)(args.dataApi).apply(function (v) { return v !== null && v !== void 0 ? v : false; });
        var scaling = normalizeScaling();
        var replicas = normalizeReplicas();
        var vpc = normalizeVpc();
        var dev = registerDev();
        if (dev === null || dev === void 0 ? void 0 : dev.enabled) {
            _this.dev = dev;
            return _this;
        }
        var password = createPassword();
        var secret = createSecret();
        var subnetGroup = createSubnetGroup();
        var instanceParameterGroup = createInstanceParameterGroup();
        var clusterParameterGroup = createClusterParameterGroup();
        var proxy = createProxy();
        var cluster = createCluster();
        var instance = createInstances();
        createProxyTarget();
        _this.cluster = cluster;
        _this.instance = instance;
        _this.secret = secret;
        _this._password = password;
        _this.proxy = proxy;
        function reference() {
            var ref = args;
            var cluster = aws_1.rds.Cluster.get("".concat(name, "Cluster"), ref.id, undefined, {
                parent: self,
            });
            var instance = aws_1.rds.ClusterInstance.get("".concat(name, "Instance"), aws_1.rds
                .getInstancesOutput({
                filters: [
                    {
                        name: "db-cluster-id",
                        values: [cluster.id],
                    },
                ],
            }, { parent: self })
                .instanceIdentifiers.apply(function (ids) {
                if (!ids.length) {
                    throw new error_js_1.VisibleError("Database instance not found in cluster ".concat(cluster.id));
                }
                return ids[0];
            }), undefined, { parent: self });
            var secretId = cluster.tags
                .apply(function (tags) { return tags === null || tags === void 0 ? void 0 : tags["sst:ref:password"]; })
                .apply(function (passwordTag) {
                if (!passwordTag)
                    throw new error_js_1.VisibleError("Failed to get password for Postgres ".concat(name, "."));
                return passwordTag;
            });
            var secret = aws_1.secretsmanager.Secret.get("".concat(name, "ProxySecret"), secretId, undefined, { parent: self });
            var secretVersion = aws_1.secretsmanager.getSecretVersionOutput({ secretId: secretId }, { parent: self });
            var password = $jsonParse(secretVersion.secretString).apply(function (v) { return v.password; });
            var proxy = cluster.tags
                .apply(function (tags) { return tags === null || tags === void 0 ? void 0 : tags["sst:ref:proxy"]; })
                .apply(function (proxyTag) {
                return proxyTag
                    ? aws_1.rds.Proxy.get("".concat(name, "Proxy"), proxyTag, undefined, {
                        parent: self,
                    })
                    : undefined;
            });
            return { cluster: cluster, instance: instance, proxy: proxy, password: password, secret: secret };
        }
        function normalizeScaling() {
            return (0, pulumi_1.output)(args.scaling).apply(function (scaling) {
                var _a, _b, _c;
                var max = (_a = scaling === null || scaling === void 0 ? void 0 : scaling.max) !== null && _a !== void 0 ? _a : "4 ACU";
                var min = (_b = scaling === null || scaling === void 0 ? void 0 : scaling.min) !== null && _b !== void 0 ? _b : "0 ACU";
                var isAutoPauseEnabled = parseACU(min) === 0;
                if ((scaling === null || scaling === void 0 ? void 0 : scaling.pauseAfter) && !isAutoPauseEnabled) {
                    throw new error_js_1.VisibleError("Cannot configure \"pauseAfter\" when the minimum ACU is not 0 for the \"".concat(name, "\" Aurora database."));
                }
                return {
                    max: max,
                    min: min,
                    pauseAfter: isAutoPauseEnabled
                        ? (_c = scaling === null || scaling === void 0 ? void 0 : scaling.pauseAfter) !== null && _c !== void 0 ? _c : "5 minutes"
                        : undefined,
                };
            });
        }
        function normalizeReplicas() {
            var _a;
            return (0, pulumi_1.output)((_a = args.replicas) !== null && _a !== void 0 ? _a : 0).apply(function (replicas) {
                if (replicas > 15) {
                    throw new error_js_1.VisibleError("Cannot create more than 15 read-only replicas for the \"".concat(name, "\" Aurora database."));
                }
                return replicas;
            });
        }
        function normalizeVpc() {
            // "vpc" is a Vpc component
            if (args.vpc instanceof vpc_js_1.Vpc) {
                return {
                    subnets: args.vpc.privateSubnets,
                    securityGroups: args.vpc.securityGroups,
                };
            }
            // "vpc" is object
            return (0, pulumi_1.output)(args.vpc);
        }
        function registerDev() {
            var _a, _b, _c;
            if (!args.dev)
                return undefined;
            if ($dev &&
                args.dev.password === undefined &&
                args.password === undefined) {
                throw new error_js_1.VisibleError("You must provide the password to connect to your locally running database either by setting the \"dev.password\" or by setting the top-level \"password\" property.");
            }
            var dev = {
                enabled: $dev,
                host: (0, pulumi_1.output)((_a = args.dev.host) !== null && _a !== void 0 ? _a : "localhost"),
                port: (0, pulumi_1.all)([args.dev.port, engine]).apply(function (_a) {
                    var port = _a[0], engine = _a[1];
                    return port !== null && port !== void 0 ? port : { postgres: 5432, mysql: 3306 }[engine];
                }),
                username: args.dev.username ? (0, pulumi_1.output)(args.dev.username) : username,
                password: (0, pulumi_1.output)((_c = (_b = args.dev.password) !== null && _b !== void 0 ? _b : args.password) !== null && _c !== void 0 ? _c : ""),
                database: args.dev.database ? (0, pulumi_1.output)(args.dev.database) : dbName,
            };
            new dev_command_js_1.DevCommand("".concat(name, "Dev"), {
                dev: {
                    title: name,
                    autostart: true,
                    command: "sst print-and-not-quit",
                },
                environment: {
                    SST_DEV_COMMAND_MESSAGE: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["Make sure your local database is using:\n\n  username: \"", "\"\n  password: \"", "\"\n  database: \"", "\"\n\nListening on \"", ":", "\"..."], ["Make sure your local database is using:\n\n  username: \"", "\"\n  password: \"", "\"\n  database: \"", "\"\n\nListening on \"", ":", "\"..."])), dev.username, dev.password, dev.database, dev.host, dev.port),
                },
            });
            return dev;
        }
        function createPassword() {
            return args.password
                ? (0, pulumi_1.output)(args.password)
                : new random_1.RandomPassword("".concat(name, "Password"), {
                    length: 32,
                    special: false,
                }, { parent: self }).result;
        }
        function createSecret() {
            var secret = new aws_1.secretsmanager.Secret("".concat(name, "ProxySecret"), {
                recoveryWindowInDays: 0,
            }, { parent: self });
            new aws_1.secretsmanager.SecretVersion("".concat(name, "ProxySecretVersion"), {
                secretId: secret.id,
                secretString: (0, pulumi_1.jsonStringify)({ username: username, password: password }),
            }, { parent: self });
            return secret;
        }
        function createSubnetGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.rds.SubnetGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.subnetGroup, "".concat(name, "SubnetGroup"), {
                subnetIds: vpc.subnets,
            }, { parent: self }), false)))();
        }
        function createInstanceParameterGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.rds.ParameterGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.instanceParameterGroup, "".concat(name, "ParameterGroup"), {
                family: (0, pulumi_1.all)([engine, version]).apply(function (_a) {
                    var engine = _a[0], version = _a[1];
                    if (engine === "postgres")
                        return "aurora-postgresql".concat(version.split(".")[0]);
                    return version.startsWith("2")
                        ? "aurora-mysql5.7"
                        : "aurora-mysql8.0";
                }),
                parameters: [],
            }, { parent: self }), false)))();
        }
        function createClusterParameterGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.rds.ClusterParameterGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.clusterParameterGroup, "".concat(name, "ClusterParameterGroup"), {
                family: (0, pulumi_1.all)([engine, version]).apply(function (_a) {
                    var engine = _a[0], version = _a[1];
                    if (engine === "postgres")
                        return "aurora-postgresql".concat(version.split(".")[0]);
                    return version.startsWith("2")
                        ? "aurora-mysql5.7"
                        : "aurora-mysql8.0";
                }),
                parameters: [],
            }, { parent: self }), false)))();
        }
        function createCluster() {
            var _a;
            var _b;
            return new ((_a = aws_1.rds.Cluster).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.cluster, "".concat(name, "Cluster"), {
                engine: engine.apply(function (engine) {
                    return engine === "postgres"
                        ? aws_1.rds.EngineType.AuroraPostgresql
                        : aws_1.rds.EngineType.AuroraMysql;
                }),
                engineMode: "provisioned",
                engineVersion: (0, pulumi_1.all)([engine, version]).apply(function (_a) {
                    var engine = _a[0], version = _a[1];
                    if (engine === "postgres")
                        return version;
                    return version.startsWith("2")
                        ? "5.7.mysql_aurora.".concat(version)
                        : "8.0.mysql_aurora.".concat(version);
                }),
                databaseName: dbName,
                masterUsername: username,
                masterPassword: password,
                dbClusterParameterGroupName: clusterParameterGroup.name,
                dbInstanceParameterGroupName: instanceParameterGroup.name,
                serverlessv2ScalingConfiguration: scaling.apply(function (scaling) { return ({
                    maxCapacity: parseACU(scaling.max),
                    minCapacity: parseACU(scaling.min),
                    secondsUntilAutoPause: scaling.pauseAfter
                        ? (0, duration_js_1.toSeconds)(scaling.pauseAfter)
                        : undefined,
                }); }),
                skipFinalSnapshot: true,
                storageEncrypted: true,
                enableHttpEndpoint: dataApi,
                dbSubnetGroupName: subnetGroup === null || subnetGroup === void 0 ? void 0 : subnetGroup.name,
                vpcSecurityGroupIds: vpc.securityGroups,
                tags: proxy.apply(function (proxy) { return (__assign({ "sst:ref:password": secret.id }, (proxy ? { "sst:ref:proxy": proxy.id } : {}))); }),
            }, { parent: self }), false)))();
        }
        function createInstances() {
            var _a;
            var _b;
            var props = {
                clusterIdentifier: cluster.id,
                instanceClass: "db.serverless",
                engine: cluster.engine.apply(function (v) { return v; }),
                engineVersion: cluster.engineVersion,
                dbSubnetGroupName: cluster.dbSubnetGroupName,
                dbParameterGroupName: instanceParameterGroup.name,
            };
            // Create primary instance
            var instance = new ((_a = aws_1.rds.ClusterInstance).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.instance, "".concat(name, "Instance"), props, {
                parent: self,
            }), false)))();
            // Create replicas
            replicas.apply(function (replicas) {
                var _a;
                var _b;
                for (var i = 0; i < replicas; i++) {
                    new ((_a = aws_1.rds.ClusterInstance).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.instance, "".concat(name, "Replica").concat(i), __assign(__assign({}, props), { promotionTier: 15 }), { parent: self }), false)))();
                }
            });
            return instance;
        }
        function createProxy() {
            return (0, pulumi_1.all)([args.proxy]).apply(function (_a) {
                var _b;
                var _c, _d;
                var proxy = _a[0];
                if (!proxy)
                    return;
                var credentials = proxy === true ? [] : (_c = proxy.credentials) !== null && _c !== void 0 ? _c : [];
                // Create secrets
                var secrets = credentials.map(function (credential) {
                    var secret = new aws_1.secretsmanager.Secret("".concat(name, "ProxySecret").concat(credential.username), {
                        recoveryWindowInDays: 0,
                    }, { parent: self });
                    new aws_1.secretsmanager.SecretVersion("".concat(name, "ProxySecretVersion").concat(credential.username), {
                        secretId: secret.id,
                        secretString: (0, pulumi_1.jsonStringify)({
                            username: credential.username,
                            password: credential.password,
                        }),
                    }, { parent: self });
                    return secret;
                });
                var role = new aws_1.iam.Role("".concat(name, "ProxyRole"), {
                    assumeRolePolicy: aws_1.iam.assumeRolePolicyForPrincipal({
                        Service: "rds.amazonaws.com",
                    }),
                    inlinePolicies: [
                        {
                            name: "inline",
                            policy: aws_1.iam.getPolicyDocumentOutput({
                                statements: [
                                    {
                                        actions: ["secretsmanager:GetSecretValue"],
                                        resources: __spreadArray([secret.arn], secrets.map(function (s) { return s.arn; }), true),
                                    },
                                ],
                            }).json,
                        },
                    ],
                }, { parent: self });
                var lookup = new rds_role_lookup_js_1.RdsRoleLookup("".concat(name, "ProxyRoleLookup"), { name: "AWSServiceRoleForRDS" }, { parent: self });
                return new ((_b = aws_1.rds.Proxy).bind.apply(_b, __spreadArray([void 0], (0, component_js_1.transform)((_d = args.transform) === null || _d === void 0 ? void 0 : _d.proxy, "".concat(name, "Proxy"), {
                    engineFamily: engine.apply(function (engine) {
                        return engine === "postgres" ? "POSTGRESQL" : "MYSQL";
                    }),
                    auths: __spreadArray([
                        {
                            authScheme: "SECRETS",
                            iamAuth: "DISABLED",
                            secretArn: secret.arn,
                        }
                    ], secrets.map(function (s) { return ({
                        authScheme: "SECRETS",
                        iamAuth: "DISABLED",
                        secretArn: s.arn,
                    }); }), true),
                    roleArn: role.arn,
                    vpcSubnetIds: vpc.subnets,
                }, { parent: self, dependsOn: [lookup] }), false)))();
            });
        }
        function createProxyTarget() {
            proxy.apply(function (proxy) {
                if (!proxy)
                    return;
                var targetGroup = new aws_1.rds.ProxyDefaultTargetGroup("".concat(name, "ProxyTargetGroup"), {
                    dbProxyName: proxy.name,
                }, { parent: self });
                new aws_1.rds.ProxyTarget("".concat(name, "ProxyTarget"), {
                    dbProxyName: proxy.name,
                    targetGroupName: targetGroup.name,
                    dbClusterIdentifier: cluster.clusterIdentifier,
                }, { parent: self });
            });
        }
        return _this;
    }
    Object.defineProperty(Aurora.prototype, "id", {
        /**
         * The ID of the RDS Cluster.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return (0, pulumi_1.output)("placeholder");
            return this.cluster.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Aurora.prototype, "clusterArn", {
        /**
         * The ARN of the RDS Cluster.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return (0, pulumi_1.output)("placeholder");
            return this.cluster.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Aurora.prototype, "secretArn", {
        /**
         * The ARN of the master user secret.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return (0, pulumi_1.output)("placeholder");
            return this.secret.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Aurora.prototype, "username", {
        /** The username of the master user. */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.username;
            return this.cluster.masterUsername;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Aurora.prototype, "password", {
        /** The password of the master user. */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.password;
            return this._password;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Aurora.prototype, "database", {
        /**
         * The name of the database.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.database;
            return this.cluster.databaseName;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Aurora.prototype, "port", {
        /**
         * The port of the database.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.port;
            return this.instance.port;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Aurora.prototype, "host", {
        /**
         * The host of the database.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.host;
            return (0, pulumi_1.all)([this.cluster.endpoint, this.proxy]).apply(function (_a) {
                var _b;
                var endpoint = _a[0], proxy = _a[1];
                return (_b = proxy === null || proxy === void 0 ? void 0 : proxy.endpoint) !== null && _b !== void 0 ? _b : (0, pulumi_1.output)(endpoint.split(":")[0]);
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Aurora.prototype, "reader", {
        /**
         * The reader endpoint of the database.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.host;
            return (0, pulumi_1.all)([this.cluster.readerEndpoint, this.proxy]).apply(function (_a) {
                var endpoint = _a[0], proxy = _a[1];
                if (proxy) {
                    throw new error_js_1.VisibleError("Reader endpoint is not currently supported for RDS Proxy. Please contact us on Discord or open a GitHub issue.");
                }
                return (0, pulumi_1.output)(endpoint.split(":")[0]);
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Aurora.prototype, "nodes", {
        get: function () {
            return {
                cluster: this.cluster,
                instance: this.instance,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Aurora.prototype.getSSTLink = function () {
        var _a, _b;
        return {
            properties: {
                clusterArn: this.clusterArn,
                secretArn: this.secretArn,
                database: this.database,
                username: this.username,
                password: this.password,
                port: this.port,
                host: this.host,
                reader: ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                    ? this.dev.host
                    : (0, pulumi_1.all)([this.cluster.readerEndpoint, this.proxy]).apply(function (_a) {
                        var endpoint = _a[0], proxy = _a[1];
                        if (proxy)
                            return (0, pulumi_1.output)(undefined);
                        return (0, pulumi_1.output)(endpoint.split(":")[0]);
                    }),
            },
            include: ((_b = this.dev) === null || _b === void 0 ? void 0 : _b.enabled)
                ? []
                : [
                    (0, permission_js_1.permission)({
                        actions: ["secretsmanager:GetSecretValue"],
                        resources: [this.secretArn],
                    }),
                    (0, permission_js_1.permission)({
                        actions: [
                            "rds-data:BatchExecuteStatement",
                            "rds-data:BeginTransaction",
                            "rds-data:CommitTransaction",
                            "rds-data:ExecuteStatement",
                            "rds-data:RollbackTransaction",
                        ],
                        resources: [this.clusterArn],
                    }),
                ],
        };
    };
    /**
     * Reference an existing Aurora cluster with its RDS cluster ID. This is useful when you
     * create a Aurora cluster in one stage and want to share it in another. It avoids having to
     * create a new Aurora cluster in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share Aurora clusters across stages.
     * :::
     *
     * @param name The name of the component.
     * @param id The ID of the existing Aurora cluster.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a cluster in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new cluster, you want to share the same cluster from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const database = $app.stage === "frank"
     *   ? sst.aws.Aurora.get("MyDatabase", "app-dev-mydatabase")
     *   : new sst.aws.Aurora("MyDatabase");
     * ```
     *
     * Here `app-dev-mydatabase` is the ID of the cluster created in the `dev` stage.
     * You can find this by outputting the cluster ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return database.id;
     * ```
     */
    Aurora.get = function (name, id, opts) {
        return new Aurora(name, {
            ref: true,
            id: id,
        }, opts);
    };
    return Aurora;
}(component_js_1.Component));
exports.Aurora = Aurora;
var __pulumiType = "sst:aws:Aurora";
// @ts-expect-error
Aurora.__pulumiType = __pulumiType;
var templateObject_1;
