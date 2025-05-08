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
exports.AppSync = void 0;
var promises_1 = require("fs/promises");
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var naming_1 = require("../naming");
var error_1 = require("../error");
var app_sync_data_source_1 = require("./app-sync-data-source");
var app_sync_resolver_1 = require("./app-sync-resolver");
var app_sync_function_1 = require("./app-sync-function");
var dns_js_1 = require("./dns.js");
var dns_validated_certificate_1 = require("./dns-validated-certificate");
var provider_1 = require("./helpers/provider");
var aws_1 = require("@pulumi/aws");
/**
 * The `AppSync` component lets you add an [Amazon AppSync GraphQL API](https://docs.aws.amazon.com/appsync/latest/devguide/what-is-appsync.html) to your app.
 *
 * @example
 *
 * #### Create a GraphQL API
 *
 * ```ts title="sst.config.ts"
 * const api = new sst.aws.AppSync("MyApi", {
 *   schema: "schema.graphql",
 * });
 * ```
 *
 * #### Add a data source
 *
 * ```ts title="sst.config.ts"
 * const lambdaDS = api.addDataSource({
 *   name: "lambdaDS",
 *   lambda: "src/lambda.handler",
 * });
 * ```
 *
 * #### Add a resolver
 *
 * ```ts title="sst.config.ts"
 * api.addResolver("Query user", {
 *   dataSource: lambdaDS.name,
 * });
 * ```
 */
