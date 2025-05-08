"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.warnOnce = warnOnce;
var alreadyWarned = new Set();
function warnOnce(message) {
    if (alreadyWarned.has(message))
        return;
    alreadyWarned.add(message);
    console.warn(message);
}
