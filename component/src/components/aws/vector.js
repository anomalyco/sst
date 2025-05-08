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
exports.Vector = void 0;
var path_1 = require("path");
var component_js_1 = require("../component.js");
var postgres_v1_js_1 = require("./postgres-v1.js");
var vector_table_js_1 = require("./providers/vector-table.js");
var function_js_1 = require("./function.js");
var permission_js_1 = require("./permission.js");
/**
 * The `Vector` component lets you store and retrieve vector data in your app.
 *
 * - It uses a vector database powered by [RDS Postgres Serverless v2](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora-serverless-v2.html).
 * - Provides a [SDK](/docs/reference/sdk/) to query, put, and remove the vector data.
 *
 * @example
 *
 * #### Create the database
 *
 * ```ts title="sst.config.ts"
 * const vector = new sst.aws.Vector("MyVectorDB", {
 *   dimension: 1536
 * });
 * ```
 *
 * #### Link to a resource
 *
 * You can link it to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [vector]
 * });
 * ```
 *
 * Once linked, you can query it in your function code using the [SDK](/docs/reference/sdk/).
 *
 * ```ts title="app/page.tsx"
 * import { VectorClient } from "sst";
 *
 * await VectorClient("MyVectorDB").query({
 *   vector: [32.4, 6.55, 11.2, 10.3, 87.9]
 * });
 * ```
 */
var Vector = /** @class */ (function (_super) {
    __extends(Vector, _super);
    function Vector(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var tableName = normalizeTableName();
        var postgres;
        if (args && "ref" in args) {
            var ref = args;
            postgres = ref.postgres;
        }
        else {
            postgres = createDB();
            createDBTable();
        }
        var queryHandler = createQueryHandler();
        var putHandler = createPutHandler();
        var removeHandler = createRemoveHandler();
        _this.postgres = postgres;
        _this.queryHandler = queryHandler;
        _this.putHandler = putHandler;
        _this.removeHandler = removeHandler;
        function normalizeTableName() {
            return "embeddings";
        }
        function createDB() {
            var _a;
            return new (postgres_v1_js_1.Postgres.bind.apply(postgres_v1_js_1.Postgres, __spreadArray([void 0], (0, component_js_1.transform)((_a = args === null || args === void 0 ? void 0 : args.transform) === null || _a === void 0 ? void 0 : _a.postgres, "".concat(name, "Database"), { vpc: "default" }, { parent: parent }), false)))();
        }
        function createDBTable() {
            new vector_table_js_1.VectorTable("".concat(name, "Table"), {
                clusterArn: postgres.nodes.cluster.arn,
                secretArn: postgres.nodes.cluster.masterUserSecrets[0].secretArn,
                databaseName: postgres.database,
                tableName: tableName,
                dimension: args.dimension,
            }, { parent: parent, dependsOn: postgres.nodes.instance });
        }
        function createQueryHandler() {
            return new function_js_1.Function("".concat(name, "Query"), {
                description: "".concat(name, " query handler"),
                bundle: useBundlePath(),
                handler: "index.query",
                environment: useHandlerEnvironment(),
                permissions: useHandlerPermissions(),
                dev: false,
            }, { parent: parent });
        }
        function createPutHandler() {
            return new function_js_1.Function("".concat(name, "Put"), {
                description: "".concat(name, " put handler"),
                bundle: useBundlePath(),
                handler: "index.put",
                environment: useHandlerEnvironment(),
                permissions: useHandlerPermissions(),
                dev: false,
            }, { parent: parent });
        }
        function createRemoveHandler() {
            return new function_js_1.Function("".concat(name, "Remove"), {
                description: "".concat(name, " remove handler"),
                bundle: useBundlePath(),
                handler: "index.remove",
                environment: useHandlerEnvironment(),
                permissions: useHandlerPermissions(),
                dev: false,
            }, { parent: parent });
        }
        function useBundlePath() {
            return path_1.default.join($cli.paths.platform, "dist", "vector-handler");
        }
        function useHandlerEnvironment() {
            return {
                CLUSTER_ARN: postgres.nodes.cluster.arn,
                SECRET_ARN: postgres.nodes.cluster.masterUserSecrets[0].secretArn,
                DATABASE_NAME: postgres.database,
                TABLE_NAME: tableName,
            };
        }
        function useHandlerPermissions() {
            return [
                {
                    actions: ["secretsmanager:GetSecretValue"],
                    resources: [postgres.nodes.cluster.masterUserSecrets[0].secretArn],
                },
                {
                    actions: ["rds-data:ExecuteStatement"],
                    resources: [postgres.nodes.cluster.arn],
                },
            ];
        }
        return _this;
    }
    /**
     * Reference an existing Vector database with the given name. This is useful when you
     * create a Vector database in one stage and want to share it in another. It avoids having to
     * create a new Vector database in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share Vector databases across stages.
     * :::
     *
     * @param name The name of the component.
     * @param clusterID The RDS cluster id of the existing Vector database.
     *
     * @example
     * Imagine you create a vector database  in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new database, you want to share the same database from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const vector = $app.stage === "frank"
     *   ? sst.aws.Vector.get("MyVectorDB", "app-dev-myvectordb")
     *   : new sst.aws.Vector("MyVectorDB", {
     *       dimension: 1536
     *     });
     * ```
     *
     * Here `app-dev-myvectordb` is the ID of the underlying Postgres cluster created in the `dev` stage.
     * You can find this by outputting the cluster ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   cluster: vector.clusterID
     * };
     * ```
     *
     * :::note
     * The Vector component creates a Postgres cluster and lambda functions for interfacing with the VectorDB.
     * The `static get` method only shares the underlying Postgres cluster. Each stage will have its own
     * lambda functions.
     * :::
     */
    Vector.get = function (name, clusterID) {
        var postgres = postgres_v1_js_1.Postgres.get("".concat(name, "Database"), clusterID);
        return new Vector(name, {
            ref: true,
            postgres: postgres,
        });
    };
    Object.defineProperty(Vector.prototype, "clusterID", {
        /**
         * The ID of the RDS Postgres Cluster.
         */
        get: function () {
            return this.postgres.nodes.cluster.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Vector.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Postgres database.
                 */
                postgres: this.postgres,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Vector.prototype.getSSTLink = function () {
        return {
            properties: {
                /** @internal */
                queryFunction: this.queryHandler.name,
                /** @internal */
                putFunction: this.putHandler.name,
                /** @internal */
                removeFunction: this.removeHandler.name,
            },
            include: [
                (0, permission_js_1.permission)({
                    actions: ["lambda:InvokeFunction"],
                    resources: [
                        this.queryHandler.nodes.function.arn,
                        this.putHandler.nodes.function.arn,
                        this.removeHandler.nodes.function.arn,
                    ],
                }),
            ],
        };
    };
    return Vector;
}(component_js_1.Component));
exports.Vector = Vector;
var __pulumiType = "sst:aws:Vector";
// @ts-expect-error
Vector.__pulumiType = __pulumiType;
