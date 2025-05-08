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
exports.Postgres = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var random_1 = require("@pulumi/random");
var vpc_1 = require("./vpc");
var vpc_v1_1 = require("./vpc-v1");
var error_1 = require("../error");
var postgres_v1_1 = require("./postgres-v1");
var size_1 = require("../size");
var dev_command_js_1 = require("../experimental/dev-command.js");
var rds_role_lookup_1 = require("./providers/rds-role-lookup");
/**
 * The `Postgres` component lets you add a Postgres database to your app using
 * [Amazon RDS Postgres](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_PostgreSQL.html).
 *
 * @example
 *
 * #### Create the database
 *
 * ```js title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const database = new sst.aws.Postgres("MyDatabase", { vpc });
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
 * import { Pool } from "pg";
 *
 * const client = new Pool({
 *   user: Resource.MyDatabase.username,
 *   password: Resource.MyDatabase.password,
 *   database: Resource.MyDatabase.database,
 *   host: Resource.MyDatabase.host,
 *   port: Resource.MyDatabase.port,
 * });
 * await client.connect();
 * ```
 *
 * #### Running locally
 *
 * By default, your RDS Postgres database is deployed in `sst dev`. But let's say you are running
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
 * ```ts title="sst.config.ts" {3-8}
 * const postgres = new sst.aws.Postgres("MyPostgres", {
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
 * This will skip deploying an RDS database and link to the locally running Postgres database
 * instead. [Check out the full example](/docs/examples/#aws-postgres-local).
 *
 * ---
 *
 * ### Cost
 *
 * By default this component uses a _Single-AZ Deployment_, _On-Demand DB Instances_ of a
 * `db.t4g.micro` at $0.016 per hour. And 20GB of _General Purpose gp3 Storage_
 * at $0.115 per GB per month.
 *
 * That works out to $0.016 x 24 x 30 + $0.115 x 20 or **$14 per month**. Adjust this for the
 * `instance` type and the `storage` you are using.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [RDS for PostgreSQL pricing](https://aws.amazon.com/rds/postgresql/pricing/#On-Demand_DB_Instances_costs) for more details.
 *
 * #### RDS Proxy
 *
 * If you enable the `proxy`, it uses _Provisioned instances_ with 2 vCPUs at $0.015 per hour.
 *
 * That works out to an **additional** $0.015 x 2 x 24 x 30 or **$22 per month**.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [RDS Proxy pricing](https://aws.amazon.com/rds/proxy/pricing/) for more details.
 */
