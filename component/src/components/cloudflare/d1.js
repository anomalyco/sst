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
exports.D1 = void 0;
var cloudflare = require("@pulumi/cloudflare");
var component_1 = require("../component");
var binding_1 = require("./binding");
var _1 = require(".");
/**
 * The `D1` component lets you add a [Cloudflare D1 database](https://developers.cloudflare.com/d1/) to
 * your app.
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * const db = new sst.cloudflare.D1("MyDatabase");
 * ```
 *
 * #### Link to a worker
 *
 * You can link the db to a worker.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "./index.ts",
 *   link: [db],
 *   url: true
 * });
 * ```
 *
 * Once linked, you can use the SDK to interact with the db.
 *
 * ```ts title="index.ts" {1} "Resource.MyDatabase.prepare"
 * import { Resource } from "sst";
 *
 * await Resource.MyDatabase.prepare(
 *   "SELECT id FROM todo ORDER BY id DESC LIMIT 1",
 * ).first();
 * ```
 */
var D1 = /** @class */ (function (_super) {
    __extends(D1, _super);
    function D1(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var db = createDB();
        _this.database = db;
        function createDB() {
            var _a;
            var _b;
            return new ((_a = cloudflare.D1Database).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.database, "".concat(name, "Database"), {
                name: "",
                accountId: _1.DEFAULT_ACCOUNT_ID,
            }, { parent: parent }), false)))();
        }
        return _this;
    }
    /**
     * When you link a D1 database, the database will be available to the worker and you can
     * query it using its [API methods](https://developers.cloudflare.com/d1/build-with-d1/d1-client-api/).
     *
     * @example
     * ```ts title="index.ts" {1} "Resource.MyDatabase.prepare"
     * import { Resource } from "sst";
     *
     * await Resource.MyDatabase.prepare(
     *   "SELECT id FROM todo ORDER BY id DESC LIMIT 1",
     * ).first();
     * ```
     *
     * @internal
     */
    D1.prototype.getSSTLink = function () {
        return {
            properties: {
                databaseId: this.database.id,
            },
            include: [
                (0, binding_1.binding)({
                    type: "d1DatabaseBindings",
                    properties: {
                        databaseId: this.database.id,
                    },
                }),
            ],
        };
    };
    Object.defineProperty(D1.prototype, "databaseId", {
        /**
         * The generated ID of the D1 database.
         */
        get: function () {
            return this.database.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(D1.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Cloudflare D1 database.
                 */
                database: this.database,
            };
        },
        enumerable: false,
        configurable: true
    });
    return D1;
}(component_1.Component));
exports.D1 = D1;
var __pulumiType = "sst:cloudflare:D1";
// @ts-expect-error
D1.__pulumiType = __pulumiType;
