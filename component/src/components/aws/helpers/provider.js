"use strict";
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
exports.useProvider = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var aws_1 = require("@pulumi/aws");
var lazy_1 = require("../../../util/lazy");
var useProviderCache = (0, lazy_1.lazy)(function () { return new Map(); });
var useProvider = function (region) {
    var cache = useProviderCache();
    var existing = cache.get(region);
    if (existing)
        return existing;
    var config = pulumi_1.runtime.allConfig();
    for (var key in config) {
        var value = config[key];
        delete config[key];
        var _a = key.split(":"), prefix = _a[0], real = _a[1];
        if (prefix !== "aws")
            continue;
        // Array and Object values are JSON encoded, ie.
        // {
        //   allowedAccountIds: '["112245769880"]',
        //   defaultTags: '{"tags":{"sst:app":"playground","sst:stage":"frank"}}',
        //   region: 'us-east-1'
        // }
        try {
            config[real] = JSON.parse(value);
        }
        catch (e) {
            config[real] = value;
        }
    }
    var provider = new aws_1.Provider("AwsProvider.sst.".concat(region), __assign(__assign({}, config), { region: region }));
    cache.set(region, provider);
    return provider;
};
exports.useProvider = useProvider;
