"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toSeconds = toSeconds;
function toSeconds(duration) {
    var _a = duration.split(" "), count = _a[0], unit = _a[1];
    var countNum = parseInt(count);
    var unitLower = unit.toLowerCase();
    if (unitLower.startsWith("second")) {
        return countNum;
    }
    else if (unitLower.startsWith("minute")) {
        return countNum * 60;
    }
    else if (unitLower.startsWith("hour")) {
        return countNum * 3600;
    }
    else if (unitLower.startsWith("day")) {
        return countNum * 86400;
    }
    throw new Error("Invalid duration ".concat(duration));
}
