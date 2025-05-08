"use strict";
/**
 * The Cloudflare DNS Adapter is used to create DNS records to manage domains hosted on
 * [Cloudflare DNS](https://developers.cloudflare.com/dns/).
 *
 * :::note
 * You need to [add the Cloudflare provider](/docs/providers/#install) to use this adapter.
 * :::
 *
 * This needs the Cloudflare provider. To add it run:
 *
 * ```bash
 * sst add cloudflare
 * ```
 *
 * This adapter is passed in as `domain.dns` when setting a custom domain, where `example.com`
 * is hosted on Cloudflare.
 *
 * ```ts
 * {
 *   domain: {
 *     name: "example.com",
 *     dns: sst.cloudflare.dns()
 *   }
 * }
 * ```
 *
 * Specify the zone ID.
 *
 * ```ts
 * {
 *   domain: {
 *     name: "example.com",
 *     dns: sst.cloudflare.dns({
 *       zone: "415e6f4653b6d95b775d350f32119abb"
 *     })
 *   }
 * }
 * ```
 *
 * @packageDocumentation
 */
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
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.dns = dns;
var cloudflare = require("@pulumi/cloudflare");
var naming_1 = require("../naming");
var zone_lookup_1 = require("./providers/zone-lookup");
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var account_id_1 = require("./account-id");
var dns_record_1 = require("./providers/dns-record");
function dns(args) {
    if (args === void 0) { args = {}; }
    return {
        provider: "cloudflare",
        createAlias: createAlias,
        createCaa: createCaa,
        createRecord: createRecord,
    };
    function lookupZone(namePrefix, recordType, recordName, opts) {
        if (args.zone) {
            var zone_1 = cloudflare.getZoneOutput({
                zoneId: args.zone,
            });
            return {
                id: zone_1.id,
                name: zone_1.name,
            };
        }
        var zone = new zone_lookup_1.ZoneLookup("".concat(namePrefix).concat(recordType).concat(recordName, "ZoneLookup"), {
            accountId: account_id_1.DEFAULT_ACCOUNT_ID,
            domain: recordName.replace(/\.$/, ""),
        }, opts);
        return {
            id: zone.zoneId,
            name: zone.zoneName,
        };
    }
    function createAlias(namePrefix, record, opts) {
        return handleCreate(namePrefix, {
            name: record.name,
            type: "CNAME",
            value: record.aliasName,
            isAlias: true,
        }, opts);
    }
    function createCaa(namePrefix, recordName, opts) {
        var zone = lookupZone(namePrefix, "CAA", recordName, opts);
        // Need to use the OverridableDnsRecord instead of the cloudflare.Record because
        // "allowOverride" does not work properly. When CAA records exist, the Terraform
        // provider will do a look up on existing records and only ignore the error if
        // there is exactly one match. But in our cases, there are two matches:
        // 1. CAA 0 issue "amazonaws.com"
        // 2. CAA 0 issuewild "amazonaws.com"
        // There can also be others ie. CAA 0 issue "letsencrypt.org"
        // So we need to use the OverridableDnsRecord to properly ignore existing records.
        return [
            new dns_record_1.DnsRecord("".concat(namePrefix, "CAA").concat(recordName, "Record"), {
                zoneId: zone.id,
                type: "CAA",
                name: zone.name,
                data: {
                    flags: "0",
                    tag: "issue",
                    value: "amazonaws.com",
                },
            }, opts),
            new dns_record_1.DnsRecord("".concat(namePrefix, "CAAWildcard").concat(recordName, "Record"), {
                zoneId: zone.id,
                type: "CAA",
                name: zone.name,
                data: {
                    flags: "0",
                    tag: "issuewild",
                    value: "amazonaws.com",
                },
            }, opts),
        ];
    }
    function createRecord(namePrefix, record, opts) {
        return handleCreate(namePrefix, record, opts);
    }
    function handleCreate(namePrefix, record, opts) {
        return (0, pulumi_1.output)(record).apply(function (record) {
            var _a;
            var _b;
            var zone = lookupZone(namePrefix, record.type, record.name, opts);
            var proxy = (0, pulumi_1.output)(args.proxy).apply(function (proxy) { var _a; return (_a = (proxy && record.isAlias)) !== null && _a !== void 0 ? _a : false; });
            var nameSuffix = (0, naming_1.logicalName)(record.name);
            var type = record.type.toUpperCase();
            return new ((_a = cloudflare.Record).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.record, "".concat(namePrefix).concat(record.type, "Record").concat(nameSuffix), __assign(__assign({ zoneId: zone.id, proxied: (0, pulumi_1.output)(proxy), type: type, name: record.name }, (type === "TXT"
                ? {
                    content: record.value.startsWith("\"")
                        ? record.value
                        : "\"".concat(record.value, "\""),
                }
                : {
                    content: record.value,
                })), { ttl: (0, pulumi_1.output)(proxy).apply(function (proxy) { return (proxy ? 1 : 60); }), allowOverwrite: args.override }), opts), false)))();
        });
    }
}
