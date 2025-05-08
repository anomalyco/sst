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
exports.ApiGatewayWebSocket = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var naming_1 = require("../naming");
var dns_validated_certificate_1 = require("./dns-validated-certificate");
var logging_1 = require("./logging");
var dns_js_1 = require("./dns.js");
var apigatewayv2_authorizer_1 = require("./apigatewayv2-authorizer");
var apigateway_websocket_route_1 = require("./apigateway-websocket-route");
var apigateway_account_1 = require("./helpers/apigateway-account");
var aws_1 = require("@pulumi/aws");
var permission_1 = require("./permission");
var error_1 = require("../error");
/**
 * The `ApiGatewayWebSocket` component lets you add an [Amazon API Gateway WebSocket API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-websocket-api.html)
 * to your app.
 *
 * @example
 *
 * #### Create the API
 *
 * ```ts title="sst.config.ts"
 * const api = new sst.aws.ApiGatewayWebSocket("MyApi");
 * ```
 *
 * #### Add a custom domain
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.ApiGatewayWebSocket("MyApi", {
 *   domain: "api.example.com"
 * });
 * ```
 *
 * #### Add routes
 *
 * ```ts title="sst.config.ts"
 * api.route("$connect", "src/connect.handler");
 * api.route("$disconnect", "src/disconnect.handler");
 * api.route("$default", "src/default.handler");
 * api.route("sendMessage", "src/sendMessage.handler");
 * ```
 */
