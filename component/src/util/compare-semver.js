"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.compareSemver = compareSemver;
exports.isALteB = isALteB;
exports.isALtB = isALtB;
function compareSemver(v1, v2) {
    if (v1 === "latest")
        return 1;
    if (/^[^\d]/.test(v1)) {
        v1 = v1.substring(1);
    }
    if (/^[^\d]/.test(v2)) {
        v2 = v2.substring(1);
    }
    var _a = v1.split(".").map(Number), major1 = _a[0], minor1 = _a[1], patch1 = _a[2];
    var _b = v2.split(".").map(Number), major2 = _b[0], minor2 = _b[1], patch2 = _b[2];
    if (major1 !== major2)
        return major1 - major2;
    if (minor1 !== minor2)
        return minor1 - minor2;
    return patch1 - patch2;
}
function isALteB(a, b) {
    return compareSemver(a, b) <= 0;
}
function isALtB(a, b) {
    return compareSemver(a, b) < 0;
}
