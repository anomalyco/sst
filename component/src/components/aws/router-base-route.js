"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.parsePattern = parsePattern;
exports.buildKvNamespace = buildKvNamespace;
exports.createKvRouteData = createKvRouteData;
exports.updateKvRoutes = updateKvRoutes;
var crypto_1 = require("crypto");
var pulumi_1 = require("@pulumi/pulumi");
var kv_routes_update_1 = require("./providers/kv-routes-update");
var kv_keys_1 = require("./providers/kv-keys");
function parsePattern(pattern) {
    var _a = pattern.split("/"), host = _a[0], path = _a.slice(1);
    return {
        host: host
            .replace(/[.+?^${}()|[\]\\]/g, "\\$&") // Escape special regex chars
            .replace(/\*/g, ".*"), // Replace * with .*
        path: "/" + path.join("/"),
    };
}
function buildKvNamespace(name) {
    // In the case multiple sites use the same kv store, we need to namespace the keys
    return crypto_1.default
        .createHash("md5")
        .update("".concat($app.name, "-").concat($app.stage, "-").concat(name))
        .digest("hex")
        .substring(0, 4);
}
function createKvRouteData(name, args, parent, routeNs, data) {
    new kv_keys_1.KvKeys("".concat(name, "RouteKey"), {
        store: args.store,
        namespace: routeNs,
        entries: {
            metadata: (0, pulumi_1.jsonStringify)(data),
        },
        purge: false,
    }, { parent: parent });
}
function updateKvRoutes(name, args, parent, routeType, routeNs, pattern) {
    return new kv_routes_update_1.KvRoutesUpdate("".concat(name, "RoutesUpdate"), {
        store: args.store,
        namespace: args.routerNamespace,
        key: "routes",
        entry: [routeType, routeNs, pattern.host, pattern.path].join(","),
    }, { parent: parent });
}
