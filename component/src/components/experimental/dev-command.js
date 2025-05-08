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
Object.defineProperty(exports, "__esModule", { value: true });
exports.DevCommand = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var link_js_1 = require("../link.js");
/**
 * The `DevCommand` lets you run a command in a separate pane when you run `sst dev`.
 *
 * :::note
 * This is an experimental feature and the API may change in the future.
 * :::
 *
 * The `sst dev` CLI starts a multiplexer with panes for separate processes. This component allows you to add a process to it.
 *
 * :::tip
 * This component does not do anything on deploy.
 * :::
 *
 * This component only works in `sst dev`. It does not do anything in `sst deploy`.
 *
 * #### Example
 *
 * For example, you can use this to run Drizzle Studio locally.
 *
 * ```ts title="sst.config.ts"
 * new sst.x.DevCommand("Studio", {
 *   link: [rds],
 *   dev: {
 *     autostart: true,
 *     command: "npx drizzle-kit studio",
 *   },
 * });
 * ```
 *
 * Here `npx drizzle-kit studio` will be run in `sst dev` and will show up under the **Studio** tab. It'll also have access to the links from `rds`.
 */
var DevCommand = /** @class */ (function (_super) {
    __extends(DevCommand, _super);
    function DevCommand(name, args, opts) {
        var _a, _b, _c, _d, _e;
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        _this.registerOutputs({
            _dev: {
                links: (0, pulumi_1.output)(args.link || [])
                    .apply(link_js_1.Link.build)
                    .apply(function (links) { return links.map(function (link) { return link.name; }); }),
                environment: args.environment,
                title: (_a = args.dev) === null || _a === void 0 ? void 0 : _a.title,
                directory: (_b = args.dev) === null || _b === void 0 ? void 0 : _b.directory,
                autostart: ((_c = args.dev) === null || _c === void 0 ? void 0 : _c.autostart) !== false,
                command: (_d = args.dev) === null || _d === void 0 ? void 0 : _d.command,
                aws: {
                    role: (_e = args.aws) === null || _e === void 0 ? void 0 : _e.role,
                },
            },
        });
        return _this;
    }
    return DevCommand;
}(component_1.Component));
exports.DevCommand = DevCommand;
var __pulumiType = "sst:sst:DevCommand";
// @ts-expect-error
DevCommand.__pulumiType = __pulumiType;
