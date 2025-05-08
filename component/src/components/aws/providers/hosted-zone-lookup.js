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
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.HostedZoneLookup = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var rpc_js_1 = require("../../rpc/rpc.js");
var HostedZoneLookup = /** @class */ (function (_super) {
    __extends(HostedZoneLookup, _super);
    function HostedZoneLookup(name, args, opts) {
        return _super.call(this, new rpc_js_1.rpc.Provider("Aws.HostedZoneLookup"), "".concat(name, ".sst.aws.HostedZoneLookup"), __assign(__assign({}, args), { zoneId: undefined }), opts) || this;
    }
    return HostedZoneLookup;
}(pulumi_1.dynamic.Resource));
exports.HostedZoneLookup = HostedZoneLookup;
