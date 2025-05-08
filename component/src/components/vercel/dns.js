"use strict";
/**
 * The Vercel DNS Adapter is used to create DNS records to manage domains hosted on [Vercel](https://vercel.com/docs/projects/domains/working-with-domains).
 *
 * :::note
 * You need to [add the Vercel provider](/docs/all-providers#directory) to use this adapter.
 * :::
 *
 * This adapter is passed in as `domain.dns` when setting a custom domain; where `example.com`
 * is hosted on Vercel.
 *
 * ```ts
 * {
 *   domain: {
 *     name: "example.com",
 *     dns: sst.vercel.dns({
 *       domain: "example.com"
 *     })
 *   }
 * }
 * ```
 *
 * #### Configure provider
 *
 * 1. To use this component, add the `@pulumiverse/vercel` provider to your app.
 *
 *    ```bash
 *    sst add @pulumiverse/vercel
 *    ```
 *
 * 2. If you don't already have a Vercel Access Token, [follow this guide](https://vercel.com/guides/how-do-i-use-a-vercel-api-access-token#creating-an-access-token) to create one.
 *
 * 3. Add a `VERCEL_API_TOKEN` environment variable with the access token value. If the domain
 * belongs to a team, also add a `VERCEL_TEAM_ID` environment variable with the Team ID. You can
 * find your Team ID inside your team's general project settings in the Vercel dashboard.
 *
 * @packageDocumentation
 */
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
var vercel_1 = require("@pulumiverse/vercel");
var dns_record_1 = require("./providers/dns-record");
var naming_1 = require("../naming");
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var account_id_1 = require("./account-id");
function dns(args) {
    return {
        provider: "vercel",
        createAlias: createAlias,
        createCaa: createCaa,
        createRecord: createRecord,
    };
    function createAlias(namePrefix, record, opts) {
        return createRecord(namePrefix, {
            name: record.name,
            // Cannot set CNAME record on the apex domain
            type: (0, pulumi_1.all)([args.domain, record.name]).apply(function (_a) {
                var domain = _a[0], recordName = _a[1];
                return recordName.startsWith(domain) ? "ALIAS" : "CNAME";
            }),
            value: record.aliasName,
        }, opts);
    }
    function createCaa(namePrefix, recordName, opts) {
        // Need to use the OverridableDnsRecord instead of the vercel.DnsRecord to
        // ignore existing CAA records. This is because the CAA records are not
        // removed.
        return [
            new dns_record_1.DnsRecord("".concat(namePrefix, "CaaRecord"), {
                domain: args.domain,
                name: args.domain,
                type: "CAA",
                value: "0 issue \"amazonaws.com\"",
            }, opts),
            new dns_record_1.DnsRecord("".concat(namePrefix, "CaaWildcardRecord"), {
                domain: args.domain,
                name: args.domain,
                type: "CAA",
                value: "0 issuewild \"amazonaws.com\"",
            }, opts),
        ];
    }
    function createRecord(namePrefix, record, opts) {
        return (0, pulumi_1.all)([args.domain, record]).apply(function (_a) {
            var domain = _a[0], record = _a[1];
            var nameSuffix = (0, naming_1.logicalName)(record.name);
            var recordName = validateRecordName();
            var dnsRecord = createRecord();
            return dnsRecord;
            function validateRecordName() {
                var recordName = record.name.replace(/\.$/, "");
                if (!recordName.endsWith(domain))
                    throw new Error("Record name \"".concat(recordName, "\" is not a subdomain of \"").concat(domain, "\"."));
                return recordName.slice(0, -(domain.length + 1));
            }
            function createRecord() {
                var _a;
                return new (vercel_1.DnsRecord.bind.apply(vercel_1.DnsRecord, __spreadArray([void 0], (0, component_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.record, "".concat(namePrefix).concat(record.type, "Record").concat(nameSuffix), {
                    domain: args.domain,
                    type: record.type,
                    name: recordName,
                    value: record.value,
                    teamId: account_id_1.DEFAULT_TEAM_ID,
                    ttl: 60,
                }, opts), false)))();
            }
        });
    }
}
