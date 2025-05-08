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
exports.KvKeys = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var rpc_js_1 = require("../../rpc/rpc.js");
var KvKeys = /** @class */ (function (_super) {
    __extends(KvKeys, _super);
    function KvKeys(name, args, opts) {
        return _super.call(this, new rpc_js_1.rpc.Provider("Aws.KvKeys"), "".concat(name, ".sst.aws.KvKeys"), args, opts) || this;
    }
    return KvKeys;
}(pulumi_1.dynamic.Resource));
exports.KvKeys = KvKeys;
