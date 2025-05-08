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
exports.Kv = void 0;
var cloudflare = require("@pulumi/cloudflare");
var component_1 = require("../component");
var binding_1 = require("./binding");
var account_id_1 = require("./account-id");
/**
 * The `Kv` component lets you add a [Cloudflare KV storage namespace](https://developers.cloudflare.com/kv/) to
 * your app.
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * const storage = new sst.cloudflare.Kv("MyStorage");
 * ```
 *
 * #### Link to a worker
 *
 * You can link KV to a worker.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "./index.ts",
 *   link: [storage],
 *   url: true
 * });
 * ```
 *
 * Once linked, you can use the SDK to interact with the bucket.
 *
 * ```ts title="index.ts" {3}
 * import { Resource } from "sst";
 *
 * await Resource.MyStorage.get("someKey");
 * ```
 */
var Kv = /** @class */ (function (_super) {
    __extends(Kv, _super);
    function Kv(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var namespace = createNamespace();
        _this.namespace = namespace;
        function createNamespace() {
            var _a;
            var _b;
            return new ((_a = cloudflare.WorkersKvNamespace).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.namespace, "".concat(name, "Namespace"), {
                title: "",
                accountId: account_id_1.DEFAULT_ACCOUNT_ID,
            }, { parent: parent }), false)))();
        }
        return _this;
    }
    /**
     * When you link a KV storage, the storage will be available to the worker and you can
     * interact with it using its [API methods](https://developers.cloudflare.com/kv/api/).
     *
     * @example
     * ```ts title="index.ts" {3}
     * import { Resource } from "sst";
     *
     * await Resource.MyStorage.get("someKey");
     * ```
     *
     * @internal
     */
    Kv.prototype.getSSTLink = function () {
        return {
            properties: {},
            include: [
                (0, binding_1.binding)({
                    type: "kvNamespaceBindings",
                    properties: {
                        namespaceId: this.namespace.id,
                    },
                }),
            ],
        };
    };
    Object.defineProperty(Kv.prototype, "id", {
        /**
         * The generated ID of the KV namespace.
         */
        get: function () {
            return this.namespace.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Kv.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Cloudflare KV namespace.
                 */
                namespace: this.namespace,
            };
        },
        enumerable: false,
        configurable: true
    });
    return Kv;
}(component_1.Component));
exports.Kv = Kv;
var __pulumiType = "sst:cloudflare:Kv";
// @ts-expect-error
Kv.__pulumiType = __pulumiType;
