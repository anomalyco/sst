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
exports.VisibleError = void 0;
var VisibleError = /** @class */ (function (_super) {
    __extends(VisibleError, _super);
    function VisibleError() {
        var message = [];
        for (var _i = 0; _i < arguments.length; _i++) {
            message[_i] = arguments[_i];
        }
        return _super.call(this, message.join("\n")) || this;
    }
    return VisibleError;
}(Error));
exports.VisibleError = VisibleError;
