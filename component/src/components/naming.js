"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.PRETTY_CHARS = void 0;
exports.logicalName = logicalName;
exports.physicalName = physicalName;
exports.prefixName = prefixName;
exports.hashNumberToPrettyString = hashNumberToPrettyString;
exports.hashStringToPrettyString = hashStringToPrettyString;
var crypto_1 = require("crypto");
function logicalName(name) {
    name = name.replace(/[^a-zA-Z0-9]/g, "");
    return name.charAt(0).toUpperCase() + name.slice(1);
}
function physicalName(max, name, suffix) {
    if (suffix === void 0) { suffix = ""; }
    // This function does the following:
    // - Removes all non-alphanumeric characters
    // - Prefixes the name with the app name and stage
    // - Truncates the name if it's too long
    // - Adds a random suffix
    // - Adds a suffix if provided
    var main = prefixName(max - 9 - suffix.length, name);
    var random = hashStringToPrettyString(crypto_1.default.randomBytes(8).toString("hex"), 8);
    return "".concat(main, "-").concat(random).concat(suffix);
}
function prefixName(max, name) {
    // This function does the following:
    // - Removes all non-alphanumeric characters
    // - Prefixes the name with the app name and stage
    // - Truncates the name if it's too long
    // ie. foo => app-stage-foo
    name = name.replace(/[^a-zA-Z0-9]/g, "");
    var stageLen = $app.stage.length;
    var nameLen = name.length;
    var strategy = nameLen + 1 >= max
        ? "name"
        : nameLen + stageLen + 2 >= max
            ? "stage+name"
            : "app+stage+name";
    if (strategy === "name")
        return "".concat(name.substring(0, max));
    if (strategy === "stage+name")
        return "".concat($app.stage.substring(0, max - nameLen - 1), "-").concat(name);
    return "".concat($app.name.substring(0, max - stageLen - nameLen - 2), "-").concat($app.stage, "-").concat(name);
}
function hashNumberToPrettyString(number, length) {
    var charLength = exports.PRETTY_CHARS.length;
    var hash = "";
    while (number > 0) {
        hash = exports.PRETTY_CHARS[number % charLength] + hash;
        number = Math.floor(number / charLength);
    }
    // Padding with 's'
    hash = hash.slice(0, length);
    while (hash.length < length) {
        hash = "s" + hash;
    }
    return hash;
}
function hashStringToPrettyString(str, length) {
    var hash = crypto_1.default.createHash("sha256");
    hash.update(str);
    var num = Number("0x" + hash.digest("hex").substring(0, 16));
    return hashNumberToPrettyString(num, length);
}
exports.PRETTY_CHARS = "abcdefhkmnorstuvwxz";
