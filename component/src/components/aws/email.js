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
var __makeTemplateObject = (this && this.__makeTemplateObject) || function (cooked, raw) {
    if (Object.defineProperty) { Object.defineProperty(cooked, "raw", { value: raw }); } else { cooked.raw = raw; }
    return cooked;
};
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
exports.Email = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var dns_js_1 = require("./dns.js");
var aws_1 = require("@pulumi/aws");
var permission_1 = require("./permission");
/**
 * The `Email` component lets you send emails in your app.
 * It uses [Amazon Simple Email Service](https://aws.amazon.com/ses/).
 *
 * You can configure it to send emails from a specific email address or from any email addresses
 * in a domain.
 *
 * :::tip
 * New AWS SES accounts are in _sandbox mode_ and need to [request production access](https://docs.aws.amazon.com/ses/latest/dg/request-production-access.html).
 * :::
 *
 * By default, new AWS SES accounts are in the _sandbox mode_ and can only send
 * email to verified email addresses and domains. It also limits your account has to a sending
 * quota. To remove these restrictions, you need to [request production access](https://docs.aws.amazon.com/ses/latest/dg/request-production-access.html).
 *
 * #### Sending from an email address
 *
 * For using an email address as the sender, you need to verify the email address.
 *
 * ```ts title="sst.config.ts"
 * const email = new sst.aws.Email("MyEmail", {
 *   sender: "spongebob@example.com",
 * });
 * ```
 *
 * #### Sending from a domain
 *
 * When you use a domain as the sender, you'll need to verify that you own the domain.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Email("MyEmail", {
 *   sender: "example.com"
 * });
 * ```
 *
 * #### Configuring DMARC
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Email("MyEmail", {
 *   sender: "example.com",
 *   dmarc: "v=DMARC1; p=quarantine; adkim=s; aspf=s;"
 * });
 * ```
 *
 * #### Link to a resource
 *
 * You can link it to a function or your Next.js app to send emails.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.aws.Function("MyApi", {
 *   handler: "sender.handler",
 *   link: [email]
 * });
 * ```
 *
 * Now in your function you can use the AWS SES SDK to send emails.
 *
 * ```ts title="sender.ts" {1, 8}
 * import { Resource } from "sst";
 * import { SESv2Client, SendEmailCommand } from "@aws-sdk/client-sesv2";
 *
 * const client = new SESv2Client();
 *
 * await client.send(
 *   new SendEmailCommand({
 *     FromEmailAddress: Resource.MyEmail.sender,
 *     Destination: {
 *       ToAddresses: ["patrick@example.com"]
 *     },
 *     Content: {
 *       Simple: {
 *         Subject: { Data: "Hello World!" },
 *         Body: { Text: { Data: "Sent from my SST app." } }
 *       }
 *     }
 *   })
 * );
 * ```
 */
