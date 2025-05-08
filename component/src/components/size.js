"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toMBs = toMBs;
exports.toGBs = toGBs;
function toMBs(size) {
    var _a = size.split(" "), count = _a[0], unit = _a[1];
    var countNum = parseFloat(count);
    if (unit === "MB") {
        return countNum;
    }
    else if (unit === "GB") {
        return countNum * 1024;
    }
    else if (unit === "TB") {
        return countNum * 1024 * 1024;
    }
    throw new Error("Invalid size ".concat(size));
}
function toGBs(size) {
    var _a = size.split(" "), count = _a[0], unit = _a[1];
    var countNum = parseFloat(count);
    if (unit === "MB") {
        return countNum / 1024;
    }
    else if (unit === "GB") {
        return countNum;
    }
    else if (unit === "TB") {
        return countNum * 1024;
    }
    throw new Error("Invalid size ".concat(size));
}
