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
exports.RouterUrlRoute = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var router_base_route_1 = require("./router-base-route");
var duration_1 = require("../duration");
/**
 * The `RouterUrlRoute` component is internally used by the `Router` component
 * to add routes.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `route` method of the `Router` component.
 */
var RouterUrlRoute = /** @class */ (function (_super) {
    __extends(RouterUrlRoute, _super);
    function RouterUrlRoute(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        (0, pulumi_1.all)([args.url, args.pattern, args.routeArgs]).apply(function (_a) {
            var url = _a[0], pattern = _a[1], routeArgs = _a[2];
            var u = new URL(url);
            var host = u.host;
            var protocol = u.protocol.slice(0, -1);
            var patternData = (0, router_base_route_1.parsePattern)(pattern);
            var namespace = (0, router_base_route_1.buildKvNamespace)(name);
            (0, router_base_route_1.createKvRouteData)(name, args, self, namespace, {
                host: host,
                rewrite: routeArgs === null || routeArgs === void 0 ? void 0 : routeArgs.rewrite,
                origin: {
                    protocol: protocol === "https" ? undefined : protocol,
                    connectionAttempts: routeArgs === null || routeArgs === void 0 ? void 0 : routeArgs.connectionAttempts,
                    timeouts: (function () {
                        var timeouts = [
                            "connectionTimeout",
                            "readTimeout",
                            "keepAliveTimeout",
                        ].flatMap(function (k) {
                            return (routeArgs === null || routeArgs === void 0 ? void 0 : routeArgs[k]) ? [[k, (0, duration_1.toSeconds)(routeArgs[k])]] : [];
                        });
                        return timeouts.length ? Object.fromEntries(timeouts) : undefined;
                    })(),
                },
            });
            (0, router_base_route_1.updateKvRoutes)(name, args, self, "url", namespace, patternData);
        });
        return _this;
    }
    return RouterUrlRoute;
}(component_1.Component));
exports.RouterUrlRoute = RouterUrlRoute;
var __pulumiType = "sst:aws:RouterUrlRoute";
// @ts-expect-error
RouterUrlRoute.__pulumiType = __pulumiType;
