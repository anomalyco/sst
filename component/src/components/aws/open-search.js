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
exports.OpenSearch = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var random_1 = require("@pulumi/random");
var error_1 = require("../error");
var size_1 = require("../size");
var dev_command_js_1 = require("../experimental/dev-command.js");
/**
 * The `OpenSearch` component lets you add an OpenSearch domain to your app using
 * [Amazon OpenSearch Service](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/what-is.html).
 *
 * @example
 *
 * #### Create the instance
 *
 * ```js title="sst.config.ts"
 * const search = new sst.aws.OpenSearch("MySearch");
 * ```
 *
 * #### Link to a resource
 *
 * You can link your instance to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [search]
 * });
 * ```
 *
 * Once linked, you can connect to it from your function code.
 *
 * ```ts title="app/page.tsx" {1,5-9}
 * import { Resource } from "sst";
 * import { Client } from "@opensearch-project/opensearch";
 *
 * const client = new Client({
 *   node: Resource.MySearch.url,
 *   auth: {
 *     username: Resource.MySearch.username,
 *     password: Resource.MySearch.password,
 *   },
 * });
 *
 * // Add a document
 * await client.index({
 *   index: "my-index",
 *   body: { message: "Hello world!" }
 * });
 *
 * // Search for documents
 * const result = await client.search({
 *   index: "my-index",
 *   body: { query: { match: { message: "world" } } }
 * });
 * ```
 *
 * #### Running locally
 *
 * By default, your OpenSearch domain is deployed in `sst dev`. But let's say you are
 * running OpenSearch locally.
 *
 * ```bash
 * docker run \
 *   --rm \
 *   -p 9200:9200 \
 *   -v $(pwd)/.sst/storage/opensearch:/usr/share/opensearch/data \
 *   -e discovery.type=single-node \
 *   -e plugins.security.disabled=true \
 *   -e OPENSEARCH_INITIAL_ADMIN_PASSWORD=^Passw0rd^ \
 *   opensearchproject/opensearch:2.17.0
 * ```
 *
 * You can connect to it in `sst dev` by configuring the `dev` prop.
 *
 * ```ts title="sst.config.ts" {3-5}
 * const opensearch = new sst.aws.OpenSearch("MyOpenSearch", {
 *   dev: {
 *     url: "http://localhost:9200",
 *     username: "admin",
 *     password: "^Passw0rd^"
 *   }
 * });
 * ```
 *
 * This will skip deploying an OpenSearch domain and link to the locally running
 * OpenSearch process instead.
 *
 * ---
 *
 * ### Cost
 *
 * By default this component uses a _Single-AZ Deployment_, _On-Demand Instances_ of a
 * `t3.small.search` at $0.036 per hour. And 10GB of _General Purpose gp3 Storage_
 * at $0.122 per GB per month.
 *
 * That works out to $0.036 x 24 x 30 + $0.122 x 10 or **$27 per month**. Adjust this for
 * the `instance` type and the `storage` you are using.
 *
 * The above are rough estimates for _us-east-1_, check out the [OpenSearch Service pricing](https://aws.amazon.com/opensearch-service/pricing/)
 * for more details.
 */
