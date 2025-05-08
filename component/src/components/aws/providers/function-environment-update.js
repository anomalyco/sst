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
exports.FunctionEnvironmentUpdate = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var rpc_js_1 = require("../../rpc/rpc.js");
/**
 * The `FunctionEnvironmentUpdate` component is internally used by the `Function` component
 * to update the environment variables of a function.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addEnvironment` method of the `Function` component.
 */
var FunctionEnvironmentUpdate = /** @class */ (function (_super) {
    __extends(FunctionEnvironmentUpdate, _super);
    function FunctionEnvironmentUpdate(name, args, opts) {
        return _super.call(this, new rpc_js_1.rpc.Provider("Aws.FunctionEnvironmentUpdate"), "".concat(name, ".sst.aws.FunctionEnvironmentUpdate"), args, opts) || this;
    }
    return FunctionEnvironmentUpdate;
}(pulumi_1.dynamic.Resource));
exports.FunctionEnvironmentUpdate = FunctionEnvironmentUpdate;
