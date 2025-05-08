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
var component_js_1 = require("../component.js");
var aws_1 = require("@pulumi/aws");
var permission_js_1 = require("./permission.js");
function parseACU(acu) {
    var result = parseFloat(acu.split(" ")[0]);
    return result;
}
/**
 * The `Postgres` component lets you add a Postgres database to your app using
 * [Amazon Aurora Serverless v2](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora-serverless-v2.html).
 *
 * For existing usage, rename `sst.aws.Postgres` to `sst.aws.Postgres.v1`. For new Postgres, use
 * the latest [`Postgres`](/docs/component/aws/postgres) component instead.
 *
 * :::caution
 * This component has been deprecated.
 * :::
 *
 * What changed:
 * - In this version, the database used AWS RDS Aurora Serverless v2, which supported RDS
 * Data API. This allowed your machine to connect to the database during "sst dev" without
 * the need for a VPN.
 * - In the new version, the database now uses AWS RDS Postgres. The "sst.aws.Vpc" component
 * has been enhanced to set up a secure tunnel, enabling seamlessly connections to the
 * database. Postgres provides greater flexibility and wider feature support while being
 * cheaper to run.
 *
 * :::note
 * Data API for Aurora Postgres Serverless v2 is still being [rolled out in all regions](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/Concepts.Aurora_Fea_Regions_DB-eng.Feature.ServerlessV2.html#Concepts.Aurora_Fea_Regions_DB-eng.Feature.ServerlessV2.apg).
 * :::
 *
 * To connect to your database from your Lambda functions, you can use the
 * [AWS Data API](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/data-api.html). It
 * does not need a persistent connection, and works over HTTP. You also don't need a VPN to
 * connect to it locally.
 *
 * @example
 *
 * #### Create the database
 *
 * ```js title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const database = new sst.aws.Postgres.v1("MyDatabase", { vpc });
 * ```
 *
 * #### Change the scaling config
 *
 * ```js title="sst.config.ts"
 * new sst.aws.Postgres.v1("MyDatabase", {
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
 */
