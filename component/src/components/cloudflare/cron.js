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
exports.Cron = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var cloudflare = require("@pulumi/cloudflare");
var component_1 = require("../component");
var worker_1 = require("./worker");
var account_id_js_1 = require("./account-id.js");
var worker_builder_1 = require("./helpers/worker-builder");
/**
 * The `Cron` component lets you add cron jobs to your app using Cloudflare.
 * It uses [Cloudflare Cron Triggers](https://developers.cloudflare.com/workers/configuration/cron-triggers/).
 *
 * @example
 * #### Minimal example
 *
 * Create a worker file that exposes a `scheduled` handler:
 *
 * ```ts title="cron.ts"
 * export default {
 *   async scheduled() {
 *     console.log("Running on a schedule");
 *   },
 * };
 * ```
 *
 * Pass in a `schedules` and a `job` worker that'll be executed.
 *
 * ```ts title="sst.config.ts"
 * new sst.cloudflare.Cron("MyCronJob", {
 *   job: "cron.ts",
 *   schedules: ["* * * * *"]
 * });
 * ```
 *
 * #### Customize the function
 *
 * ```js title="sst.config.ts"
 * new sst.cloudflare.Cron("MyCronJob", {
 *   schedules: ["* * * * *"],
 *   job: {
 *     handler: "cron.ts",
 *     link: [bucket]
 *   }
 * });
 * ```
 */
var Cron = /** @class */ (function (_super) {
    __extends(Cron, _super);
    function Cron(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var worker = createWorker();
        var trigger = createTrigger();
        _this.worker = worker;
        _this.trigger = trigger;
        function createWorker() {
            return (0, worker_builder_1.workerBuilder)("".concat(name, "Handler"), args.job);
        }
        function createTrigger() {
            return (0, pulumi_1.all)([args.schedules]).apply(function (_a) {
                var _b;
                var _c;
                var schedules = _a[0];
                return new ((_b = cloudflare.WorkerCronTrigger).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.trigger, "".concat(name, "Trigger"), {
                    accountId: account_id_js_1.DEFAULT_ACCOUNT_ID,
                    scriptName: worker.script.name,
                    schedules: schedules,
                }, { parent: parent }), false)))();
            });
        }
        return _this;
    }
    Object.defineProperty(Cron.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Cloudflare Worker.
                 */
                worker: this.worker.script,
                /**
                 * The Cloudflare Worker Cron Trigger.
                 */
                trigger: this.trigger,
            };
        },
        enumerable: false,
        configurable: true
    });
    return Cron;
}(component_1.Component));
exports.Cron = Cron;
var __pulumiType = "sst:cloudflare:Cron";
// @ts-expect-error
worker_1.Worker.__pulumiType = __pulumiType;
