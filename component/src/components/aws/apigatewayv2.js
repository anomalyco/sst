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
exports.ApiGatewayV2 = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var naming_1 = require("../naming");
var error_1 = require("../error");
var dns_validated_certificate_1 = require("./dns-validated-certificate");
var logging_1 = require("./logging");
var dns_1 = require("./dns");
var apigatewayv2_lambda_route_1 = require("./apigatewayv2-lambda-route");
var apigatewayv2_authorizer_1 = require("./apigatewayv2-authorizer");
var aws_1 = require("@pulumi/aws");
var apigatewayv2_url_route_1 = require("./apigatewayv2-url-route");
var duration_1 = require("../duration");
var apigatewayv2_private_route_1 = require("./apigatewayv2-private-route");
var vpc_1 = require("./vpc");
/**
 * The `ApiGatewayV2` component lets you add an [Amazon API Gateway HTTP API](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api.html) to your app.
 *
 * @example
 *
 * #### Create the API
 *
 * ```ts title="sst.config.ts"
 * const api = new sst.aws.ApiGatewayV2("MyApi");
 * ```
 *
 * #### Add a custom domain
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.ApiGatewayV2("MyApi", {
 *   domain: "api.example.com"
 * });
 * ```
 *
 * #### Add routes
 *
 * ```ts title="sst.config.ts"
 * api.route("GET /", "src/get.handler");
 * api.route("POST /", "src/post.handler");
 * ```
 *
 * #### Configure the routes
 *
 * You can configure the route.
 *
 * ```ts title="sst.config.ts"
 * api.route("GET /", "src/get.handler", {
 *   auth: { iam: true }
 * });
 * ```
 *
 * #### Configure the route handler
 *
 * You can configure the route handler function.
 *
 * ```ts title="sst.config.ts"
 * api.route("POST /", {
 *   handler: "src/post.handler",
 *   memory: "2048 MB"
 * });
 * ```
 *
 * #### Default props for all routes
 *
 * You can use the `transform` to set some default props for all your routes. For example,
 * instead of setting the `memory` for each route.
 *
 * ```ts title="sst.config.ts"
 * api.route("GET /", { handler: "src/get.handler", memory: "2048 MB" });
 * api.route("POST /", { handler: "src/post.handler", memory: "2048 MB" });
 * ```
 *
 * You can set it through the `transform`.
 *
 * ```ts title="sst.config.ts" {6}
 * const api = new sst.aws.ApiGatewayV2("MyApi", {
 *   transform: {
 *     route: {
 *       handler: (args, opts) => {
 *         // Set the default if it's not set by the route
 *         args.memory ??= "2048 MB";
 *       }
 *     }
 *   }
 * });
 *
 * api.route("GET /", "src/get.handler");
 * api.route("POST /", "src/post.handler");
 * ```
 *
 * With this we set the `memory` if it's not overridden by the route.
 */
