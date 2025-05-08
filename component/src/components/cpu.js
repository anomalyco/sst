"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toNumber = toNumber;
function toNumber(cpu) {
    var _a = cpu.split(" "), count = _a[0], unit = _a[1];
    var countNum = parseFloat(count);
    if (unit === "vCPU") {
        return countNum * 1024;
    }
    throw new Error("Invalid CPU ".concat(cpu));
}