var Email = /** @class */ (function (_super) {
    __extends(Email, _super);
    function Email(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        if (args && "ref" in args) {
            var ref = reference();
            _this._sender = ref.identity.emailIdentity;
            _this.identity = ref.identity;
            _this.configurationSet = ref.configurationSet;
            return _this;
        }
        var isDomain = checkIsDomain();
        var dns = normalizeDns();
        var dmarc = normalizeDmarc();
        var configurationSet = createConfigurationSet();
        var identity = createIdentity();
        createEvents();
        isDomain.apply(function (isDomain) {
            if (!isDomain)
                return;
            createDkimRecords();
            createDmarcRecord();
            waitForVerification();
        });
        _this._sender = (0, pulumi_1.output)(args.sender);
        _this.identity = identity;
        _this.configurationSet = configurationSet;
        function reference() {
            var ref = args;
            var identity = aws_1.sesv2.EmailIdentity.get("".concat(name, "Identity"), ref.sender, undefined, { parent: self });
            var configurationSet = aws_1.sesv2.ConfigurationSet.get("".concat(name, "Config"), identity.configurationSetName.apply(function (v) { return v; }), undefined, { parent: self });
            return {
                identity: identity,
                configurationSet: configurationSet,
            };
        }
        function checkIsDomain() {
            return (0, pulumi_1.output)(args.sender).apply(function (sender) { return !sender.includes("@"); });
        }
        function normalizeDns() {
            var _a;
            (0, pulumi_1.all)([args.dns, isDomain]).apply(function (_a) {
                var dns = _a[0], isDomain = _a[1];
                if (!isDomain && dns)
                    throw new Error("The \"dns\" property is only valid when \"sender\" is a domain.");
            });
            return (_a = args.dns) !== null && _a !== void 0 ? _a : (0, dns_js_1.dns)();
        }
        function normalizeDmarc() {
            var _a;
            (0, pulumi_1.all)([args.dmarc, isDomain]).apply(function (_a) {
                var dmarc = _a[0], isDomain = _a[1];
                if (!isDomain && dmarc)
                    throw new Error("The \"dmarc\" property is only valid when \"sender\" is a domain.");
            });
            return (_a = args.dmarc) !== null && _a !== void 0 ? _a : "v=DMARC1; p=none;";
        }
        function createConfigurationSet() {
            var _a;
            var _b;
            return new ((_a = aws_1.sesv2.ConfigurationSet).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.configurationSet, "".concat(name, "Config"), { configurationSetName: "" }, { parent: self }), false)))();
        }
        function createIdentity() {
            var _a;
            var _b;
            return new ((_a = aws_1.sesv2.EmailIdentity).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.identity, "".concat(name, "Identity"), {
                emailIdentity: args.sender,
                configurationSetName: configurationSet.configurationSetName,
            }, { parent: self }), false)))();
        }
        function createEvents() {
            var _a;
            (0, pulumi_1.output)((_a = args.events) !== null && _a !== void 0 ? _a : []).apply(function (events) {
                return events.forEach(function (event) {
                    new aws_1.sesv2.ConfigurationSetEventDestination("".concat(name, "Event").concat(event.name), {
                        configurationSetName: configurationSet.configurationSetName,
                        eventDestinationName: event.name,
                        eventDestination: __assign(__assign(__assign({ matchingEventTypes: event.types.map(function (t) {
                                return t.toUpperCase().replaceAll("-", "_");
                            }) }, (event.bus
                            ? { eventBridgeDestination: { eventBusArn: event.bus } }
                            : {})), (event.topic
                            ? { snsDestination: { topicArn: event.topic } }
                            : {})), { enabled: true }),
                    }, { parent: self });
                });
            });
        }
        function createDkimRecords() {
            (0, pulumi_1.all)([dns, identity === null || identity === void 0 ? void 0 : identity.dkimSigningAttributes.tokens]).apply(function (_a) {
                var dns = _a[0], tokens = _a[1];
                if (!dns)
                    return;
                tokens === null || tokens === void 0 ? void 0 : tokens.map(function (token) {
                    return dns.createRecord(name, {
                        type: "CNAME",
                        name: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["", "._domainkey.", ""], ["", "._domainkey.", ""])), token, args.sender),
                        value: "".concat(token, ".dkim.amazonses.com"),
                    }, { parent: self });
                });
            });
        }
        function createDmarcRecord() {
            (0, pulumi_1.output)(dns).apply(function (dns) {
                if (!dns)
                    return;
                dns.createRecord(name, {
                    type: "TXT",
                    name: (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["_dmarc.", ""], ["_dmarc.", ""])), args.sender),
                    value: dmarc,
                }, { parent: self });
            });
        }
        function waitForVerification() {
            new aws_1.ses.DomainIdentityVerification("".concat(name, "Verification"), {
                domain: args.sender,
            }, { parent: self, dependsOn: identity });
        }
        return _this;
    }
    Object.defineProperty(Email.prototype, "sender", {
        /**
         * The sender email address or domain name.
         */
        get: function () {
            return this._sender;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Email.prototype, "configSet", {
        /**
         * The name of the configuration set.
         */
        get: function () {
            return this.configurationSet.configurationSetName;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Email.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon SES identity.
                 */
                identity: this.identity,
                /**
                 * The Amazon SES configuration set.
                 */
                configurationSet: this.configurationSet,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Email.prototype.getSSTLink = function () {
        return {
            properties: {
                sender: this._sender,
                configSet: this.configSet,
            },
            include: [
                (0, permission_1.permission)({
                    actions: ["ses:*"],
                    resources: [this.identity.arn, this.configurationSet.arn],
                }),
                // When the SES account is in sandbox mode, it seems you have to include verified
                // receipients inside `resources`. Needs further investigation.
                (0, permission_1.permission)({
                    actions: ["ses:SendEmail", "ses:SendRawEmail"],
                    resources: ["*"],
                }),
            ],
        };
    };
    /**
     * Reference an existing Email component with the given Amazon SES identity. This is useful
     * when you create an SES identity in one stage and want to share it in another stage. It
     * avoids having to create a new Email component in the other stage.
     *
     * @param name The name of the component.
     * @param sender The email address or domain name of the existing SES identity.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create an Email component in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new component, you want to share the one from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const email = $app.stage === "frank"
     *   ? sst.aws.Email.get("MyEmail", "spongebob@example.com")
     *   : new sst.aws.Email("MyEmail", {
     *       sender: "spongebob@example.com",
     *     });
     * ```
     */
    Email.get = function (name, sender, opts) {
        return new Email(name, {
            ref: true,
            sender: sender,
        }, opts);
    };
    return Email;
}(component_1.Component));
exports.Email = Email;
var __pulumiType = "sst:aws:Email";
// @ts-expect-error
Email.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2;