var Postgres = /** @class */ (function (_super) {
    __extends(Postgres, _super);
    function Postgres(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        if (args && "ref" in args) {
            var ref = args;
            _this.cluster = ref.cluster;
            _this.instance = ref.instance;
            return _this;
        }
        var parent = _this;
        var scaling = normalizeScaling();
        var version = normalizeVersion();
        var databaseName = normalizeDatabaseName();
        var subnetGroup = createSubnetGroup();
        var cluster = createCluster();
        var instance = createInstance();
        _this.cluster = cluster;
        _this.instance = instance;
        function normalizeScaling() {
            return (0, pulumi_1.output)(args.scaling).apply(function (scaling) {
                var _a, _b;
                return ({
                    minCapacity: parseACU((_a = scaling === null || scaling === void 0 ? void 0 : scaling.min) !== null && _a !== void 0 ? _a : "0.5 ACU"),
                    maxCapacity: parseACU((_b = scaling === null || scaling === void 0 ? void 0 : scaling.max) !== null && _b !== void 0 ? _b : "4 ACU"),
                });
            });
        }
        function normalizeVersion() {
            return (0, pulumi_1.output)(args.version).apply(function (version) { return version !== null && version !== void 0 ? version : "15.5"; });
        }
        function normalizeDatabaseName() {
            return (0, pulumi_1.output)(args.databaseName).apply(function (name) { return name !== null && name !== void 0 ? name : $app.name.replaceAll("-", "_"); });
        }
        function createSubnetGroup() {
            var _a;
            var _b;
            if (args.vpc === "default")
                return;
            return new ((_a = aws_1.rds.SubnetGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.subnetGroup, "".concat(name, "SubnetGroup"), {
                subnetIds: (0, pulumi_1.output)(args.vpc).privateSubnets,
            }, { parent: parent }), false)))();
        }
        function createCluster() {
            var _a;
            var _b;
            return new ((_a = aws_1.rds.Cluster).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.cluster, "".concat(name, "Cluster"), {
                engine: aws_1.rds.EngineType.AuroraPostgresql,
                engineMode: "provisioned",
                engineVersion: version,
                databaseName: databaseName,
                masterUsername: "postgres",
                manageMasterUserPassword: true,
                serverlessv2ScalingConfiguration: scaling,
                skipFinalSnapshot: true,
                enableHttpEndpoint: true,
                dbSubnetGroupName: subnetGroup === null || subnetGroup === void 0 ? void 0 : subnetGroup.name,
                vpcSecurityGroupIds: args.vpc === "default"
                    ? undefined
                    : (0, pulumi_1.output)(args.vpc).securityGroups,
            }, { parent: parent }), false)))();
        }
        function createInstance() {
            var _a;
            var _b;
            return new ((_a = aws_1.rds.ClusterInstance).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.instance, "".concat(name, "Instance"), {
                clusterIdentifier: cluster.id,
                instanceClass: "db.serverless",
                engine: aws_1.rds.EngineType.AuroraPostgresql,
                engineVersion: cluster.engineVersion,
                dbSubnetGroupName: subnetGroup === null || subnetGroup === void 0 ? void 0 : subnetGroup.name,
            }, { parent: parent }), false)))();
        }
        return _this;
    }
    Object.defineProperty(Postgres.prototype, "secret", {
        get: function () {
            var _this = this;
            return this.secretArn.apply(function (val) {
                if (_this._dbSecret)
                    return _this._dbSecret;
                if (!val)
                    return;
                _this._dbSecret = aws_1.secretsmanager.getSecretVersionOutput({
                    secretId: val,
                });
                return _this._dbSecret;
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "clusterID", {
        /**
         * The ID of the RDS Cluster.
         */
        get: function () {
            return this.cluster.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "clusterArn", {
        /**
         * The ARN of the RDS Cluster.
         */
        get: function () {
            return this.cluster.arn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "secretArn", {
        /**
         * The ARN of the master user secret.
         */
        get: function () {
            return this.cluster.masterUserSecrets[0].secretArn;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "username", {
        /** The username of the master user. */
        get: function () {
            return this.cluster.masterUsername;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "password", {
        /** The password of the master user. */
        get: function () {
            var _this = this;
            return this.cluster.masterPassword.apply(function (val) {
                if (val)
                    return (0, pulumi_1.output)(val);
                var parsed = (0, pulumi_1.jsonParse)(_this.secret.apply(function (secret) {
                    return secret ? secret.secretString : (0, pulumi_1.output)("{}");
                }));
                return parsed.password;
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "database", {
        /**
         * The name of the database.
         */
        get: function () {
            return this.cluster.databaseName;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "port", {
        /**
         * The port of the database.
         */
        get: function () {
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
            return this.instance.endpoint;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Postgres.prototype, "nodes", {
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
    Postgres.prototype.getSSTLink = function () {
        return {
            properties: {
                clusterArn: this.clusterArn,
                secretArn: this.secretArn,
                database: this.cluster.databaseName,
                username: this.username,
                password: this.password,
                port: this.port,
                host: this.host,
            },
            include: [
                (0, permission_js_1.permission)({
                    actions: ["secretsmanager:GetSecretValue"],
                    resources: [
                        this.cluster.masterUserSecrets[0].secretArn.apply(function (v) { return v !== null && v !== void 0 ? v : "arn:aws:iam::rdsdoesnotusesecretmanager"; }),
                    ],
                }),
                (0, permission_js_1.permission)({
                    actions: [
                        "rds-data:BatchExecuteStatement",
                        "rds-data:BeginTransaction",
                        "rds-data:CommitTransaction",
                        "rds-data:ExecuteStatement",
                        "rds-data:RollbackTransaction",
                    ],
                    resources: [this.cluster.arn],
                }),
            ],
        };
    };
    /**
     * Reference an existing Postgres cluster with the given cluster name. This is useful when you
     * create a Postgres cluster in one stage and want to share it in another. It avoids having to
     * create a new Postgres cluster in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share Postgres clusters across stages.
     * :::
     *
     * @param name The name of the component.
     * @param clusterID The id of the existing Postgres cluster.
     *
     * @example
     * Imagine you create a cluster in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new cluster, you want to share the same cluster from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const database = $app.stage === "frank"
     *   ? sst.aws.Postgres.v1.get("MyDatabase", "app-dev-mydatabase")
     *   : new sst.aws.Postgres.v1("MyDatabase");
     * ```
     *
     * Here `app-dev-mydatabase` is the ID of the cluster created in the `dev` stage.
     * You can find this by outputting the cluster ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   cluster: database.clusterID
     * };
     * ```
     */
    Postgres.get = function (name, clusterID) {
        var cluster = aws_1.rds.Cluster.get("".concat(name, "Cluster"), clusterID);
        var instances = aws_1.rds.getInstancesOutput({
            filters: [{ name: "db-cluster-id", values: [clusterID] }],
        });
        var instance = aws_1.rds.ClusterInstance.get("".concat(name, "Instance"), instances.apply(function (instances) {
            if (instances.instanceIdentifiers.length === 0)
                throw new Error("No instance found for cluster ".concat(clusterID));
            return instances.instanceIdentifiers[0];
        }));
        return new Postgres(name, {
            ref: true,
            cluster: cluster,
            instance: instance,
        });
    };
    return Postgres;
}(component_js_1.Component));
exports.Postgres = Postgres;
var __pulumiType = "sst:aws:Postgres";
// @ts-expect-error
Postgres.__pulumiType = __pulumiType;
