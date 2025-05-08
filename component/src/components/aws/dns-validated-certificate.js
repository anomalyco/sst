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
exports.DnsValidatedCertificate = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var DnsValidatedCertificate = /** @class */ (function (_super) {
    __extends(DnsValidatedCertificate, _super);
    function DnsValidatedCertificate(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var domainName = args.domainName, alternativeNames = args.alternativeNames, dns = args.dns;
        var certificate = createCertificate();
        var records = createDnsRecords();
        _this.certificateValidation = validateCertificate();
        function createCertificate() {
            return new aws_1.acm.Certificate("".concat(name, "Certificate"), {
                domainName: domainName,
                validationMethod: "DNS",
                subjectAlternativeNames: alternativeNames !== null && alternativeNames !== void 0 ? alternativeNames : [],
            }, { parent: parent });
        }
        function createDnsRecords() {
            return (0, pulumi_1.all)([dns, domainName, certificate.domainValidationOptions]).apply(function (_a) {
                var dns = _a[0], domainName = _a[1], options = _a[2];
                // filter unique records
                var records = [];
                options = options.filter(function (option) {
                    var key = option.resourceRecordType + option.resourceRecordName;
                    if (records.includes(key))
                        return false;
                    records.push(key);
                    return true;
                });
                // create CAA record if domain not hosted on Route53
                var caaRecords = dns.provider === "aws"
                    ? undefined
                    : dns.createCaa(name, domainName, { parent: parent });
                // create records
                return options.map(function (option) {
                    return dns.createRecord(name, {
                        type: option.resourceRecordType,
                        name: option.resourceRecordName,
                        value: option.resourceRecordValue,
                    }, { parent: parent, dependsOn: caaRecords ? __spreadArray([], caaRecords, true) : [] });
                });
            });
        }
        function validateCertificate() {
            return new aws_1.acm.CertificateValidation("".concat(name, "Validation"), {
                certificateArn: certificate.arn,
            }, { parent: parent, dependsOn: records });
        }
        return _this;
    }
    Object.defineProperty(DnsValidatedCertificate.prototype, "arn", {
        get: function () {
            return this.certificateValidation.certificateArn;
        },
        enumerable: false,
        configurable: true
    });
    return DnsValidatedCertificate;
}(component_1.Component));
exports.DnsValidatedCertificate = DnsValidatedCertificate;
var __pulumiType = "sst:aws:Certificate";
// @ts-expect-error
DnsValidatedCertificate.__pulumiType = __pulumiType;