var Postgres = /** @class */ (function (_super) {
    __extends(Postgres, _super);
    function Postgres(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var _version = 2;
        var self = _this;
        if (args && "ref" in args) {
            var ref = reference();
            _this.instance = ref.instance;
            _this._password = ref.password;
            _this.proxy = (0, pulumi_1.output)(ref.proxy);
            return _this;
        }
        registerVersion();
        var multiAz = (0, pulumi_1.output)(args.multiAz).apply(function (v) { return v !== null && v !== void 0 ? v : false; });
        var engineVersion = (0, pulumi_1.output)(args.version).apply(function (v) { return v !== null && v !== void 0 ? v : "16.4"; });
        var instanceType = (0, pulumi_1.output)(args.instance).apply(function (v) { return v !== null && v !== void 0 ? v : "t4g.micro"; });
        var username = (0, pulumi_1.output)(args.username).apply(function (v) { return v !== null && v !== void 0 ? v : "postgres"; });
        var storage = normalizeStorage();
        var dbName = (0, pulumi_1.output)(args.database).apply(function (v) { return v !== null && v !== void 0 ? v : $app.name.replaceAll("-", "_"); });
        var vpc = normalizeVpc();
        var dev = registerDev();
        if (dev === null || dev === void 0 ? void 0 : dev.enabled) {
            _this.dev = dev;
            return _this;
        }
        var password = createPassword();
        var secret = createSecret();
        var subnetGroup = createSubnetGroup();
        var parameterGroup = createParameterGroup();
        var instance = createInstance();
        createReplicas();
        var proxy = createProxy();
        _this.instance = instance;
        _this._password = password;
        _this.proxy = proxy;
        function reference() {
            var ref = args;
            var instance = aws_1.rds.Instance.get("".concat(name, "Instance"), ref.id, undefined, {
                parent: self,
            });
            var input = instance.tags.apply(function (tags) {
                registerVersion((tags === null || tags === void 0 ? void 0 : tags["sst:component-version"])
                    ? parseInt(tags["sst:component-version"])
                    : undefined);
                return {
                    proxyId: (0, pulumi_1.output)(ref.proxyId),
                    passwordTag: tags === null || tags === void 0 ? void 0 : tags["sst:lookup:password"],
                };
            });
            var proxy = input.proxyId.apply(function (proxyId) {
                return proxyId
                    ? aws_1.rds.Proxy.get("".concat(name, "Proxy"), proxyId, undefined, {
                        parent: self,
                    })
                    : undefined;
            });
            var password = input.passwordTag.apply(function (passwordTag) {
                if (!passwordTag)
                    throw new error_1.VisibleError("Failed to get password for Postgres ".concat(name, "."));
                var secret = aws_1.secretsmanager.getSecretVersionOutput({ secretId: passwordTag }, { parent: self });
                return $jsonParse(secret.secretString).apply(function (v) { return v.password; });
            });
            return { instance: instance, proxy: proxy, password: password };
        }
        function registerVersion(overrideVersion) {
            self.registerVersion({
                new: _version,
                old: overrideVersion !== null && overrideVersion !== void 0 ? overrideVersion : $cli.state.version[name],
                message: [
                    "This component has been renamed. Please change:\n",
                    "\"sst.aws.Postgres\" to \"sst.aws.Postgres.v".concat($cli.state.version[name], "\"\n"),
                    "Learn more https://sst.dev/docs/components/#versioning",
                ].join("\n"),
            });
        }
        function normalizeStorage() {
            var _a;
            return (0, pulumi_1.output)((_a = args.storage) !== null && _a !== void 0 ? _a : "20 GB").apply(function (v) {
                var size = (0, size_1.toGBs)(v);
                if (size < 20) {
                    throw new error_1.VisibleError("Storage must be at least 20 GB for the ".concat(name, " Postgres database."));
                }
                if (size > 65536) {
                    throw new error_1.VisibleError("Storage cannot be greater than 65536 GB (64 TB) for the ".concat(name, " Postgres database."));
                }
                return size;
            });
        }
        function normalizeVpc() {
            // "vpc" is a Vpc.v1 component
            if (args.vpc instanceof vpc_v1_1.Vpc) {
                throw new error_1.VisibleError("You are using the \"Vpc.v1\" component. Please migrate to the latest \"Vpc\" component.");
            }
            // "vpc" is a Vpc component
            if (args.vpc instanceof vpc_1.Vpc) {
                return {
                    subnets: args.vpc.privateSubnets,
                };
            }
            // "vpc" is object
            return (0, pulumi_1.output)(args.vpc);
        }
        function registerDev() {
            var _a, _b, _c, _d;
            if (!args.dev)
                return undefined;
            if ($dev &&
                args.dev.password === undefined &&
                args.password === undefined) {
                throw new error_1.VisibleError("You must provide the password to connect to your locally running Postgres database either by setting the \"dev.password\" or by setting the top-level \"password\" property.");
            }
            var dev = {
                enabled: $dev,
                host: (0, pulumi_1.output)((_a = args.dev.host) !== null && _a !== void 0 ? _a : "localhost"),
                port: (0, pulumi_1.output)((_b = args.dev.port) !== null && _b !== void 0 ? _b : 5432),
                username: args.dev.username ? (0, pulumi_1.output)(args.dev.username) : username,
                password: (0, pulumi_1.output)((_d = (_c = args.dev.password) !== null && _c !== void 0 ? _c : args.password) !== null && _d !== void 0 ? _d : ""),
                database: args.dev.database ? (0, pulumi_1.output)(args.dev.database) : dbName,
            };
            new dev_command_js_1.DevCommand("".concat(name, "Dev"), {
                dev: {
                    title: name,
                    autostart: true,
                    command: "sst print-and-not-quit",
                },
                environment: {
                    SST_DEV_COMMAND_MESSAGE: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["Make sure your local PostgreSQL server is using:\n\n  username: \"", "\"\n  password: \"", "\"\n  database: \"", "\"\n\nListening on \"", ":", "\"..."], ["Make sure your local PostgreSQL server is using:\n\n  username: \"", "\"\n  password: \"", "\"\n  database: \"", "\"\n\nListening on \"", ":", "\"..."])), dev.username, dev.password, dev.database, dev.host, dev.port),
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
        function createSubnetGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.rds.SubnetGroup).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.subnetGroup, "".concat(name, "SubnetGroup"), {
                subnetIds: vpc.subnets,
            }, { parent: self }), false)))();
        }
        function createParameterGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.rds.ParameterGroup).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.parameterGroup, "".concat(name, "ParameterGroup"), {
                family: engineVersion.apply(function (v) { return "postgres".concat(v.split(".")[0]); }),
                parameters: [
                    {
                        name: "rds.force_ssl",
                        value: "0",
                    },
                    {
                        name: "rds.logical_replication",
                        value: "1",
                        applyMethod: "pending-reboot",
                    },
                ],
            }, { parent: self }), false)))();
        }
        function createSecret() {
            var secret = new aws_1.secretsmanager.Secret("".concat(name, "ProxySecret"), {
                recoveryWindowInDays: 0,
            }, { parent: self });
            new aws_1.secretsmanager.SecretVersion("".concat(name, "ProxySecretVersion"), {
                secretId: secret.id,
                secretString: (0, pulumi_1.jsonStringify)({
                    username: username,
                    password: password,
                }),
            }, { parent: self });
            return secret;
        }
        function createInstance() {
            var _a;
            var _b;
            return new ((_a = aws_1.rds.Instance).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.instance, "".concat(name, "Instance"), {
                dbName: dbName,
                dbSubnetGroupName: subnetGroup.name,
                engine: "postgres",
                engineVersion: engineVersion,
                instanceClass: (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["db.", ""], ["db.", ""])), instanceType),
                username: username,
                password: password,
                parameterGroupName: parameterGroup.name,
                skipFinalSnapshot: true,
                storageEncrypted: true,
                storageType: "gp3",
                allocatedStorage: 20,
                maxAllocatedStorage: storage,
                multiAz: multiAz,
                backupRetentionPeriod: 7,
                performanceInsightsEnabled: true,
                tags: {
                    "sst:component-version": _version.toString(),
                    "sst:lookup:password": secret.id,
                },
            }, { parent: self, deleteBeforeReplace: true }), false)))();
        }
        function createReplicas() {
            var _a;
            return (0, pulumi_1.output)((_a = args.replicas) !== null && _a !== void 0 ? _a : 0).apply(function (replicas) {
                return Array.from({ length: replicas }).map(function (_, i) {
                    return new aws_1.rds.Instance("".concat(name, "Replica").concat(i), {
                        replicateSourceDb: instance.identifier,
                        dbName: (0, pulumi_1.interpolate)(templateObject_3 || (templateObject_3 = __makeTemplateObject(["", "_replica", ""], ["", "_replica", ""])), instance.dbName, i),
                        dbSubnetGroupName: instance.dbSubnetGroupName,
                        availabilityZone: instance.availabilityZone,
                        engine: instance.engine,
                        engineVersion: instance.engineVersion,
                        instanceClass: instance.instanceClass,
                        username: instance.username,
                        password: instance.password.apply(function (v) { return v; }),
                        parameterGroupName: instance.parameterGroupName,
                        skipFinalSnapshot: true,
                        storageEncrypted: instance.storageEncrypted.apply(function (v) { return v; }),
                        storageType: instance.storageType,
                        allocatedStorage: instance.allocatedStorage,
                        maxAllocatedStorage: instance.maxAllocatedStorage.apply(function (v) { return v; }),
                    }, { parent: self });
                });
            });
        }
        function createProxy() {
            return (0, pulumi_1.output)(args.proxy).apply(function (proxy) {
                var _a;
                var _b, _c;
                if (!proxy)
                    return;
                var credentials = proxy === true ? [] : (_b = proxy.credentials) !== null && _b !== void 0 ? _b : [];
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
                var lookup = new rds_role_lookup_1.RdsRoleLookup("".concat(name, "ProxyRoleLookup"), { name: "AWSServiceRoleForRDS" }, { parent: self });
                var rdsProxy = new ((_a = aws_1.rds.Proxy).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.proxy, "".concat(name, "Proxy"), {
                    engineFamily: "POSTGRESQL",
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
                var targetGroup = new aws_1.rds.ProxyDefaultTargetGroup("".concat(name, "ProxyTargetGroup"), {
                    dbProxyName: rdsProxy.name,
                }, { parent: self });
                new aws_1.rds.ProxyTarget("".concat(name, "ProxyTarget"), {
                    dbProxyName: rdsProxy.name,
                    targetGroupName: targetGroup.name,
                    dbInstanceIdentifier: instance.identifier,
                }, { parent: self });
                return rdsProxy;
            });
        }
        return _this;
    }
    Object.defineProperty(Postgres.prototype, "id", {
        /**
         * The identifier of the Postgres instance.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return (0, pulumi_1.output)("placeholder");
            return this.instance.identifier;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "proxyId", {
        /**
         * The name of the Postgres proxy.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return (0, pulumi_1.output)("placeholder");
            return this.proxy.apply(function (v) {
                if (!v) {
                    throw new error_1.VisibleError("Proxy is not enabled. Enable it with \"proxy: true\".");
                }
                return v.id;
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "username", {
        /** The username of the master user. */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.username;
            return this.instance.username;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "password", {
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
    Object.defineProperty(Postgres.prototype, "database", {
        /**
         * The name of the database.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.database;
            return this.instance.dbName;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "port", {
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
    Object.defineProperty(Postgres.prototype, "host", {
        /**
         * The host of the database.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.host;
            return (0, pulumi_1.all)([this.instance.endpoint, this.proxy]).apply(function (_a) {
                var _b;
                var endpoint = _a[0], proxy = _a[1];
                return (_b = proxy === null || proxy === void 0 ? void 0 : proxy.endpoint) !== null && _b !== void 0 ? _b : (0, pulumi_1.output)(endpoint.split(":")[0]);
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "nodes", {
        get: function () {
            return {
                instance: this.instance,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Postgres.prototype.getSSTLink = function () {
        return {
            properties: {
                database: this.database,
                username: this.username,
                password: this.password,
                port: this.port,
                host: this.host,
            },
        };
    };
    /**
     * Reference an existing Postgres database with the given name. This is useful when you
     * create a Postgres database in one stage and want to share it in another. It avoids
     * having to create a new Postgres database in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share Postgres databases across stages.
     * :::
     *
     * @param name The name of the component.
     * @param args The arguments to get the Postgres database.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a database in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new database, you want to share the same database from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const database = $app.stage === "frank"
     *   ? sst.aws.Postgres.get("MyDatabase", {
     *       id: "app-dev-mydatabase",
     *       proxyId: "app-dev-mydatabase-proxy",
     *     })
     *   : new sst.aws.Postgres("MyDatabase", {
     *       proxy: true,
     *     });
     * ```
     *
     * Here `app-dev-mydatabase` is the ID of the database, and `app-dev-mydatabase-proxy`
     * is the ID of the proxy created in the `dev` stage. You can find these by outputting
     * the database ID and proxy ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   id: database.id,
     *   proxyId: database.proxyId,
     * };
     * ```
     */
    Postgres.get = function (name, args, opts) {
        return new Postgres(name, {
            ref: true,
            id: args.id,
            proxyId: args.proxyId,
        }, opts);
    };
    Postgres.v1 = postgres_v1_1.Postgres;
    return Postgres;
}(component_1.Component));
exports.Postgres = Postgres;
var __pulumiType = "sst:aws:Postgres";
// @ts-expect-error
Postgres.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2, templateObject_3;
