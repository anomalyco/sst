"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.lazy = lazy;
function lazy(callback) {
    var loaded = false;
    var result;
    return function () {
        if (!loaded) {
            loaded = true;
            result = callback();
        }
        return result;
    };
}
