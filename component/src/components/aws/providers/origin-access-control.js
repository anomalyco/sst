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
exports.OriginAccessControl = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var rpc_js_1 = require("../../rpc/rpc.js");
var OriginAccessControl = /** @class */ (function (_super) {
    __extends(OriginAccessControl, _super);
    function OriginAccessControl(name, args, opts) {
        return _super.call(this, new rpc_js_1.rpc.Provider("Aws.OriginAccessControl"), "".concat(name, ".sst.aws.OriginAccessControl"), args, opts) || this;
    }
    return OriginAccessControl;
}(pulumi_1.dynamic.Resource));
exports.OriginAccessControl = OriginAccessControl;
