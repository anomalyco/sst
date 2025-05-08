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
exports.Queue = void 0;
var cloudflare = require("@pulumi/cloudflare");
var component_1 = require("../component");
var binding_1 = require("./binding");
var account_id_1 = require("./account-id");
/**
 * The `Queue` component lets you add a [Cloudflare Queue](https://developers.cloudflare.com/queues/) to
 * your app.
 */
var Queue = /** @class */ (function (_super) {
    __extends(Queue, _super);
    function Queue(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var queue = create();
        _this.queue = queue;
        function create() {
            var _a;
            var _b;
            return new ((_a = cloudflare.Queue).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.queue, "".concat(name, "Queue"), {
                name: "",
                accountId: account_id_1.DEFAULT_ACCOUNT_ID,
            }, { parent: parent }), false)))();
        }
        return _this;
    }
    Queue.prototype.getSSTLink = function () {
        return {
            properties: {},
            include: [
                (0, binding_1.binding)({
                    type: "queueBindings",
                    properties: {
                        queue: this.queue.name,
                    },
                }),
            ],
        };
    };
    Object.defineProperty(Queue.prototype, "id", {
        /**
         * The generated id of the queue
         */
        get: function () {
            return this.queue.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Queue.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Cloudflare queue.
                 */
                queue: this.queue,
            };
        },
        enumerable: false,
        configurable: true
    });
    return Queue;
}(component_1.Component));
exports.Queue = Queue;
var __pulumiType = "sst:cloudflare:Queue";
// @ts-expect-error
Queue.__pulumiType = __pulumiType;
