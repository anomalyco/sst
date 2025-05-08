"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.URL_UNAVAILABLE = void 0;
exports.linkable = linkable;
var error_1 = require("../error");
exports.URL_UNAVAILABLE = "http://url-unavailable-in-dev.mode";
/** @deprecated
 * instead try
 * ```
 * sst.Linkable.wrap(MyResource, (resource) => ({
 *   properties: { ... },
 *   with: [
 *     sst.aws.permission({ actions: ["foo:*"], resources: [resource.arn] })
 *   ]
 * }))
 * ```
 */
function linkable(obj, cb) {
    throw new error_1.VisibleError([
        "sst.aws.linkable is deprecated. Use sst.Linkable.wrap instead.",
        "sst.Linkable.wrap(MyResource, (resource) => ({",
        "  properties: { ... },",
        "  with: [",
        '    sst.aws.permission({ actions: ["foo:*"], resources: [resource.arn] })',
        "  ]",
        "}))",
    ].join("\n"));
}