var AppSync = /** @class */ (function (_super) {
    __extends(AppSync, _super);
    function AppSync(name, args, opts) {
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var domain = normalizeDomain();
        var schema = loadSchema();
        var api = createGraphQLApi();
        var certificateArn = createSsl();
        var domainName = createDomainName();
        createDnsRecords();
        _this.constructorName = name;
        _this.constructorOpts = opts;
        _this.api = api;
        _this.domainName = domainName;
        _this.registerOutputs({ _hint: _this.url });
        function normalizeDomain() {
            if (!args.domain)
                return;
            // validate
            (0, pulumi_1.output)(args.domain).apply(function (domain) {
                if (typeof domain === "string")
                    return;
                if (!domain.name)
                    throw new Error("Missing \"name\" for domain.");
                if (domain.dns === false && !domain.cert)
                    throw new Error("Need to provide a validated certificate via \"cert\" when DNS is disabled");
            });
            // normalize
            return (0, pulumi_1.output)(args.domain).apply(function (domain) {
                var _a;
                var norm = typeof domain === "string" ? { name: domain } : domain;
                return {
                    name: norm.name,
                    dns: norm.dns === false ? undefined : (_a = norm.dns) !== null && _a !== void 0 ? _a : (0, dns_js_1.dns)(),
                    cert: norm.cert,
                };
            });
        }
        function loadSchema() {
            var _this = this;
            return (0, pulumi_1.output)(args.schema).apply(function (schema) { return __awaiter(_this, void 0, void 0, function () { return __generator(this, function (_a) {
                return [2 /*return*/, promises_1.default.readFile(schema, { encoding: "utf-8" })];
            }); }); });
        }
        function createGraphQLApi() {
            var _a;
            var _b;
            return new ((_a = aws_1.appsync.GraphQLApi).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.api, "".concat(name, "Api"), {
                schema: schema,
                authenticationType: "API_KEY",
            }, { parent: parent }), false)))();
        }
        function createSsl() {
            if (!domain)
                return;
            return domain.apply(function (domain) {
                if (domain.cert)
                    return (0, pulumi_1.output)(domain.cert);
                // Certificates used for AppSync are required to be created in the us-east-1 region
                return new dns_validated_certificate_1.DnsValidatedCertificate("".concat(name, "Ssl"), {
                    domainName: domain.name,
                    dns: domain.dns,
                }, { parent: parent, provider: (0, provider_1.useProvider)("us-east-1") }).arn;
            });
        }
        function createDomainName() {
            var _a;
            var _b;
            if (!domain || !certificateArn)
                return;
            var domainName = new ((_a = aws_1.appsync.DomainName).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.domainName, "".concat(name, "DomainName"), {
                domainName: domain === null || domain === void 0 ? void 0 : domain.name,
                certificateArn: certificateArn,
            }, { parent: parent }), false)))();
            new aws_1.appsync.DomainNameApiAssociation("".concat(name, "DomainAssociation"), {
                apiId: api.id,
                domainName: domainName.domainName,
            });
            return domainName;
        }
        function createDnsRecords() {
            if (!domain || !domainName)
                return;
            domain.apply(function (domain) {
                if (!domain.dns)
                    return;
                domain.dns.createAlias(name, {
                    name: domain.name,
                    aliasName: domainName.appsyncDomainName,
                    aliasZone: domainName.hostedZoneId,
                }, { parent: parent });
            });
        }
        return _this;
    }
    Object.defineProperty(AppSync.prototype, "id", {
        /**
         * The GraphQL API ID.
         */
        get: function () {
            return this.api.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(AppSync.prototype, "url", {
        /**
         * The URL of the GraphQL API.
         */
        get: function () {
            return this.domainName
                ? (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["https://", "/graphql"], ["https://", "/graphql"])), this.domainName.domainName) : this.api.uris["GRAPHQL"];
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(AppSync.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon AppSync GraphQL API.
                 */
                api: this.api,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Add a data source to this AppSync API.
     *
     * @param args Configure the data source.
     *
     * @example
     *
     * Add a Lambda function as a data source.
     *
     * ```js title="sst.config.ts"
     * api.addDataSource({
     *   name: "lambdaDS",
     *   lambda: "src/lambda.handler"
     * });
     * ```
     *
     * Customize the Lambda function.
     *
     * ```js title="sst.config.ts"
     * api.addDataSource({
     *   name: "lambdaDS",
     *   lambda: {
     *     handler: "src/lambda.handler",
     *     timeout: "60 seconds"
     *   }
     * });
     * ```
     *
     * Add a data source with an existing Lambda function.
     *
     * ```js title="sst.config.ts"
     * api.addDataSource({
     *   name: "lambdaDS",
     *   lambda: "arn:aws:lambda:us-east-1:123456789012:function:my-function"
     * })
     * ```
     *
     * Add a DynamoDB table as a data source.
     *
     * ```js title="sst.config.ts"
     * api.addDataSource({
     *   name: "dynamoDS",
     *   dynamodb: "arn:aws:dynamodb:us-east-1:123456789012:table/my-table"
     * })
     * ```
     */
    AppSync.prototype.addDataSource = function (args) {
        var self = this;
        var selfName = this.constructorName;
        var nameSuffix = (0, naming_1.logicalName)(args.name);
        return new app_sync_data_source_1.AppSyncDataSource("".concat(selfName, "DataSource").concat(nameSuffix), __assign({ apiId: self.api.id, apiComponentName: selfName }, args), { provider: this.constructorOpts.provider });
    };
    /**
     * Add a function to this AppSync API.
     *
     * @param args Configure the function.
     *
     * @example
     *
     * Add a function using a Lambda data source.
     *
     * ```js title="sst.config.ts"
     * api.addFunction({
     *   name: "myFunction",
     *   dataSource: "lambdaDS",
     * });
     * ```
     *
     * Add a function using a DynamoDB data source.
     *
     * ```js title="sst.config.ts"
     * api.addResolver("Query user", {
     *   name: "myFunction",
     *   dataSource: "dynamoDS",
     *   requestTemplate: `{
     *     "version": "2017-02-28",
     *     "operation": "Scan",
     *   }`,
     *   responseTemplate: `{
     *     "users": $utils.toJson($context.result.items)
     *   }`,
     * });
     * ```
     */
    AppSync.prototype.addFunction = function (args) {
        var self = this;
        var selfName = this.constructorName;
        var nameSuffix = (0, naming_1.logicalName)(args.name);
        return new app_sync_function_1.AppSyncFunction("".concat(selfName, "Function").concat(nameSuffix), __assign({ apiId: self.api.id }, args), { provider: this.constructorOpts.provider });
    };
    /**
     * Add a resolver to this AppSync API.
     *
     * @param operation The type and name of the operation.
     * @param args Configure the resolver.
     *
     * @example
     *
     * Add a resolver using a Lambda data source.
     *
     * ```js title="sst.config.ts"
     * api.addResolver("Query user", {
     *   dataSource: "lambdaDS",
     * });
     * ```
     *
     * Add a resolver using a DynamoDB data source.
     *
     * ```js title="sst.config.ts"
     * api.addResolver("Query user", {
     *   dataSource: "dynamoDS",
     *   requestTemplate: `{
     *     "version": "2017-02-28",
     *     "operation": "Scan",
     *   }`,
     *   responseTemplate: `{
     *     "users": $utils.toJson($context.result.items)
     *   }`,
     * });
     * ```
     *
     * Add a pipeline resolver.
     *
     * ```js title="sst.config.ts"
     * api.addResolver("Query user", {
     *   functions: [
     *     "MyFunction1",
     *     "MyFunction2"
     *   ]
     *   code: `
     *     export function request(ctx) {
     *       return {};
     *     }
     *     export function response(ctx) {
     *       return ctx.result;
     *     }
     *   `,
     * });
     * ```
     */
    AppSync.prototype.addResolver = function (operation, args) {
        var self = this;
        var selfName = this.constructorName;
        // Parse field and type
        var parts = operation.trim().split(/\s+/);
        if (parts.length !== 2)
            throw new error_1.VisibleError("Invalid resolver ".concat(operation));
        var type = parts[0], field = parts[1];
        var nameSuffix = "".concat((0, naming_1.logicalName)(type)) + "".concat((0, naming_1.logicalName)(field));
        return new app_sync_resolver_1.AppSyncResolver("".concat(selfName, "Resolver").concat(nameSuffix), __assign({ apiId: self.api.id, type: type, field: field }, args), { provider: this.constructorOpts.provider });
    };
    /** @internal */
    AppSync.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
            },
        };
    };
    return AppSync;
}(component_1.Component));
exports.AppSync = AppSync;
var __pulumiType = "sst:aws:AppSync";
// @ts-expect-error
AppSync.__pulumiType = __pulumiType;
var templateObject_1;