var ApiGatewayV2 = /** @class */ (function (_super) {
    __extends(ApiGatewayV2, _super);
    function ApiGatewayV2(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var accessLog = normalizeAccessLog();
        var domain = normalizeDomain();
        var cors = normalizeCors();
        var vpc = normalizeVpc();
        var vpcLink = createVpcLink();
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
        _this.apigDomain = apigDomain;
        _this.apiMapping = apiMapping;
        _this.logGroup = logGroup;
        _this.vpcLink = vpcLink;
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
                    dns: norm.dns === false ? undefined : (_a = norm.dns) !== null && _a !== void 0 ? _a : (0, dns_1.dns)(),
                    cert: norm.cert,
                };
            });
        }
        function normalizeCors() {
            return (0, pulumi_1.output)(args.cors).apply(function (cors) {
                if (cors === false)
                    return {};
                var defaultCors = {
                    allowHeaders: ["*"],
                    allowMethods: ["*"],
                    allowOrigins: ["*"],
                };
                return cors === true || cors === undefined
                    ? defaultCors
                    : __assign(__assign(__assign({}, defaultCors), cors), { maxAge: cors.maxAge && (0, duration_1.toSeconds)(cors.maxAge) });
            });
        }
        function normalizeVpc() {
            // "vpc" is undefined
            if (!args.vpc)
                return;
            // "vpc" is a Vpc component
            if (args.vpc instanceof vpc_1.Vpc) {
                return {
                    subnets: args.vpc.publicSubnets,
                    securityGroups: args.vpc.securityGroups,
                };
            }
            // "vpc" is object
            return (0, pulumi_1.output)(args.vpc);
        }
        function createVpcLink() {
            var _a;
            var _b;
            if (!vpc)
                return;
            return new ((_a = aws_1.apigatewayv2.VpcLink).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.vpcLink, "".concat(name, "VpcLink"), {
                securityGroupIds: vpc.securityGroups,
                subnetIds: vpc.subnets,
            }, { parent: parent }), false)))();
        }
        function createApi() {
            var _a;
            var _b;
            return new ((_a = aws_1.apigatewayv2.Api).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.api, "".concat(name, "Api"), {
                protocolType: "HTTP",
                corsConfiguration: cors,
            }, { parent: parent }), false)))();
        }
        function createLogGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.cloudwatch.LogGroup).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.logGroup, "".concat(name, "AccessLog"), {
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
                        httpMethod: "\"$context.httpMethod\"",
                        path: "\"$context.path\"",
                        routeKey: "\"$context.routeKey\"",
                        status: "$context.status", // integer value, do not wrap in quotes
                        responseLatency: "$context.responseLatency", // integer value, do not wrap in quotes
                        // integration info
                        integrationRequestId: "\"$context.integration.requestId\"",
                        integrationStatus: "\"$context.integration.status\"",
                        integrationLatency: "\"$context.integration.latency\"",
                        integrationServiceStatus: "\"$context.integration.integrationStatus\"",
                        // caller info
                        ip: "\"$context.identity.sourceIp\"",
                        userAgent: "\"$context.identity.userAgent\"",
                        //cognitoIdentityId:`"$context.identity.cognitoIdentityId"`, // not supported in us-west-2 region
                    }),
                },
            }, { parent: parent }), false)))();
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
            return (0, pulumi_1.output)(domain).apply(function (domain) {
                var _a;
                var _b;
                return domain.nameId
                    ? aws_1.apigatewayv2.DomainName.get("".concat(name, "DomainName"), domain.nameId, {}, { parent: parent })
                    : new ((_a = aws_1.apigatewayv2.DomainName).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.domainName, "".concat(name, "DomainName"), {
                        domainName: domain.name,
                        domainNameConfiguration: certificateArn.apply(function (certificateArn) { return ({
                            certificateArn: certificateArn,
                            endpointType: "REGIONAL",
                            securityPolicy: "TLS_1_2",
                        }); }),
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
                    stage: stage.name,
                    apiMappingKey: path,
                }, { parent: parent });
            });
        }
        return _this;
    }
    Object.defineProperty(ApiGatewayV2.prototype, "url", {
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
                    return key ? "https://".concat(domain, "/").concat(key, "/") : "https://".concat(domain);
                })
                : this.api.apiEndpoint;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(ApiGatewayV2.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var self = this;
            return {
                /**
                 * The Amazon API Gateway HTTP API.
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
                /**
                 * The API Gateway HTTP API VPC link.
                 */
                vpcLink: this.vpcLink,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Add a route to the API Gateway HTTP API. The route is a combination of
     * - An HTTP method and a path, `{METHOD} /{path}`.
     * - Or a `$default` route.
     *
     * :::tip
     * The `$default` route is a default or catch-all route. It'll match if no other route matches.
     * :::
     *
     * A method could be one of `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, or `ANY`. Here `ANY` matches any HTTP method.
     *
     * The path can be a combination of
     * - Literal segments, `/notes`, `/notes/new`, etc.
     * - Parameter segments, `/notes/{noteId}`, `/notes/{noteId}/attachments/{attachmentId}`, etc.
     * - Greedy segments, `/{proxy+}`, `/notes/{proxy+}`,  etc. The `{proxy+}` segment is a greedy segment that matches all child paths. It needs to be at the end of the path.
     *
     * :::tip
     * The `{proxy+}` is a greedy segment, it matches all its child paths.
     * :::
     *
     * The `$default` is a reserved keyword for the default route. It'll be matched if no other route matches.
     *
     * :::note
     * You cannot have duplicate routes.
     * :::
     *
     * When a request comes in, the API Gateway will look for the most specific match. If no route matches, the `$default` route will be invoked.
     *
     * @param rawRoute The path for the route.
     * @param handler The function that'll be invoked.
     * @param args Configure the route.
     *
     * @example
     * Add a simple route.
     *
     * ```js title="sst.config.ts"
     * api.route("GET /", "src/get.handler");
     * ```
     *
     * Match any HTTP method.
     *
     * ```js title="sst.config.ts"
     * api.route("ANY /", "src/route.handler");
     * ```
     *
     * Add a default route.
     *
     * ```js title="sst.config.ts"
     * api.route("GET /", "src/get.handler")
     * api.route("$default", "src/default.handler");
     * ```
     *
     * Add a parameterized route.
     *
     * ```js title="sst.config.ts"
     * api.route("GET /notes/{id}", "src/get.handler");
     * ```
     *
     * Add a greedy route.
     *
     * ```js title="sst.config.ts"
     * api.route("GET /notes/{proxy+}", "src/greedy.handler");
     * ```
     *
     * Enable auth for a route.
     *
     * ```js title="sst.config.ts"
     * api.route("GET /", "src/get.handler")
     * api.route("POST /", "src/post.handler", {
     *   auth: {
     *     iam: true
     *   }
     * });
     * ```
     *
     * Customize the route handler.
     *
     * ```js title="sst.config.ts"
     * api.route("GET /", {
     *   handler: "src/get.handler",
     *   memory: "2048 MB"
     * });
     * ```
     *
     * Or pass in the ARN of an existing Lambda function.
     *
     * ```js title="sst.config.ts"
     * api.route("GET /", "arn:aws:lambda:us-east-1:123456789012:function:my-function");
     * ```
     */
    ApiGatewayV2.prototype.route = function (rawRoute, handler, args) {
        var _a, _b, _c, _d;
        if (args === void 0) { args = {}; }
        var route = this.parseRoute(rawRoute);
        var transformed = (0, component_1.transform)((_b = (_a = this.constructorArgs.transform) === null || _a === void 0 ? void 0 : _a.route) === null || _b === void 0 ? void 0 : _b.args, this.buildRouteId(route), args, { provider: this.constructorOpts.provider });
        return new apigatewayv2_lambda_route_1.ApiGatewayV2LambdaRoute(transformed[0], __assign({ api: {
                name: this.constructorName,
                id: this.api.id,
                executionArn: this.api.executionArn,
            }, route: route, handler: handler, handlerLink: this.constructorArgs.link, handlerTransform: (_d = (_c = this.constructorArgs.transform) === null || _c === void 0 ? void 0 : _c.route) === null || _d === void 0 ? void 0 : _d.handler }, transformed[1]), transformed[2]);
    };
    /**
     * Add a URL route to the API Gateway HTTP API.
     *
     * @param rawRoute The path for the route.
     * @param url The URL to forward to.
     * @param args Configure the route.
     *
     * @example
     * Add a simple route.
     *
     * ```js title="sst.config.ts"
     * api.routeUrl("GET /", "https://google.com");
     * ```
     *
     * Enable auth for a route.
     *
     * ```js title="sst.config.ts"
     * api.routeUrl("POST /", "https://google.com", {
     *   auth: {
     *     iam: true
     *   }
     * });
     * ```
     */
    ApiGatewayV2.prototype.routeUrl = function (rawRoute, url, args) {
        var _a, _b;
        if (args === void 0) { args = {}; }
        var route = this.parseRoute(rawRoute);
        var transformed = (0, component_1.transform)((_b = (_a = this.constructorArgs.transform) === null || _a === void 0 ? void 0 : _a.route) === null || _b === void 0 ? void 0 : _b.args, this.buildRouteId(route), args, { provider: this.constructorOpts.provider });
        return new apigatewayv2_url_route_1.ApiGatewayV2UrlRoute(transformed[0], __assign({ api: {
                name: this.constructorName,
                id: this.api.id,
                executionArn: this.api.executionArn,
            }, route: route, url: url }, transformed[1]), transformed[2]);
    };
    /**
     * Adds a private route to the API Gateway HTTP API.
     *
     * To add private routes, you need to have a VPC link. Make sure to pass in a `vpc`.
     * Learn more about [adding private routes](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-develop-integrations-private.html).
     *
     * :::tip
     * You need to pass `vpc` to add a private route.
     * :::
     *
     * A couple of things to note:
     *
     * 1. Your API Gateway HTTP API also needs to be in the **same VPC** as the service.
     *
     * 2. You also need to verify that your VPC's [**availability zones support VPC link**](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-vpc-links.html#http-api-vpc-link-availability).
     *
     * 3. Run `aws ec2 describe-availability-zones` to get a list of AZs for your
     *    account.
     *
     * 4. Only list the AZ ID's that support VPC link.
     *    ```ts title="sst.config.ts" {4}
     *    vpc: {
     *      az: ["eu-west-3a", "eu-west-3c"]
     *    }
     *    ```
     *    If the VPC picks an AZ automatically that doesn't support VPC link, you'll get
     *    the following error:
     *    ```
     *    operation error ApiGatewayV2: BadRequestException: Subnet is in Availability
     *    Zone 'euw3-az2' where service is not available
     *    ```
     *
     * @param rawRoute The path for the route.
     * @param arn The ARN of the AWS Load Balancer or Cloud Map service.
     * @param args Configure the route.
     *
     * @example
     * Here are a few examples using the private route. Add a route to Application Load Balancer.
     *
     * ```js title="sst.config.ts"
     * const loadBalancerArn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-load-balancer/50dc6c495c0c9188";
     * api.routePrivate("GET /", loadBalancerArn);
     * ```
     *
     * Add a route to AWS Cloud Map service.
     *
     * ```js title="sst.config.ts"
     * const serviceArn = "arn:aws:servicediscovery:us-east-2:123456789012:service/srv-id?stage=prod&deployment=green_deployment";
     * api.routePrivate("GET /", serviceArn);
     * ```
     *
     * Enable IAM authentication for a route.
     *
     * ```js title="sst.config.ts"
     * api.routePrivate("GET /", serviceArn, {
     *   auth: {
     *     iam: true
     *   }
     * });
     * ```
     */
    ApiGatewayV2.prototype.routePrivate = function (rawRoute, arn, args) {
        var _a, _b;
        if (args === void 0) { args = {}; }
        if (!this.vpcLink)
            throw new error_1.VisibleError("To add private routes, you need to have a VPC link. Configure \"vpc\" for the \"".concat(this.constructorName, "\" API to create a VPC link."));
        var route = this.parseRoute(rawRoute);
        var transformed = (0, component_1.transform)((_b = (_a = this.constructorArgs.transform) === null || _a === void 0 ? void 0 : _a.route) === null || _b === void 0 ? void 0 : _b.args, this.buildRouteId(route), args, { provider: this.constructorOpts.provider });
        return new apigatewayv2_private_route_1.ApiGatewayV2PrivateRoute(transformed[0], __assign({ api: {
                name: this.constructorName,
                id: this.api.id,
                executionArn: this.api.executionArn,
            }, route: route, vpcLink: this.vpcLink.id, arn: arn }, transformed[1]), transformed[2]);
    };
    ApiGatewayV2.prototype.parseRoute = function (rawRoute) {
        if (rawRoute.toLowerCase() === "$default")
            return "$default";
        var parts = rawRoute.split(" ");
        if (parts.length !== 2) {
            throw new error_1.VisibleError("Invalid route ".concat(rawRoute, ". A route must be in the format \"METHOD /path\"."));
        }
        var _a = rawRoute.split(" "), methodRaw = _a[0], path = _a[1];
        var method = methodRaw.toUpperCase();
        if (![
            "ANY",
            "DELETE",
            "GET",
            "HEAD",
            "OPTIONS",
            "PATCH",
            "POST",
            "PUT",
        ].includes(method))
            throw new error_1.VisibleError("Invalid method ".concat(methodRaw, " in route ").concat(rawRoute));
        if (!path.startsWith("/"))
            throw new error_1.VisibleError("Invalid path ".concat(path, " in route ").concat(rawRoute, ". Path must start with \"/\"."));
        return "".concat(method, " ").concat(path);
    };
    ApiGatewayV2.prototype.buildRouteId = function (route) {
        var suffix = (0, naming_1.logicalName)((0, naming_1.hashStringToPrettyString)([component_1.outputId, route].join(""), 6));
        return "".concat(this.constructorName, "Route").concat(suffix);
    };
    /**
     * Add an authorizer to the API Gateway HTTP API.
     *
     * @param args Configure the authorizer.
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
    ApiGatewayV2.prototype.addAuthorizer = function (args) {
        var self = this;
        var selfName = this.constructorName;
        var nameSuffix = (0, naming_1.logicalName)(args.name);
        return new apigatewayv2_authorizer_1.ApiGatewayV2Authorizer("".concat(selfName, "Authorizer").concat(nameSuffix), __assign({ api: {
                id: self.api.id,
                name: selfName,
                executionArn: this.api.executionArn,
            }, type: "http" }, args), { provider: this.constructorOpts.provider });
    };
    /** @internal */
    ApiGatewayV2.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
            },
        };
    };
    return ApiGatewayV2;
}(component_1.Component));
exports.ApiGatewayV2 = ApiGatewayV2;
var __pulumiType = "sst:aws:ApiGatewayV2";
// @ts-expect-error
ApiGatewayV2.__pulumiType = __pulumiType;
