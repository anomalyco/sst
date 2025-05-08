"use strict";
/**
 * The AWS DNS Adapter is used to create DNS records to manage domains hosted on
 * [Route 53](https://aws.amazon.com/route53/).
 *
 * This adapter is passed in as `domain.dns` when setting a custom domain.
 *
 * @example
 *
 * ```ts
 * {
 *   domain: {
 *     name: "example.com",
 *     dns: sst.aws.dns()
 *   }
 * }
 * ```
 *
 * You can also specify a hosted zone ID if you have multiple hosted zones with the same domain.
 *
 * ```ts
 * {
 *   domain: {
 *     name: "example.com",
 *     dns: sst.aws.dns({
 *       zone: "Z2FDTNDATAQYW2"
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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype);
    return g.next = verb(0), g["throw"] = verb(1), g["return"] = verb(2), typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
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
var naming_1 = require("../naming");
var hosted_zone_lookup_1 = require("./providers/hosted-zone-lookup");
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var error_1 = require("../error");
function dns(args) {
    if (args === void 0) { args = {}; }
    return {
        provider: "aws",
        createAlias: createAlias,
        createCaa: createCaa,
        createRecord: createRecord,
    };
    /**
     * Creates alias records in the hosted zone.
     *
     * @param namePrefix The prefix to use for the resource names.
     * @param record The alias record to create.
     * @param opts The component resource options.
     */
    function createAlias(namePrefix, record, opts) {
        return ["A", "AAAA"].map(function (type) {
            return _createRecord(namePrefix, {
                type: type,
                name: record.name,
                aliases: [
                    {
                        name: record.aliasName,
                        zoneId: record.aliasZone,
                        evaluateTargetHealth: true,
                    },
                ],
            }, opts);
        });
    }
    function createCaa(namePrefix, recordName, opts) {
        // placeholder
        return undefined;
    }
    /**
     * Creates a DNS record in the hosted zone.
     *
     * @param namePrefix The prefix to use for the resource names.
     * @param record The DNS record to create.
     * @param opts The component resource options.
     */
    function createRecord(namePrefix, record, opts) {
        return _createRecord(namePrefix, {
            type: record.type,
            name: record.name,
            ttl: 60,
            records: [record.value],
        }, opts);
    }
    function _createRecord(namePrefix, partial, opts) {
        return (0, pulumi_1.output)(partial).apply(function (partial) {
            var nameSuffix = (0, naming_1.logicalName)(partial.name);
            var zoneId = lookupZone();
            var dnsRecord = createRecord();
            return dnsRecord;
            function lookupZone() {
                var _this = this;
                if (args.zone) {
                    return (0, pulumi_1.output)(args.zone).apply(function (zoneId) { return __awaiter(_this, void 0, void 0, function () {
                        var zone;
                        return __generator(this, function (_a) {
                            switch (_a.label) {
                                case 0: return [4 /*yield*/, aws_1.route53.getZone({ zoneId: zoneId })];
                                case 1:
                                    zone = _a.sent();
                                    if (!partial.name.replace(/\.$/, "").endsWith(zone.name)) {
                                        throw new error_1.VisibleError("The DNS record \"".concat(partial.name, "\" cannot be created because the domain name does not match the hosted zone \"").concat(zone.name, "\" (").concat(zoneId, ")."));
                                    }
                                    return [2 /*return*/, zoneId];
                            }
                        });
                    }); });
                }
                return new hosted_zone_lookup_1.HostedZoneLookup("".concat(namePrefix).concat(partial.type, "ZoneLookup").concat(nameSuffix), {
                    domain: (0, pulumi_1.output)(partial.name).apply(function (name) {
                        return name.replace(/\.$/, "");
                    }),
                }, opts).zoneId;
            }
            function createRecord() {
                var _a;
                var _b;
                return new ((_a = aws_1.route53.Record).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.record, "".concat(namePrefix).concat(partial.type, "Record").concat(nameSuffix), __assign({ zoneId: zoneId, allowOverwrite: args.override }, partial), opts), false)))();
            }
        });
    }
}
