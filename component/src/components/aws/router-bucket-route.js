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
exports.RouterBucketRoute = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var router_base_route_1 = require("./router-base-route");
var duration_1 = require("../duration");
/**
 * The `RouterBucketRoute` component is internally used by the `Router` component
 * to add routes.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `routeBucket` method of the `Router` component.
 */
var RouterBucketRoute = /** @class */ (function (_super) {
    __extends(RouterBucketRoute, _super);
    function RouterBucketRoute(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        (0, pulumi_1.all)([args.pattern, args.routeArgs]).apply(function (_a) {
            var pattern = _a[0], routeArgs = _a[1];
            var patternData = (0, router_base_route_1.parsePattern)(pattern);
            var namespace = (0, router_base_route_1.buildKvNamespace)(name);
            (0, router_base_route_1.createKvRouteData)(name, args, self, namespace, {
                domain: (0, pulumi_1.output)(args.bucket).nodes.bucket.bucketRegionalDomainName,
                rewrite: routeArgs === null || routeArgs === void 0 ? void 0 : routeArgs.rewrite,
                origin: {
                    connectionAttempts: routeArgs === null || routeArgs === void 0 ? void 0 : routeArgs.connectionAttempts,
                    timeouts: {
                        connectionTimeout: (routeArgs === null || routeArgs === void 0 ? void 0 : routeArgs.connectionTimeout) &&
                            (0, duration_1.toSeconds)(routeArgs === null || routeArgs === void 0 ? void 0 : routeArgs.connectionTimeout),
                    },
                },
            });
            (0, router_base_route_1.updateKvRoutes)(name, args, self, "bucket", namespace, patternData);
        });
        return _this;
    }
    return RouterBucketRoute;
}(component_1.Component));
exports.RouterBucketRoute = RouterBucketRoute;
var __pulumiType = "sst:aws:RouterBucketRoute";
// @ts-expect-error
RouterBucketRoute.__pulumiType = __pulumiType;