var ApiGatewayWebSocket = /** @class */ (function (_super) {
    __extends(ApiGatewayWebSocket, _super);
    function ApiGatewayWebSocket(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var accessLog = normalizeAccessLog();
        var domain = normalizeDomain();
        var apigAccount = (0, apigateway_account_1.setupApiGatewayAccount)(name, opts);
        var api = createApi();
        var logGroup = createLogGroup();
        var stage = createStage();
        var certificateArn = createSsl();
        var apigDomain = createDomainName();
        createDnsRecords();
        var apiMapping = createDomainMapping();
        _this.constructorName = name;
        _this.constructorArgs = args;
        _this.constructorOpts = opts;
        _this.api = api;
        _this.stage = stage;
        _this.apigDomain = apigDomain;
        _this.apiMapping = apiMapping;
        _this.logGroup = logGroup;
        _this.registerOutputs({
            _hint: _this.url,
        });
        function normalizeAccessLog() {
            return (0, pulumi_1.output)(args.accessLog).apply(function (accessLog) {
                var _a;
                return (__assign(__assign({}, accessLog), { retention: (_a = accessLog === null || accessLog === void 0 ? void 0 : accessLog.retention) !== null && _a !== void 0 ? _a : "1 month" }));
            });
        }
        function normalizeDomain() {
            if (!args.domain)
                return;
            return (0, pulumi_1.output)(args.domain).apply(function (domain) {
                var _a;
                // validate
                if (typeof domain !== "string") {
                    if (domain.name && domain.nameId)
                        throw new error_1.VisibleError("Cannot configure both domain \"name\" and \"nameId\" for the \"".concat(name, "\" API."));
                    if (!domain.name && !domain.nameId)
                        throw new error_1.VisibleError("Either domain \"name\" or \"nameId\" is required for the \"".concat(name, "\" API."));
                    if (domain.dns === false && !domain.cert)
                        throw new error_1.VisibleError("Domain \"cert\" is required when \"dns\" is disabled for the \"".concat(name, "\" API."));
                }
                // normalize
                var norm = typeof domain === "string" ? { name: domain } : domain;
                return {
                    name: norm.name,
                    nameId: norm.nameId,
                    path: norm.path,
                    dns: norm.dns === false ? undefined : (_a = norm.dns) !== null && _a !== void 0 ? _a : (0, dns_js_1.dns)(),
                    cert: norm.cert,
                };
            });
        }
        function createApi() {
            var _a;
            var _b;
            return new ((_a = aws_1.apigatewayv2.Api).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.api, "".concat(name, "Api"), {
                protocolType: "WEBSOCKET",
                routeSelectionExpression: "$request.body.action",
            }, { parent: parent }), false)))();
        }
        function createLogGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.cloudwatch.LogGroup).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.accessLog, "".concat(name, "AccessLog"), {
                name: "/aws/vendedlogs/apis/".concat((0, naming_1.physicalName)(64, name)),
                retentionInDays: accessLog.apply(function (accessLog) { return logging_1.RETENTION[accessLog.retention]; }),
            }, { parent: parent, ignoreChanges: ["name"] }), false)))();
        }
        function createStage() {
            var _a;
            var _b;
            return new ((_a = aws_1.apigatewayv2.Stage).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.stage, "".concat(name, "Stage"), {
                apiId: api.id,
                autoDeploy: true,
                name: "$default",
                accessLogSettings: {
                    destinationArn: logGroup.arn,
                    format: JSON.stringify({
                        // request info
                        requestTime: "\"$context.requestTime\"",
                        requestId: "\"$context.requestId\"",
                        eventType: "\"$context.eventType\"",
                        routeKey: "\"$context.routeKey\"",
                        status: "$context.status", // integer value, do not wrap in quotes
                        // integration info
                        integrationRequestId: "\"$context.awsEndpointRequestId\"",
                        integrationStatus: "\"$context.integrationStatus\"",
                        integrationLatency: "\"$context.integrationLatency\"",
                        integrationServiceStatus: "\"$context.integration.integrationStatus\"",
                        // caller info
                        ip: "\"$context.identity.sourceIp\"",
                        userAgent: "\"$context.identity.userAgent\"",
                        //cognitoIdentityId:`"$context.identity.cognitoIdentityId"`, // not supported in us-west-2 region
                        connectedAt: "\"$context.connectedAt\"",
                        connectionId: "\"$context.connectionId\"",
                    }),
                },
            }, { parent: parent, dependsOn: apigAccount }), false)))();
        }
        function createSsl() {
            if (!domain)
                return (0, pulumi_1.output)(undefined);
            return domain.apply(function (domain) {
                if (domain.cert)
                    return (0, pulumi_1.output)(domain.cert);
                if (domain.nameId)
                    return (0, pulumi_1.output)(undefined);
                return new dns_validated_certificate_1.DnsValidatedCertificate("".concat(name, "Ssl"), {
                    domainName: domain.name,
                    dns: domain.dns,
                }, { parent: parent }).arn;
            });
        }
        function createDomainName() {
            if (!domain || !certificateArn)
                return;
            return (0, pulumi_1.all)([domain, certificateArn]).apply(function (_a) {
                var _b;
                var _c;
                var domain = _a[0], certificateArn = _a[1];
                return domain.nameId
                    ? aws_1.apigatewayv2.DomainName.get("".concat(name, "DomainName"), domain.nameId, {}, { parent: parent })
                    : new ((_b = aws_1.apigatewayv2.DomainName).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.domainName, "".concat(name, "DomainName"), {
                        domainName: domain.name,
                        domainNameConfiguration: {
                            certificateArn: certificateArn,
                            endpointType: "REGIONAL",
                            securityPolicy: "TLS_1_2",
                        },
                    }, { parent: parent }), false)))();
            });
        }
        function createDnsRecords() {
            if (!domain || !apigDomain)
                return;
            domain.apply(function (domain) {
                if (!domain.dns)
                    return;
                if (domain.nameId)
                    return;
                domain.dns.createAlias(name, {
                    name: domain.name,
                    aliasName: apigDomain.domainNameConfiguration.targetDomainName,
                    aliasZone: apigDomain.domainNameConfiguration.hostedZoneId,
                }, { parent: parent });
            });
        }
        function createDomainMapping() {
            var _a;
            if (!domain || !apigDomain)
                return;
            return (_a = domain.path) === null || _a === void 0 ? void 0 : _a.apply(function (path) {
                return new aws_1.apigatewayv2.ApiMapping("".concat(name, "DomainMapping"), {
                    apiId: api.id,
                    domainName: apigDomain.id,
                    stage: "$default",
                    apiMappingKey: path,
                }, { parent: parent });
            });
        }
        return _this;
    }
    Object.defineProperty(ApiGatewayWebSocket.prototype, "url", {
        /**
         * The URL of the API.
         *
         * If the `domain` is set, this is the URL with the custom domain.
         * Otherwise, it's the auto-generated API Gateway URL.
         */
        get: function () {
            // Note: If mapping key is set, the URL needs a trailing slash. Without the
            //       trailing slash, the API fails with the error {"message":"Not Found"}
            return this.apigDomain && this.apiMapping
                ? (0, pulumi_1.all)([this.apigDomain.domainName, this.apiMapping.apiMappingKey]).apply(function (_a) {
                    var domain = _a[0], key = _a[1];
                    return key ? "wss://".concat(domain, "/").concat(key, "/") : "wss://".concat(domain);
                })
                : (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["", "/", ""], ["", "/", ""])), this.api.apiEndpoint, this.stage.name);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(ApiGatewayWebSocket.prototype, "managementEndpoint", {
        /**
         * The management endpoint for the API used by the API Gateway Management API client.
         * This is useful for sending messages to connected clients.
         *
         * @example
         * ```js
         * import { Resource } from "sst";
         * import { ApiGatewayManagementApiClient } from "@aws-sdk/client-apigatewaymanagementapi";
         *
         * const client = new ApiGatewayManagementApiClient({
         *   endpoint: Resource.MyApi.managementEndpoint,
         * });
         * ```
         */
        get: function () {
            var _this = this;
            // ie. https://v1lmfez2nj.execute-api.us-east-1.amazonaws.com/$default
            return this.api.apiEndpoint.apply(function (endpoint) {
                return (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["", "/", ""], ["", "/", ""])), endpoint.replace("wss", "https"), _this.stage.name);
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(ApiGatewayWebSocket.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var self = this;
            return {
                /**
                 * The Amazon API Gateway V2 API.
                 */
                api: this.api,
                /**
                 * The API Gateway HTTP API domain name.
                 */
                get domainName() {
                    if (!self.apigDomain)
                        throw new error_1.VisibleError("\"nodes.domainName\" is not available when domain is not configured for the \"".concat(self.constructorName, "\" API."));
                    return self.apigDomain;
                },
                /**
                 * The CloudWatch LogGroup for the access logs.
                 */
                logGroup: this.logGroup,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Add a route to the API Gateway WebSocket API.
     *
     * There are three predefined routes:
     * - `$connect`: When the client connects to the API.
     * - `$disconnect`: When the client or the server disconnects from the API.
     * - `$default`: The default or catch-all route.
     *
     * In addition, you can create custom routes. When a request comes in, the API Gateway
     * will look for the specific route defined by the user. If no route matches, the `$default`
     * route will be invoked.
     *
     * @param route The path for the route.
     * @param handler The function that'll be invoked.
     * @param args Configure the route.
     *
     * @example
     * Add a simple route.
     *
     * ```js title="sst.config.ts"
     * api.route("sendMessage", "src/sendMessage.handler");
     * ```
     *
     * Add a predefined route.
     *
     * ```js title="sst.config.ts"
     * api.route("$default", "src/default.handler");
     * ```
     *
     * Enable auth for a route.
     *
     * ```js title="sst.config.ts"
     * api.route("sendMessage", "src/sendMessage.handler", {
     *   auth: {
     *     iam: true
     *   }
     * });
     * ```
     *
     * Customize the route handler.
     *
     * ```js title="sst.config.ts"
     * api.route("sendMessage", {
     *   handler: "src/sendMessage.handler",
     *   memory: "2048 MB"
     * });
     * ```
     *
     * Or pass in the ARN of an existing Lambda function.
     *
     * ```js title="sst.config.ts"
     * api.route("sendMessage", "arn:aws:lambda:us-east-1:123456789012:function:my-function");
     * ```
     */
    ApiGatewayWebSocket.prototype.route = function (route, handler, args) {
        var _a, _b, _c, _d;
        if (args === void 0) { args = {}; }
        var prefix = this.constructorName;
        var suffix = (0, naming_1.logicalName)(["$connect", "$disconnect", "$default"].includes(route)
            ? route
            : (0, naming_1.hashStringToPrettyString)("".concat(component_1.outputId).concat(route), 6));
        var transformed = (0, component_1.transform)((_b = (_a = this.constructorArgs.transform) === null || _a === void 0 ? void 0 : _a.route) === null || _b === void 0 ? void 0 : _b.args, "".concat(prefix, "Route").concat(suffix), args, { provider: this.constructorOpts.provider });
        return new apigateway_websocket_route_1.ApiGatewayWebSocketRoute(transformed[0], __assign({ api: {
                name: prefix,
                id: this.api.id,
                executionArn: this.api.executionArn,
            }, route: route, handler: handler, handlerTransform: (_d = (_c = this.constructorArgs.transform) === null || _c === void 0 ? void 0 : _c.route) === null || _d === void 0 ? void 0 : _d.handler }, transformed[1]), transformed[2]);
    };
    /**
     * Add an authorizer to the API Gateway WebSocket API.
     *
     * @param name The name of the authorizer.
     * @param args Configure the authorizer.
     *
     * @example
     * Add a Lambda authorizer.
     *
     * ```js title="sst.config.ts"
     * api.addAuthorizer({
     *   name: "myAuthorizer",
     *   lambda: {
     *     function: "src/authorizer.index"
     *   }
     * });
     * ```
     *
     * Add a JWT authorizer.
     *
     * ```js title="sst.config.ts"
     * const authorizer = api.addAuthorizer({
     *   name: "myAuthorizer",
     *   jwt: {
     *     issuer: "https://issuer.com/",
     *     audiences: ["https://api.example.com"],
     *     identitySource: "$request.header.AccessToken"
     *   }
     * });
     * ```
     *
     * Add a Cognito UserPool as a JWT authorizer.
     *
     * ```js title="sst.config.ts"
     * const pool = new sst.aws.CognitoUserPool("MyUserPool");
     * const poolClient = userPool.addClient("Web");
     *
     * const authorizer = api.addAuthorizer({
     *   name: "myCognitoAuthorizer",
     *   jwt: {
     *     issuer: $interpolate`https://cognito-idp.${aws.getRegionOutput().name}.amazonaws.com/${pool.id}`,
     *     audiences: [poolClient.id]
     *   }
     * });
     * ```
     *
     * Now you can use the authorizer in your routes.
     *
     * ```js title="sst.config.ts"
     * api.route("GET /", "src/get.handler", {
     *   auth: {
     *     jwt: {
     *       authorizer: authorizer.id
     *     }
     *   }
     * });
     * ```
     */
    ApiGatewayWebSocket.prototype.addAuthorizer = function (name, args) {
        var self = this;
        var constructorName = this.constructorName;
        return new apigatewayv2_authorizer_1.ApiGatewayV2Authorizer("".concat(constructorName, "Authorizer").concat(name), __assign({ api: {
                id: self.api.id,
                name: constructorName,
                executionArn: this.api.executionArn,
            }, type: "websocket", name: name }, args), { provider: this.constructorOpts.provider });
    };
    /** @internal */
    ApiGatewayWebSocket.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
                managementEndpoint: this.managementEndpoint,
            },
            include: [
                (0, permission_1.permission)({
                    actions: ["execute-api:ManageConnections"],
                    resources: [(0, pulumi_1.interpolate)(templateObject_3 || (templateObject_3 = __makeTemplateObject(["", "/*/*/@connections/*"], ["", "/*/*/@connections/*"])), this.api.executionArn)],
                }),
            ],
        };
    };
    return ApiGatewayWebSocket;
}(component_1.Component));
exports.ApiGatewayWebSocket = ApiGatewayWebSocket;
var __pulumiType = "sst:aws:ApiGatewayWebSocket";
// @ts-expect-error
ApiGatewayWebSocket.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2, templateObject_3;
