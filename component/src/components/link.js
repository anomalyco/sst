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
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Link = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var error_js_1 = require("./error.js");
var linkable_js_1 = require("./linkable.js");
var Link;
(function (Link) {
    var Ref = /** @class */ (function (_super) {
        __extends(Ref, _super);
        function Ref(target, type, properties, include) {
            var _this = _super.call(this, "sst:sst:LinkRef", target + "LinkRef", {
                properties: properties,
                include: include,
            }, {}) || this;
            _this.registerOutputs({
                target: target,
                include: include,
                properties: __assign({ type: type.replaceAll(":", ".") }, properties),
            });
            return _this;
        }
        return Ref;
    }(pulumi_1.ComponentResource));
    Link.Ref = Ref;
    function reset() {
        var links = new Set();
        // Ensure component names are unique
        pulumi_1.runtime.registerStackTransformation(function (args) {
            var isLinkable = args.type.startsWith("sst:") ||
                linkable_js_1.Linkable.wrappedResources.has(args.type);
            if (isLinkable && !args.opts.parent) {
                var lcname = args.name.toLowerCase();
                // "App" is reserved and cannot be used as a component name.
                if (lcname === "app") {
                    throw new error_js_1.VisibleError("Component name \"".concat(args.name, "\" is reserved. Please choose a different name for your \"").concat(args.type, "\" component."));
                }
                // Ensure linkable resources have unique names. This includes all SST components
                // and non-SST components that are linkable.
                if (links.has(lcname)) {
                    throw new error_js_1.VisibleError("Component name ".concat(args.name, " is not unique."));
                }
                links.add(lcname);
            }
            return {
                opts: args.opts,
                props: args.props,
            };
        });
        // Create link refs
        pulumi_1.runtime.registerStackTransformation(function (args) {
            var resource = args.resource;
            process.nextTick(function () {
                if (Link.isLinkable(resource) && !args.opts.parent) {
                    try {
                        var link = resource.getSSTLink();
                        new Ref(args.name, args.type, link.properties, link.include);
                    }
                    catch (e) { }
                }
            });
            return {
                opts: args.opts,
                props: args.props,
            };
        });
    }
    Link.reset = reset;
    function isLinkable(obj) {
        return "getSSTLink" in obj;
    }
    Link.isLinkable = isLinkable;
    function build(links) {
        return links
            .map(function (link) {
            if (!link)
                throw new error_js_1.VisibleError("An undefined link was passed into a `link` array.");
            return link;
        })
            .filter(function (l) { return isLinkable(l); })
            .map(function (l) {
            var link = l.getSSTLink();
            return (0, pulumi_1.all)([l.urn, link]).apply(function (_a) {
                var urn = _a[0], link = _a[1];
                return ({
                    name: urn.split("::").at(-1),
                    properties: __assign(__assign({}, link.properties), { type: urn.split("::").at(-2) }),
                });
            });
        });
    }
    Link.build = build;
    function getProperties(links) {
        var linkProperties = (0, pulumi_1.output)(links !== null && links !== void 0 ? links : []).apply(function (links) {
            return links
                .map(function (link) {
                if (!link)
                    throw new error_js_1.VisibleError("An undefined link was passed into a `link` array.");
                return link;
            })
                .filter(function (l) { return isLinkable(l); })
                .map(function (l) { return ({
                urn: l.urn,
                properties: l.getSSTLink().properties,
            }); });
        });
        return (0, pulumi_1.output)(linkProperties).apply(function (e) {
            return Object.fromEntries(e.map(function (_a) {
                var urn = _a.urn, properties = _a.properties;
                var name = urn.split("::").at(-1);
                var data = __assign(__assign({}, properties), { type: urn.split("::").at(-2) });
                return [name, data];
            }));
        });
    }
    Link.getProperties = getProperties;
    function propertiesToEnv(properties) {
        return (0, pulumi_1.output)(properties).apply(function (properties) {
            var env = Object.fromEntries(Object.entries(properties).map(function (_a) {
                var key = _a[0], value = _a[1];
                return ["SST_RESOURCE_".concat(key), JSON.stringify(value)];
            }));
            env["SST_RESOURCE_App"] = JSON.stringify({
                name: $app.name,
                stage: $app.stage,
            });
            return env;
        });
    }
    Link.propertiesToEnv = propertiesToEnv;
    function getInclude(type, input) {
        if (!input)
            return (0, pulumi_1.output)([]);
        return (0, pulumi_1.output)(input).apply(function (links) {
            return links.filter(isLinkable).flatMap(function (l) {
                var link = l.getSSTLink();
                return (link.include || []).filter(function (i) { return i.type === type; });
            });
        });
    }
    Link.getInclude = getInclude;
    /** @deprecated
     * Use sst.Linkable.wrap instead.
     */
    function linkable(obj, cb) {
        console.warn("sst.linkable is deprecated. Use sst.Linkable.wrap instead.");
        obj.prototype.getSSTLink = function () {
            return cb(this);
        };
    }
    Link.linkable = linkable;
})(Link || (exports.Link = Link = {}));