var OpenSearch = /** @class */ (function (_super) {
    __extends(OpenSearch, _super);
    function OpenSearch(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        if (args && "ref" in args) {
            var ref = reference();
            _this.domain = ref.domain;
            _this._username = ref.username;
            _this._password = ref.password;
            return _this;
        }
        var engineVersion = (0, pulumi_1.output)(args.version).apply(function (v) { return v !== null && v !== void 0 ? v : "OpenSearch_2.17"; });
        var instanceType = (0, pulumi_1.output)(args.instance).apply(function (v) { return v !== null && v !== void 0 ? v : "t3.small"; });
        var username = (0, pulumi_1.output)(args.username).apply(function (v) { return v !== null && v !== void 0 ? v : "admin"; });
        var storage = normalizeStorage();
        var dev = registerDev();
        if (dev === null || dev === void 0 ? void 0 : dev.enabled) {
            _this.dev = dev;
            return _this;
        }
        var password = createPassword();
        var secret = createSecret();
        var domain = createDomain();
        var policy = createPolicy();
        _this.domain = domain;
        _this._username = username;
        _this._password = password;
        _this.registerOutputs({
            _hint: _this.url,
        });
        function reference() {
            var ref = args;
            // Note: passing in `parent` causes Pulumi to lookup the current component's
            //       generated ID for the Domain. Not the one passed int. Need to look into
            //       this.
            //const domain = opensearch.Domain.get(`${name}Domain`, ref.id, undefined, {
            //  parent: self,
            //});
            var domain = aws_1.opensearch.Domain.get("".concat(name, "Domain"), ref.id);
            var input = domain.tags.apply(function (tags) {
                if (!(tags === null || tags === void 0 ? void 0 : tags["sst:ref:username"]))
                    throw new error_1.VisibleError("Failed to get username for OpenSearch ".concat(name, "."));
                if (!(tags === null || tags === void 0 ? void 0 : tags["sst:ref:password"]))
                    throw new error_1.VisibleError("Failed to get password for OpenSearch ".concat(name, "."));
                return {
                    username: tags["sst:ref:username"],
                    password: tags["sst:ref:password"],
                };
            });
            var secret = aws_1.secretsmanager.getSecretVersionOutput({ secretId: input.password }, { parent: self });
            var password = $jsonParse(secret.secretString).apply(function (v) { return v.password; });
            return { domain: domain, username: input.username, password: password };
        }
        function normalizeStorage() {
            var _a;
            return (0, pulumi_1.output)((_a = args.storage) !== null && _a !== void 0 ? _a : "10 GB").apply(function (v) {
                var size = (0, size_1.toGBs)(v);
                if (size < 10) {
                    throw new error_1.VisibleError("Storage must be at least 10 GB for the ".concat(name, " OpenSearch domain."));
                }
                return size;
            });
        }
        function registerDev() {
            var _a, _b, _c;
            if (!args.dev)
                return undefined;
            if ($dev &&
                args.dev.password === undefined &&
                args.password === undefined) {
                throw new error_1.VisibleError("You must provide the password to connect to your locally running OpenSearch domain either by setting the \"dev.password\" or by setting the top-level \"password\" property.");
            }
            var dev = {
                enabled: $dev,
                url: (0, pulumi_1.output)((_a = args.dev.url) !== null && _a !== void 0 ? _a : "http://localhost:9200"),
                username: args.dev.username ? (0, pulumi_1.output)(args.dev.username) : username,
                password: (0, pulumi_1.output)((_c = (_b = args.dev.password) !== null && _b !== void 0 ? _b : args.password) !== null && _c !== void 0 ? _c : ""),
            };
            new dev_command_js_1.DevCommand("".concat(name, "Dev"), {
                dev: {
                    title: name,
                    autostart: true,
                    command: "sst print-and-not-quit",
                },
                environment: {
                    SST_DEV_COMMAND_MESSAGE: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["Make sure your local OpenSearch server is using:\n\n  username: \"", "\"\n  password: \"", "\"\n\nListening on \"", "\"..."], ["Make sure your local OpenSearch server is using:\n\n  username: \"", "\"\n  password: \"", "\"\n\nListening on \"", "\"..."])), dev.username, dev.password, dev.url),
                },
            });
            return dev;
        }
        function createPassword() {
            return args.password
                ? (0, pulumi_1.output)(args.password)
                : new random_1.RandomPassword("".concat(name, "Password"), {
                    length: 32,
                    minLower: 1,
                    minUpper: 1,
                    minNumeric: 1,
                    minSpecial: 1,
                }, { parent: self }).result;
        }
        function createSecret() {
            var secret = new aws_1.secretsmanager.Secret("".concat(name, "Secret"), {
                recoveryWindowInDays: 0,
            }, { parent: self });
            new aws_1.secretsmanager.SecretVersion("".concat(name, "SecretVersion"), {
                secretId: secret.id,
                secretString: (0, pulumi_1.jsonStringify)({
                    username: username,
                    password: password,
                }),
            }, { parent: self });
            return secret;
        }
        function createDomain() {
            var _a;
            var _b;
            return new ((_a = aws_1.opensearch.Domain).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.domain, "".concat(name, "Domain"), {
                engineVersion: engineVersion,
                clusterConfig: {
                    instanceType: (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["", ".search"], ["", ".search"])), instanceType),
                    instanceCount: 1,
                    dedicatedMasterEnabled: false,
                    zoneAwarenessEnabled: false,
                },
                ebsOptions: {
                    ebsEnabled: true,
                    volumeSize: storage,
                    volumeType: "gp3",
                },
                advancedSecurityOptions: {
                    enabled: true,
                    internalUserDatabaseEnabled: true,
                    masterUserOptions: {
                        masterUserName: username,
                        masterUserPassword: password,
                    },
                },
                nodeToNodeEncryption: {
                    enabled: true,
                },
                encryptAtRest: {
                    enabled: true,
                },
                domainEndpointOptions: {
                    enforceHttps: true,
                    tlsSecurityPolicy: "Policy-Min-TLS-1-2-2019-07",
                },
                tags: {
                    "sst:ref:password": secret.id,
                    "sst:ref:username": username,
                },
            }, { parent: self }), false)))();
        }
        function createPolicy() {
            return new aws_1.opensearch.DomainPolicy("".concat(name, "DomainPolicy"), {
                domainName: domain.domainName,
                accessPolicies: aws_1.iam.getPolicyDocumentOutput({
                    statements: [
                        {
                            principals: [{ type: "*", identifiers: ["*"] }],
                            actions: ["*"],
                            resources: ["*"],
                        },
                    ],
                }).json,
            }, { parent: self });
        }
        return _this;
    }
    Object.defineProperty(OpenSearch.prototype, "id", {
        /**
         * The ID of the OpenSearch component.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return (0, pulumi_1.output)("placeholder");
            return this.domain.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(OpenSearch.prototype, "username", {
        /** The username of the master user. */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.username;
            return this._username;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(OpenSearch.prototype, "password", {
        /** The password of the master user. */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.password;
            return this._password;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(OpenSearch.prototype, "url", {
        /**
         * The endpoint of the domain.
         */
        get: function () {
            var _a;
            if ((_a = this.dev) === null || _a === void 0 ? void 0 : _a.enabled)
                return this.dev.url;
            return (0, pulumi_1.interpolate)(templateObject_3 || (templateObject_3 = __makeTemplateObject(["https://", ""], ["https://", ""])), this.domain.endpoint);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(OpenSearch.prototype, "nodes", {
        get: function () {
            return {
                domain: this.domain,
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    OpenSearch.prototype.getSSTLink = function () {
        return {
            properties: {
                username: this.username,
                password: this.password,
                url: this.url,
            },
        };
    };
    /**
     * Reference an existing OpenSearch domain with the given name. This is useful when you
     * create a domain in one stage and want to share it in another. It avoids
     * having to create a new domain in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share OpenSearch domains across stages.
     * :::
     *
     * @param name The name of the component.
     * @param id The ID of the existing OpenSearch component.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a domain in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new domain, you want to share the same domain from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const search = $app.stage === "frank"
     *   ? sst.aws.OpenSearch.get("MyOpenSearch", "arn:aws:es:us-east-1:123456789012:domain/app-dev-myopensearch-efsmkrbt")
     *   : new sst.aws.OpenSearch("MyOpenSearch");
     * ```
     *
     * Here `arn:aws:es:us-east-1:123456789012:domain/app-dev-myopensearch-efsmkrbt` is the
     * ID of the OpenSearch component created in the `dev` stage.
     * You can find this by outputting the ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   id: search.id,
     * };
     * ```
     */
    OpenSearch.get = function (name, id, opts) {
        return new OpenSearch(name, {
            ref: true,
            id: id,
        }, opts);
    };
    return OpenSearch;
}(component_1.Component));
exports.OpenSearch = OpenSearch;
var __pulumiType = "sst:aws:OpenSearch";
// @ts-expect-error
OpenSearch.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2, templateObject_3;
