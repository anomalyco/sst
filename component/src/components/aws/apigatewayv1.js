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
exports.ApiGatewayV1 = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var naming_1 = require("../naming");
var error_1 = require("../error");
var logging_1 = require("./logging");
var apigatewayv1_lambda_route_1 = require("./apigatewayv1-lambda-route");
var apigatewayv1_authorizer_1 = require("./apigatewayv1-authorizer");
var apigateway_account_1 = require("./helpers/apigateway-account");
var aws_1 = require("@pulumi/aws");
var dns_1 = require("./dns");
var dns_validated_certificate_1 = require("./dns-validated-certificate");
var apigatewayv1_integration_route_1 = require("./apigatewayv1-integration-route");
var apigatewayv1_usage_plan_1 = require("./apigatewayv1-usage-plan");
var provider_1 = require("./helpers/provider");
/**
 * The `ApiGatewayV1` component lets you add an [Amazon API Gateway REST API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html) to your app.
 *
 * @example
 *
 * #### Create the API
 *
 * ```ts title="sst.config.ts"
 * const api = new sst.aws.ApiGatewayV1("MyApi");
 * ```
 *
 * #### Add routes
 *
 * ```ts title="sst.config.ts"
 * api.route("GET /", "src/get.handler");
 * api.route("POST /", "src/post.handler");
 *
 * api.deploy();
 * ```
 *
 * :::note
 * You need to call the `deploy` method after you've added all your routes.
 * :::
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
 * const api = new sst.aws.ApiGatewayV1("MyApi", {
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
var ApiGatewayV1 = /** @class */ (function (_super) {
    __extends(ApiGatewayV1, _super);
    function ApiGatewayV1(name, args, opts) {
        if (args === void 0) { args = {}; }
        if (opts === void 0) { opts = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        _this.resources = {};
        _this.routes = [];
        _this.deployed = false;
        var parent = _this;
        var region = normalizeRegion();
        var endpoint = normalizeEndpoint();
        var apigAccount = (0, apigateway_account_1.setupApiGatewayAccount)(name, opts);
        var api = createApi();
        _this.resources["/"] = api.rootResourceId;
        _this.constructorName = name;
        _this.constructorArgs = args;
        _this.constructorOpts = opts;
        _this.api = api;
        _this.region = region;
        _this.endpointType = endpoint.types;
        function normalizeRegion() {
            return (0, aws_1.getRegionOutput)(undefined, { parent: parent }).name;
        }
        function normalizeEndpoint() {
            return (0, pulumi_1.output)(args.endpoint).apply(function (endpoint) {
                if (!endpoint)
                    return { types: "EDGE" };
                if (endpoint.type === "private" && !endpoint.vpcEndpointIds)
                    throw new error_1.VisibleError("Please provide the VPC endpoint IDs for the private endpoint.");
                return endpoint.type === "regional"
                    ? { types: "REGIONAL" }
                    : endpoint.type === "private"
                        ? {
                            types: "PRIVATE",
                            vpcEndpointIds: endpoint.vpcEndpointIds,
                        }
                        : { types: "EDGE" };
            });
        }
        function createApi() {
            var _a;
            var _b;
            return new ((_a = aws_1.apigateway.RestApi).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.api, "".concat(name, "Api"), {
                endpointConfiguration: endpoint,
            }, { parent: parent, dependsOn: apigAccount }), false)))();
        }
        return _this;
    }
    Object.defineProperty(ApiGatewayV1.prototype, "url", {
        /**
         * The URL of the API.
         */
        get: function () {
            return this.apigDomain && this.apiMapping
                ? (0, pulumi_1.all)([this.apigDomain.domainName, this.apiMapping.basePath]).apply(function (_a) {
                    var domain = _a[0], key = _a[1];
                    return key ? "https://".concat(domain, "/").concat(key, "/") : "https://".concat(domain);
                })
                : (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["https://", ".execute-api.", ".amazonaws.com/", "/"], ["https://", ".execute-api.", ".amazonaws.com/", "/"])), this.api.id, this.region, $app.stage);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(ApiGatewayV1.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var self = this;
            return {
                /**
                 * The Amazon API Gateway REST API
                 */
                api: this.api,
                /**
                 * The Amazon API Gateway REST API stage
                 */
                stage: this.stage,
                /**
                 * The CloudWatch LogGroup for the access logs.
                 */
                logGroup: this.logGroup,
                /**
                 * The API Gateway REST API domain name.
                 */
                get domainName() {
                    if (!self.deployed)
                        throw new error_1.VisibleError("\"nodes.domainName\" is not available before the \"".concat(self.constructorName, "\" API is deployed."));
                    if (!self.apigDomain)
                        throw new error_1.VisibleError("\"nodes.domainName\" is not available when domain is not configured for the \"".concat(self.constructorName, "\" API."));
                    return self.apigDomain;
                },
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Add a route to the API Gateway REST API. The route is a combination of an HTTP method and a path, `{METHOD} /{path}`.
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
     * When a request comes in, the API Gateway will look for the most specific match.
     *
     * :::note
     * You cannot have duplicate routes.
     * :::
     *
     * @param route The path for the route.
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
    ApiGatewayV1.prototype.route = function (route, handler, args) {
        var _a, _b, _c, _d;
        if (args === void 0) { args = {}; }
        var _e = this.parseRoute(route), method = _e.method, path = _e.path;
        this.createResource(path);
        var transformed = (0, component_1.transform)((_b = (_a = this.constructorArgs.transform) === null || _a === void 0 ? void 0 : _a.route) === null || _b === void 0 ? void 0 : _b.args, this.buildRouteId(method, path), args, { provider: this.constructorOpts.provider });
        var apigRoute = new apigatewayv1_lambda_route_1.ApiGatewayV1LambdaRoute(transformed[0], __assign({ api: {
                name: this.constructorName,
                id: this.api.id,
                executionArn: this.api.executionArn,
            }, method: method, path: path, resourceId: this.resources[path], handler: handler, handlerTransform: (_d = (_c = this.constructorArgs.transform) === null || _c === void 0 ? void 0 : _c.route) === null || _d === void 0 ? void 0 : _d.handler }, transformed[1]), transformed[2]);
        this.routes.push(apigRoute);
        return apigRoute;
    };
    /**
     * Add a custom integration to the API Gateway REST API.
     *
     * Learn more about [integrations for REST APIs](https://docs.aws.amazon.com/apigateway/latest/developerguide/how-to-integration-settings.html).
     *
     * @param route The path for the route.
     * @param integration The integration configuration.
     * @param args Configure the route.
     *
     * @example
     * Add a route to trigger a Step Functions state machine execution.
     *
     * ```js title="sst.config.ts"
     * api.routeIntegration("POST /run-my-state-machine", {
     *   type: "aws",
     *   uri: "arn:aws:apigateway:us-east-1:states:startExecution",
     *   credentials: "arn:aws:iam::123456789012:role/apigateway-execution-role",
     *   integrationHttpMethod: "POST",
     *   requestTemplates: {
     *     "application/json": JSON.stringify({
     *       input: "$input.json('$')",
     *       stateMachineArn: "arn:aws:states:us-east-1:123456789012:stateMachine:MyStateMachine",
     *     }),
     *   },
     *   passthroughBehavior: "when-no-match",
     * });
     * ```
     */
    ApiGatewayV1.prototype.routeIntegration = function (route, integration, args) {
        var _a, _b;
        if (args === void 0) { args = {}; }
        var _c = this.parseRoute(route), method = _c.method, path = _c.path;
        this.createResource(path);
        var transformed = (0, component_1.transform)((_b = (_a = this.constructorArgs.transform) === null || _a === void 0 ? void 0 : _a.route) === null || _b === void 0 ? void 0 : _b.args, this.buildRouteId(method, path), args, { provider: this.constructorOpts.provider });
        var apigRoute = new apigatewayv1_integration_route_1.ApiGatewayV1IntegrationRoute(transformed[0], __assign({ api: {
                name: this.constructorName,
                id: this.api.id,
                executionArn: this.api.executionArn,
            }, method: method, path: path, resourceId: this.resources[path], integration: integration }, transformed[1]), transformed[2]);
        this.routes.push(apigRoute);
        return apigRoute;
    };
    ApiGatewayV1.prototype.parseRoute = function (route) {
        var parts = route.split(" ");
        if (parts.length !== 2) {
            throw new error_1.VisibleError("Invalid route ".concat(route, ". A route must be in the format \"METHOD /path\"."));
        }
        var _a = route.split(" "), methodRaw = _a[0], path = _a[1];
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
            throw new error_1.VisibleError("Invalid method ".concat(methodRaw, " in route ").concat(route));
        if (!path.startsWith("/"))
            throw new error_1.VisibleError("Invalid path ".concat(path, " in route ").concat(route, ". Path must start with \"/\"."));
        return { method: method, path: path };
    };
    ApiGatewayV1.prototype.buildRouteId = function (method, path) {
        var suffix = (0, naming_1.logicalName)((0, naming_1.hashStringToPrettyString)([component_1.outputId, method, path].join(""), 6));
        return "".concat(this.constructorName, "Route").concat(suffix);
    };
    ApiGatewayV1.prototype.createResource = function (path) {
        var pathParts = path.replace(/^\//, "").split("/");
        for (var i = 0, l = pathParts.length; i < l; i++) {
            var parentPath = "/" + pathParts.slice(0, i).join("/");
            var subPath = "/" + pathParts.slice(0, i + 1).join("/");
            if (!this.resources[subPath]) {
                var suffix = (0, naming_1.logicalName)((0, naming_1.hashStringToPrettyString)([this.api.id, subPath].join(""), 6));
                var resource = new aws_1.apigateway.Resource("".concat(this.constructorName, "Resource").concat(suffix), {
                    restApi: this.api.id,
                    parentId: parentPath === "/"
                        ? this.api.rootResourceId
                        : this.resources[parentPath],
                    pathPart: pathParts[i],
                }, { parent: this });
                this.resources[subPath] = resource.id;
            }
        }
    };
    /**
     * Add an authorizer to the API Gateway REST API.
     *
     * @param args Configure the authorizer.
     * @example
     * Add a Lambda token authorizer.
     *
     * ```js title="sst.config.ts"
     * api.addAuthorizer({
     *   name: "myAuthorizer",
     *   tokenFunction: "src/authorizer.index"
     * });
     * ```
     *
     * Add a Lambda REQUEST authorizer.
     *
     * ```js title="sst.config.ts"
     * api.addAuthorizer({
     *   name: "myAuthorizer",
     *   requestFunction: "src/authorizer.index"
     * });
     * ```
     *
     * Add a Cognito User Pool authorizer.
     *
     * ```js title="sst.config.ts"
     * const userPool = new aws.cognito.UserPool();
     *
     * api.addAuthorizer({
     *   name: "myAuthorizer",
     *   userPools: [userPool.arn]
     * });
     * ```
     *
     * Customize the authorizer.
     *
     * ```js title="sst.config.ts"
     * api.addAuthorizer({
     *   name: "myAuthorizer",
     *   tokenFunction: "src/authorizer.index",
     *   ttl: 30
     * });
     * ```
     */
    ApiGatewayV1.prototype.addAuthorizer = function (args) {
        var self = this;
        var selfName = this.constructorName;
        var nameSuffix = (0, naming_1.logicalName)(args.name);
        return new apigatewayv1_authorizer_1.ApiGatewayV1Authorizer("".concat(selfName, "Authorizer").concat(nameSuffix), __assign({ api: {
                id: self.api.id,
                name: selfName,
                executionArn: self.api.executionArn,
            } }, args), { provider: this.constructorOpts.provider });
    };
    /**
     * Add a usage plan to the API Gateway REST API.
     *
     * @param name The name of the usage plan.
     * @param args Configure the usage plan.
     * @example
     * Add a usage plan with throttle and quota.
     *
     * ```js title="sst.config.ts"
     * const plan = api.addUsagePlan("MyPlan", {
     *   throttle: {
     *     rate: 100,
     *     burst: 200,
     *   },
     *   quota: {
     *     limit: 1000,
     *     period: "month",
     *     offset: 0,
     *   }
     * });
     * ```
     *
     * Create an API key for the usage plan.
     *
     * ```js title="sst.config.ts"
     * const key = plan.addApiKey("MyKey");
     * ```
     *
     * You can link the API key to other resources, like a function. Once linked,
     * include the key in the `x-api-key` header in your API requests.
     *
     * ```ts title="src/lambda.ts"
     * import { Resource } from "sst";
     *
     * await fetch(Resource.MyApi.url, {
     *   headers: {
     *     "x-api-key": Resource.MyKey.value,
     *   }
     * });
     * ```
     */
    ApiGatewayV1.prototype.addUsagePlan = function (name, args) {
        if (!this.stage)
            throw new error_1.VisibleError("Cannot add a usage plan to the \"".concat(this.constructorName, "\" API before it's deployed. Make sure to call deploy() to deploy the API first."));
        return new apigatewayv1_usage_plan_1.ApiGatewayV1UsagePlan(name, __assign({ apiId: this.api.id, apiStage: this.stage.stageName }, args), { provider: this.constructorOpts.provider });
    };
    /**
     * Create a deployment for the API Gateway REST API.
     *
     * :::note
     * You need to call this after you've added all your routes.
     * :::
     *
     * Due to the way API Gateway V1 is created internally, you'll need to call this method after
     * you've added all your routes.
     */
    ApiGatewayV1.prototype.deploy = function () {
        var name = this.constructorName;
        var args = this.constructorArgs;
        var parent = this;
        var api = this.api;
        var routes = this.routes;
        var region = this.region;
        var endpointType = this.endpointType;
        var accessLog = normalizeAccessLog();
        var domain = normalizeDomain();
        var corsRoutes = createCorsRoutes();
        var corsResponses = createCorsResponses();
        var deployment = createDeployment();
        var logGroup = createLogGroup();
        var stage = createStage();
        var certificateArn = createSsl();
        var apigDomain = createDomainName();
        createDnsRecords();
        var apiMapping = createDomainMapping();
        this.deployed = true;
        this.logGroup = logGroup;
        this.stage = stage;
        this.apigDomain = apigDomain;
        this.apiMapping = apiMapping;
        this.registerOutputs({
            _hint: this.url,
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
        function createCorsRoutes() {
            var resourceIds = routes.map(function (route) { return route.nodes.integration.resourceId; });
            return (0, pulumi_1.all)([args.cors, resourceIds]).apply(function (_a) {
                var cors = _a[0], resourceIds = _a[1];
                if (cors === false)
                    return [];
                // filter unique resource ids
                var uniqueResourceIds = __spreadArray([], new Set(resourceIds), true);
                // create cors integrations for the paths
                return uniqueResourceIds.map(function (resourceId) {
                    var method = new aws_1.apigateway.Method("".concat(name, "CorsMethod").concat(resourceId), {
                        restApi: api.id,
                        resourceId: resourceId,
                        httpMethod: "OPTIONS",
                        authorization: "NONE",
                    }, { parent: parent });
                    var methodResponse = new aws_1.apigateway.MethodResponse("".concat(name, "CorsMethodResponse").concat(resourceId), {
                        restApi: api.id,
                        resourceId: resourceId,
                        httpMethod: method.httpMethod,
                        statusCode: "204",
                        responseParameters: {
                            "method.response.header.Access-Control-Allow-Headers": true,
                            "method.response.header.Access-Control-Allow-Methods": true,
                            "method.response.header.Access-Control-Allow-Origin": true,
                        },
                    }, { parent: parent });
                    var integration = new aws_1.apigateway.Integration("".concat(name, "CorsIntegration").concat(resourceId), {
                        restApi: api.id,
                        resourceId: resourceId,
                        httpMethod: method.httpMethod,
                        type: "MOCK",
                        requestTemplates: {
                            "application/json": "{ statusCode: 200 }",
                        },
                    }, { parent: parent });
                    var integrationResponse = new aws_1.apigateway.IntegrationResponse("".concat(name, "CorsIntegrationResponse").concat(resourceId), {
                        restApi: api.id,
                        resourceId: resourceId,
                        httpMethod: method.httpMethod,
                        statusCode: methodResponse.statusCode,
                        responseParameters: {
                            "method.response.header.Access-Control-Allow-Headers": "'*'",
                            "method.response.header.Access-Control-Allow-Methods": "'OPTIONS,GET,PUT,POST,DELETE,PATCH,HEAD'",
                            "method.response.header.Access-Control-Allow-Origin": "'*'",
                        },
                    }, { parent: parent, dependsOn: [integration] });
                    return { method: method, methodResponse: methodResponse, integration: integration, integrationResponse: integrationResponse };
                });
            });
        }
        function createCorsResponses() {
            return (0, pulumi_1.output)(args.cors).apply(function (cors) {
                if (cors === false)
                    return [];
                return ["4XX", "5XX"].map(function (type) {
                    return new aws_1.apigateway.Response("".concat(name, "Cors").concat(type, "Response"), {
                        restApiId: api.id,
                        responseType: "DEFAULT_".concat(type),
                        responseParameters: {
                            "gatewayresponse.header.Access-Control-Allow-Origin": "'*'",
                            "gatewayresponse.header.Access-Control-Allow-Headers": "'*'",
                        },
                        responseTemplates: {
                            "application/json": '{"message":$context.error.messageString}',
                        },
                    }, { parent: parent });
                });
            });
        }
        function createDeployment() {
            var _a;
            var _b;
            var resources = (0, pulumi_1.all)([corsRoutes, corsResponses]).apply(function (_a) {
                var corsRoutes = _a[0], corsResponses = _a[1];
                return [
                    api,
                    corsRoutes.map(function (v) { return Object.values(v); }),
                    corsResponses,
                    routes.map(function (route) { return [
                        route.nodes.integration,
                        route.nodes.method,
                    ]; }),
                ].flat(3);
            });
            // filter serializable output values
            var resourcesSanitized = (0, pulumi_1.all)([resources]).apply(function (_a) {
                var resources = _a[0];
                return resources.map(function (resource) {
                    return Object.fromEntries(Object.entries(resource).filter(function (_a) {
                        var k = _a[0], v = _a[1];
                        return !k.startsWith("_") && typeof v !== "function";
                    }));
                });
            });
            return new ((_a = aws_1.apigateway.Deployment).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.deployment, "".concat(name, "Deployment"), {
                restApi: api.id,
                triggers: (0, pulumi_1.all)([resourcesSanitized]).apply(function (_a) {
                    var resources = _a[0];
                    return Object.fromEntries(resources.map(function (resource) { return [
                        resource.urn,
                        JSON.stringify(resource),
                    ]; }));
                }),
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
            return new ((_a = aws_1.apigateway.Stage).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.stage, "".concat(name, "Stage"), {
                restApi: api.id,
                stageName: $app.stage,
                deployment: deployment.id,
                accessLogSettings: {
                    destinationArn: logGroup.arn,
                    format: JSON.stringify({
                        // request info
                        requestTime: "\"$context.requestTime\"",
                        requestId: "\"$context.requestId\"",
                        httpMethod: "\"$context.httpMethod\"",
                        path: "\"$context.path\"",
                        resourcePath: "\"$context.resourcePath\"",
                        status: "$context.status", // integer value, do not wrap in quotes
                        responseLatency: "$context.responseLatency", // integer value, do not wrap in quotes
                        xrayTraceId: "\"$context.xrayTraceId\"",
                        // integration info
                        functionResponseStatus: "\"$context.integration.status\"",
                        integrationRequestId: "\"$context.integration.requestId\"",
                        integrationLatency: "\"$context.integration.latency\"",
                        integrationServiceStatus: "\"$context.integration.integrationStatus\"",
                        // caller info
                        ip: "\"$context.identity.sourceIp\"",
                        userAgent: "\"$context.identity.userAgent\"",
                        principalId: "\"$context.authorizer.principalId\"",
                    }),
                },
            }, { parent: parent }), false)))();
        }
        function createSsl() {
            if (!domain)
                return;
            return (0, pulumi_1.all)([domain, endpointType, region]).apply(function (_a) {
                var domain = _a[0], endpointType = _a[1], region = _a[2];
                if (domain.cert)
                    return (0, pulumi_1.output)(domain.cert);
                if (domain.nameId)
                    return (0, pulumi_1.output)(undefined);
                return new dns_validated_certificate_1.DnsValidatedCertificate("".concat(name, "Ssl"), {
                    domainName: domain.name,
                    dns: domain.dns,
                }, {
                    parent: parent,
                    provider: endpointType === "EDGE" && region !== "us-east-1"
                        ? (0, provider_1.useProvider)("us-east-1")
                        : undefined,
                }).arn;
            });
        }
        function createDomainName() {
            if (!domain || !certificateArn)
                return;
            return (0, pulumi_1.all)([domain, endpointType]).apply(function (_a) {
                var _b;
                var _c;
                var domain = _a[0], endpointType = _a[1];
                return domain.nameId
                    ? aws_1.apigateway.DomainName.get("".concat(name, "DomainName"), domain.nameId, {}, { parent: parent })
                    : new ((_b = aws_1.apigateway.DomainName).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.domainName, "".concat(name, "DomainName"), __assign({ domainName: domain === null || domain === void 0 ? void 0 : domain.name, endpointConfiguration: { types: endpointType } }, (endpointType === "REGIONAL"
                        ? {
                            regionalCertificateArn: certificateArn,
                        }
                        : { certificateArn: certificateArn })), { parent: parent }), false)))();
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
                    aliasName: endpointType.apply(function (v) {
                        return v === "EDGE"
                            ? apigDomain.cloudfrontDomainName
                            : apigDomain.regionalDomainName;
                    }),
                    aliasZone: endpointType.apply(function (v) {
                        return v === "EDGE"
                            ? apigDomain.cloudfrontZoneId
                            : apigDomain.regionalZoneId;
                    }),
                }, { parent: parent });
            });
        }
        function createDomainMapping() {
            var _a;
            if (!domain || !apigDomain)
                return;
            return (_a = domain.path) === null || _a === void 0 ? void 0 : _a.apply(function (path) {
                return new aws_1.apigateway.BasePathMapping("".concat(name, "DomainMapping"), {
                    restApi: api.id,
                    domainName: apigDomain.id,
                    stageName: stage.stageName,
                    basePath: path,
                }, { parent: parent });
            });
        }
    };
    /** @internal */
    ApiGatewayV1.prototype.getSSTLink = function () {
        return {
            properties: {
                url: this.url,
            },
        };
    };
    return ApiGatewayV1;
}(component_1.Component));
exports.ApiGatewayV1 = ApiGatewayV1;
var __pulumiType = "sst:aws:ApiGatewayV1";
// @ts-expect-error
ApiGatewayV1.__pulumiType = __pulumiType;
var templateObject_1;
