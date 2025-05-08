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
exports.Bucket = void 0;
var cloudflare = require("@pulumi/cloudflare");
var component_1 = require("../component");
var binding_js_1 = require("./binding.js");
var account_id_1 = require("./account-id");
/**
 * The `Bucket` component lets you add a [Cloudflare R2 Bucket](https://developers.cloudflare.com/r2/) to
 * your app.
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * const bucket = new sst.cloudflare.Bucket("MyBucket");
 * ```
 *
 * #### Link to a worker
 *
 * You can link the bucket to a worker.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "./index.ts",
 *   link: [bucket],
 *   url: true
 * });
 * ```
 *
 * Once linked, you can use the SDK to interact with the bucket.
 *
 * ```ts title="index.ts" {3}
 * import { Resource } from "sst";
 *
 * await Resource.MyBucket.list();
 * ```
 */
var Bucket = /** @class */ (function (_super) {
    __extends(Bucket, _super);
    function Bucket(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var bucket = createBucket();
        _this.bucket = bucket;
        function createBucket() {
            var _a;
            var _b;
            return new ((_a = cloudflare.R2Bucket).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.bucket, "".concat(name, "Bucket"), {
                name: "",
                accountId: account_id_1.DEFAULT_ACCOUNT_ID,
            }, { parent: parent }), false)))();
        }
        return _this;
    }
    /**
     * When you link a bucket to a worker, you can interact with it using these
     * [Bucket methods](https://developers.cloudflare.com/r2/api/workers/workers-api-reference/#bucket-method-definitions).
     *
     * @example
     * ```ts title="index.ts" {3}
     * import { Resource } from "sst";
     *
     * await Resource.MyBucket.list();
     * ```
     *
     * @internal
     */
    Bucket.prototype.getSSTLink = function () {
        return {
            properties: {},
            include: [
                (0, binding_js_1.binding)({
                    type: "r2BucketBindings",
                    properties: {
                        bucketName: this.bucket.name,
                    },
                }),
            ],
        };
    };
    Object.defineProperty(Bucket.prototype, "name", {
        /**
         * The generated name of the R2 Bucket.
         */
        get: function () {
            return this.bucket.name;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Bucket.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Cloudflare R2 Bucket.
                 */
                bucket: this.bucket,
            };
        },
        enumerable: false,
        configurable: true
    });
    return Bucket;
}(component_1.Component));
exports.Bucket = Bucket;
var __pulumiType = "sst:cloudflare:Bucket";
// @ts-expect-error
Bucket.__pulumiType = __pulumiType;
