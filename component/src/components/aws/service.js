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
exports.Service = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_js_1 = require("../component.js");
var dns_js_1 = require("./dns.js");
var error_js_1 = require("../error.js");
var dns_validated_certificate_js_1 = require("./dns-validated-certificate.js");
var linkable_js_1 = require("./linkable.js");
var aws_1 = require("@pulumi/aws");
var vpc_js_1 = require("./vpc.js");
var dev_command_js_1 = require("../experimental/dev-command.js");
var duration_js_1 = require("../duration.js");
var fargate_js_1 = require("./fargate.js");
var naming_js_1 = require("../naming.js");
/**
 * The `Service` component lets you create containers that are always running, like web or
 * application servers. It uses [Amazon ECS](https://aws.amazon.com/ecs/) on [AWS Fargate](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/AWS_Fargate.html).
 *
 * @example
 *
 * #### Create a Service
 *
 * Services are run inside an ECS Cluster. If you haven't already, create one.
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const cluster = new sst.aws.Cluster("MyCluster", { vpc });
 * ```
 *
 * Add the service to it.
 *
 * ```ts title="sst.config.ts"
 * const service = new sst.aws.Service("MyService", { cluster });
 * ```
 *
 * #### Configure the container image
 *
 * By default, the service will look for a Dockerfile in the root directory. Optionally
 * configure the image context and dockerfile.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   image: {
 *     context: "./app",
 *     dockerfile: "Dockerfile"
 *   }
 * });
 * ```
 *
 * To add multiple containers in the service, pass in an array of containers args.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   containers: [
 *     {
 *       name: "app",
 *       image: "nginxdemos/hello:plain-text"
 *     },
 *     {
 *       name: "admin",
 *       image: {
 *         context: "./admin",
 *         dockerfile: "Dockerfile"
 *       }
 *     }
 *   ]
 * });
 * ```
 *
 * This is useful for running sidecar containers.
 *
 * #### Enable auto-scaling
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   scaling: {
 *     min: 4,
 *     max: 16,
 *     cpuUtilization: 50,
 *     memoryUtilization: 50
 *   }
 * });
 * ```
 *
 * #### Expose through API Gateway
 *
 * You can give your service a public URL by exposing it through API Gateway HTTP API. You can
 * also optionally give it a custom domain.
 *
 * ```ts title="sst.config.ts"
 * const service = new sst.aws.Service("MyService", {
 *   cluster,
 *   serviceRegistry: {
 *     port: 80
 *   }
 * });
 *
 * const api = new sst.aws.ApiGatewayV2("MyApi", {
 *   vpc,
 *   domain: "example.com"
 * });
 * api.routePrivate("$default", service.nodes.cloudmapService.arn);
 * ```
 *
 * #### Add a load balancer
 *
 * You can also expose your service by adding a load balancer to it and optionally
 * adding a custom domain.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   loadBalancer: {
 *     domain: "example.com",
 *     rules: [
 *       { listen: "80/http" },
 *       { listen: "443/https", forward: "80/http" }
 *     ]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your service. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {5} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources in your service.
 *
 * ```ts title="app.ts"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 *
 * #### Service discovery
 *
 * This component automatically creates a Cloud Map service host name for the
 * service. So anything in the same VPC can access it using the service's host name.
 *
 * For example, if you link the service to a Lambda function that's in the same VPC.
 *
 * ```ts title="sst.config.ts" {2,4}
 * new sst.aws.Function("MyFunction", {
 *   vpc,
 *   url: true,
 *   link: [service],
 *   handler: "lambda.handler"
 * });
 * ```
 *
 * You can access the service by its host name using the [SDK](/docs/reference/sdk/).
 *
 * ```ts title="lambda.ts"
 * import { Resource } from "sst";
 *
 * await fetch(`http://${Resource.MyService.service}`);
 * ```
 *
 * [Check out an example](/docs/examples/#aws-cluster-service-discovery).
 *
 * ---
 *
 * ### Cost
 *
 * By default, this uses a _Linux/X86_ _Fargate_ container with 0.25 vCPUs at $0.04048 per
 * vCPU per hour and 0.5 GB of memory at $0.004445 per GB per hour. It includes 20GB of
 * _Ephemeral Storage_ for free with additional storage at $0.000111 per GB per hour. Each
 * container also gets a public IPv4 address at $0.005 per hour.
 *
 * It works out to $0.04048 x 0.25 x 24 x 30 + $0.004445 x 0.5 x 24 x 30 + $0.005
 * x 24 x 30 or **$12 per month**.
 *
 * If you are using all Fargate Spot instances with `capacity: "spot"`, it's $0.01218784 x 0.25
 * x 24 x 30 + $0.00133831 x 0.5 x 24 x 30 + $0.005 x 24 x 30 or **$6 per month**
 *
 * Adjust this for the `cpu`, `memory` and `storage` you are using. And
 * check the prices for _Linux/ARM_ if you are using `arm64` as your `architecture`.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [Fargate pricing](https://aws.amazon.com/fargate/pricing/) and the
 * [Public IPv4 Address pricing](https://aws.amazon.com/vpc/pricing/) for more details.
 *
 * #### Scaling
 *
 * By default, `scaling` is disabled. If enabled, adjust the above for the number of containers.
 *
 * #### API Gateway
 *
 * If you expose your service through API Gateway, you'll need to add the cost of
 * [API Gateway HTTP API](https://aws.amazon.com/api-gateway/pricing/#HTTP_APIs) as well.
 * For services that don't get a lot of traffic, this ends up being a lot cheaper since API
 * Gateway is pay per request.
 *
 * Learn more about using
 * [Cluster with API Gateway](/docs/examples/#aws-cluster-with-api-gateway).
 *
 * #### Application Load Balancer
 *
 * If you add `loadBalancer` _HTTP_ or _HTTPS_ `rules`, an ALB is created at $0.0225 per hour,
 * $0.008 per LCU-hour, and $0.005 per hour if HTTPS with a custom domain is used. Where LCU
 * is a measure of how much traffic is processed.
 *
 * That works out to $0.0225 x 24 x 30 or **$16 per month**. Add $0.005 x 24 x 30 or **$4 per
 * month** for HTTPS. Also add the LCU-hour used.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [Application Load Balancer pricing](https://aws.amazon.com/elasticloadbalancing/pricing/)
 * for more details.
 *
 * #### Network Load Balancer
 *
 * If you add `loadBalancer` _TCP_, _UDP_, or _TLS_ `rules`, an NLB is created at $0.0225 per hour and
 * $0.006 per NLCU-hour. Where NCLU is a measure of how much traffic is processed.
 *
 * That works out to $0.0225 x 24 x 30 or **$16 per month**. Also add the NLCU-hour used.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [Network Load Balancer pricing](https://aws.amazon.com/elasticloadbalancing/pricing/)
 * for more details.
 */
var Service = /** @class */ (function (_super) {
    __extends(Service, _super);
    function Service(name, args, opts) {
        if (opts === void 0) { opts = {}; }
        var _a, _b;
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        _this._name = name;
        var self = _this;
        var clusterArn = args.cluster.nodes.cluster.arn;
        var clusterName = args.cluster.nodes.cluster.name;
        var region = (0, aws_1.getRegionOutput)({}, opts).name;
        var dev = normalizeDev();
        var wait = (0, pulumi_1.output)((_a = args.wait) !== null && _a !== void 0 ? _a : false);
        var architecture = (0, fargate_js_1.normalizeArchitecture)(args);
        var cpu = (0, fargate_js_1.normalizeCpu)(args);
        var memory = (0, fargate_js_1.normalizeMemory)(cpu, args);
        var storage = (0, fargate_js_1.normalizeStorage)(args);
        var containers = (0, fargate_js_1.normalizeContainers)("service", args, name, architecture);
        var lbArgs = normalizeLoadBalancer();
        var scaling = normalizeScaling();
        var capacity = normalizeCapacity();
        var vpc = normalizeVpc();
        var taskRole = (0, fargate_js_1.createTaskRole)(name, args, opts, self, !!dev);
        _this.dev = !!dev;
        _this.cloudmapNamespace = vpc.cloudmapNamespaceName;
        _this.taskRole = taskRole;
        if (dev) {
            _this.devUrl = !lbArgs ? undefined : dev.url;
            registerReceiver();
            return _this;
        }
        var executionRole = (0, fargate_js_1.createExecutionRole)(name, args, opts, self);
        var taskDefinition = (0, fargate_js_1.createTaskDefinition)(name, args, opts, self, containers, architecture, cpu, memory, storage, taskRole, executionRole);
        var certificateArn = createSsl();
        var loadBalancer = createLoadBalancer();
        var targetGroups = createTargets();
        createListeners();
        var cloudmapService = createCloudmapService();
        var service = createService();
        var autoScalingTarget = createAutoScaling();
        createDnsRecords();
        _this._service = service;
        _this.cloudmapService = cloudmapService;
        _this.executionRole = executionRole;
        _this.taskDefinition = taskDefinition;
        _this.loadBalancer = loadBalancer;
        _this.autoScalingTarget = autoScalingTarget;
        _this.domain = (lbArgs === null || lbArgs === void 0 ? void 0 : lbArgs.domain)
            ? lbArgs.domain.apply(function (domain) { return domain === null || domain === void 0 ? void 0 : domain.name; })
            : (0, pulumi_1.output)(undefined);
        _this._url = !self.loadBalancer
            ? undefined
            : (0, pulumi_1.all)([self.domain, (_b = self.loadBalancer) === null || _b === void 0 ? void 0 : _b.dnsName]).apply(function (_a) {
                var domain = _a[0], loadBalancer = _a[1];
                return domain ? "https://".concat(domain, "/") : "http://".concat(loadBalancer);
            });
        _this.registerOutputs({ _hint: _this._url });
        registerReceiver();
        function normalizeDev() {
            var _a, _b;
            if (!$dev)
                return undefined;
            if (args.dev === false)
                return undefined;
            return {
                url: (0, pulumi_1.output)((_b = (_a = args.dev) === null || _a === void 0 ? void 0 : _a.url) !== null && _b !== void 0 ? _b : linkable_js_1.URL_UNAVAILABLE),
            };
        }
        function normalizeVpc() {
            // "vpc" is a Vpc component
            if (args.cluster.vpc instanceof vpc_js_1.Vpc) {
                var vpc_1 = args.cluster.vpc;
                return {
                    isSstVpc: true,
                    id: vpc_1.id,
                    loadBalancerSubnets: lbArgs === null || lbArgs === void 0 ? void 0 : lbArgs.pub.apply(function (v) {
                        return v ? vpc_1.publicSubnets : vpc_1.privateSubnets;
                    }),
                    containerSubnets: vpc_1.publicSubnets,
                    securityGroups: vpc_1.securityGroups,
                    cloudmapNamespaceId: vpc_1.nodes.cloudmapNamespace.id,
                    cloudmapNamespaceName: vpc_1.nodes.cloudmapNamespace.name,
                };
            }
            // "vpc" is object
            return (0, pulumi_1.output)(args.cluster.vpc).apply(function (vpc) { return (__assign({ isSstVpc: false }, vpc)); });
        }
        function normalizeScaling() {
            return (0, pulumi_1.all)([lbArgs === null || lbArgs === void 0 ? void 0 : lbArgs.type, args.scaling]).apply(function (_a) {
                var _b, _c, _d, _e, _f;
                var type = _a[0], v = _a[1];
                if (type !== "application" && (v === null || v === void 0 ? void 0 : v.requestCount))
                    throw new error_js_1.VisibleError("Request count scaling is only supported for http/https protocols.");
                return {
                    min: (_b = v === null || v === void 0 ? void 0 : v.min) !== null && _b !== void 0 ? _b : 1,
                    max: (_c = v === null || v === void 0 ? void 0 : v.max) !== null && _c !== void 0 ? _c : 1,
                    cpuUtilization: (_d = v === null || v === void 0 ? void 0 : v.cpuUtilization) !== null && _d !== void 0 ? _d : 70,
                    memoryUtilization: (_e = v === null || v === void 0 ? void 0 : v.memoryUtilization) !== null && _e !== void 0 ? _e : 70,
                    requestCount: (_f = v === null || v === void 0 ? void 0 : v.requestCount) !== null && _f !== void 0 ? _f : false,
                };
            });
        }
        function normalizeCapacity() {
            if (!args.capacity)
                return;
            return (0, pulumi_1.output)(args.capacity).apply(function (v) {
                if (v === "spot")
                    return { spot: { weight: 1 }, fargate: { weight: 0 } };
                return v;
            });
        }
        function normalizeLoadBalancer() {
            var _a;
            var loadBalancer = ((_a = args.loadBalancer) !== null && _a !== void 0 ? _a : args.public);
            if (!loadBalancer)
                return;
            // normalize rules
            var rules = (0, pulumi_1.all)([loadBalancer, containers]).apply(function (_a) {
                var _b;
                var lb = _a[0], containers = _a[1];
                // validate rules
                var lbRules = (_b = lb.rules) !== null && _b !== void 0 ? _b : lb.ports;
                if (!lbRules || lbRules.length === 0)
                    throw new error_js_1.VisibleError("You must provide the ports to expose via \"loadBalancer.rules\".");
                // validate container defined when multiple containers exists
                if (containers.length > 1) {
                    lbRules.forEach(function (v) {
                        if (!v.container)
                            throw new error_js_1.VisibleError("You must provide a container name in \"loadBalancer.rules\" when there is more than one container.");
                    });
                }
                // parse protocols and ports
                var rules = lbRules.map(function (v) {
                    var _a, _b, _c, _d, _e, _f;
                    var listenParts = v.listen.split("/");
                    var listenPort = parseInt(listenParts[0]);
                    var listenProtocol = listenParts[1];
                    var listenConditions = v.conditions || v.path
                        ? {
                            path: (_b = (_a = v.conditions) === null || _a === void 0 ? void 0 : _a.path) !== null && _b !== void 0 ? _b : v.path,
                            query: (_c = v.conditions) === null || _c === void 0 ? void 0 : _c.query,
                            header: (_d = v.conditions) === null || _d === void 0 ? void 0 : _d.header,
                        }
                        : undefined;
                    if (protocolType(listenProtocol) === "network" && listenConditions)
                        throw new error_js_1.VisibleError("Invalid rule conditions for listen protocol \"".concat(v.listen, "\". Only \"http\" protocols support conditions."));
                    var redirectParts = (_e = v.redirect) === null || _e === void 0 ? void 0 : _e.split("/");
                    var redirectPort = redirectParts && parseInt(redirectParts[0]);
                    var redirectProtocol = redirectParts && redirectParts[1];
                    if (redirectPort && redirectProtocol) {
                        if (protocolType(listenProtocol) !== protocolType(redirectProtocol))
                            throw new error_js_1.VisibleError("The listen protocol \"".concat(v.listen, "\" must match the redirect protocol \"").concat(v.redirect, "\"."));
                        return {
                            type: "redirect",
                            listenPort: listenPort,
                            listenProtocol: listenProtocol,
                            listenConditions: listenConditions,
                            redirectPort: redirectPort,
                            redirectProtocol: redirectProtocol,
                        };
                    }
                    var forwardParts = v.forward ? v.forward.split("/") : listenParts;
                    var forwardPort = forwardParts && parseInt(forwardParts[0]);
                    var forwardProtocol = forwardParts && forwardParts[1];
                    if (protocolType(listenProtocol) !== protocolType(forwardProtocol))
                        throw new error_js_1.VisibleError("The listen protocol \"".concat(v.listen, "\" must match the forward protocol \"").concat(v.forward, "\"."));
                    return {
                        type: "forward",
                        listenPort: listenPort,
                        listenProtocol: listenProtocol,
                        listenConditions: listenConditions,
                        forwardPort: forwardPort,
                        forwardProtocol: forwardProtocol,
                        container: (_f = v.container) !== null && _f !== void 0 ? _f : containers[0].name,
                    };
                });
                // validate protocols are consistent
                var appProtocols = rules.filter(function (rule) { return protocolType(rule.listenProtocol) === "application"; });
                if (appProtocols.length > 0 && appProtocols.length < rules.length)
                    throw new error_js_1.VisibleError("Protocols must be either all http/https, or all tcp/udp/tcp_udp/tls.");
                // validate certificate exists for https/tls protocol
                rules.forEach(function (rule) {
                    if (["https", "tls"].includes(rule.listenProtocol) && !lb.domain) {
                        throw new error_js_1.VisibleError("You must provide a custom domain for ".concat(rule.listenProtocol.toUpperCase(), " protocol."));
                    }
                });
                return rules;
            });
            // normalize domain
            var domain = (0, pulumi_1.output)(loadBalancer).apply(function (lb) {
                var _a, _b;
                if (!lb.domain)
                    return undefined;
                // normalize domain
                var domain = typeof lb.domain === "string" ? { name: lb.domain } : lb.domain;
                return {
                    name: domain.name,
                    aliases: (_a = domain.aliases) !== null && _a !== void 0 ? _a : [],
                    dns: domain.dns === false ? undefined : (_b = domain.dns) !== null && _b !== void 0 ? _b : (0, dns_js_1.dns)(),
                    cert: domain.cert,
                };
            });
            // normalize type
            var type = (0, pulumi_1.output)(rules).apply(function (rules) {
                return rules[0].listenProtocol.startsWith("http") ? "application" : "network";
            });
            // normalize public/private
            var pub = (0, pulumi_1.output)(loadBalancer).apply(function (lb) { var _a; return (_a = lb === null || lb === void 0 ? void 0 : lb.public) !== null && _a !== void 0 ? _a : true; });
            // normalize health check
            var health = (0, pulumi_1.all)([type, rules, loadBalancer]).apply(function (_a) {
                var _b;
                var type = _a[0], rules = _a[1], lb = _a[2];
                return Object.fromEntries(Object.entries((_b = lb === null || lb === void 0 ? void 0 : lb.health) !== null && _b !== void 0 ? _b : {}).map(function (_a) {
                    var _b, _c, _d, _e;
                    var k = _a[0], v = _a[1];
                    if (!rules.find(function (r) { return "".concat(r.forwardPort, "/").concat(r.forwardProtocol) === k; }))
                        throw new error_js_1.VisibleError("Cannot configure health check for \"".concat(k, "\". Make sure it is defined in \"loadBalancer.ports\"."));
                    return [
                        k,
                        {
                            path: (_b = v.path) !== null && _b !== void 0 ? _b : "/",
                            interval: v.interval ? (0, duration_js_1.toSeconds)(v.interval) : 30,
                            timeout: v.timeout
                                ? (0, duration_js_1.toSeconds)(v.timeout)
                                : type === "application"
                                    ? 5
                                    : 6,
                            healthyThreshold: (_c = v.healthyThreshold) !== null && _c !== void 0 ? _c : 5,
                            unhealthyThreshold: (_d = v.unhealthyThreshold) !== null && _d !== void 0 ? _d : 2,
                            matcher: (_e = v.successCodes) !== null && _e !== void 0 ? _e : "200",
                        },
                    ];
                }));
            });
            return { type: type, rules: rules, domain: domain, pub: pub, health: health };
        }
        function createLoadBalancer() {
            var _a, _b;
            var _c, _d;
            if (!lbArgs)
                return;
            var securityGroup = new ((_a = aws_1.ec2.SecurityGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_c = args === null || args === void 0 ? void 0 : args.transform) === null || _c === void 0 ? void 0 : _c.loadBalancerSecurityGroup, "".concat(name, "LoadBalancerSecurityGroup"), {
                description: "Managed by SST",
                vpcId: vpc.id,
                egress: [
                    {
                        fromPort: 0,
                        toPort: 0,
                        protocol: "-1",
                        cidrBlocks: ["0.0.0.0/0"],
                    },
                ],
                ingress: [
                    {
                        fromPort: 0,
                        toPort: 0,
                        protocol: "-1",
                        cidrBlocks: ["0.0.0.0/0"],
                    },
                ],
            }, { parent: self }), false)))();
            return new ((_b = aws_1.lb.LoadBalancer).bind.apply(_b, __spreadArray([void 0], (0, component_js_1.transform)((_d = args.transform) === null || _d === void 0 ? void 0 : _d.loadBalancer, "".concat(name, "LoadBalancer"), {
                internal: lbArgs.pub.apply(function (v) { return !v; }),
                loadBalancerType: lbArgs.type,
                subnets: vpc.loadBalancerSubnets,
                securityGroups: [securityGroup.id],
                enableCrossZoneLoadBalancing: true,
            }, { parent: self }), false)))();
        }
        function createTargets() {
            if (!loadBalancer || !lbArgs)
                return;
            return (0, pulumi_1.all)([lbArgs.rules, lbArgs.health]).apply(function (_a) {
                var rules = _a[0], health = _a[1];
                var targets = {};
                rules.forEach(function (r) {
                    var _a;
                    var _b, _c;
                    if (r.type !== "forward")
                        return;
                    var container = r.container;
                    var forwardProtocol = r.forwardProtocol.toUpperCase();
                    var forwardPort = r.forwardPort;
                    var targetId = "".concat(container).concat(forwardProtocol).concat(forwardPort);
                    var target = (_b = targets[targetId]) !== null && _b !== void 0 ? _b : new ((_a = aws_1.lb.TargetGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.target, "".concat(name, "Target").concat(targetId), {
                        // TargetGroup names allow for 32 chars, but an 8 letter suffix
                        // ie. "-1234567" is automatically added.
                        // - If we don't specify "name" or "namePrefix", we need to ensure
                        //   the component name is less than 24 chars. Hard to guarantee.
                        // - If we specify "name", we need to ensure the $app-$stage-$name
                        //   if less than 32 chars. Hard to guarantee.
                        // - Hence we will use "namePrefix".
                        namePrefix: forwardProtocol,
                        port: forwardPort,
                        protocol: forwardProtocol,
                        targetType: "ip",
                        vpcId: vpc.id,
                        healthCheck: health["".concat(r.forwardPort, "/").concat(r.forwardProtocol)],
                    }, { parent: self }), false)))();
                    targets[targetId] = target;
                });
                return targets;
            });
        }
        function createListeners() {
            if (!lbArgs || !loadBalancer || !targetGroups)
                return;
            return (0, pulumi_1.all)([lbArgs.rules, targetGroups, certificateArn]).apply(function (_a) {
                var rules = _a[0], targets = _a[1], cert = _a[2];
                // Group listeners by protocol and port
                // Because listeners with the same protocol and port but different path
                // are just rules of the same listener.
                var listenersById = {};
                rules.forEach(function (r) {
                    var _a;
                    var listenProtocol = r.listenProtocol.toUpperCase();
                    var listenPort = r.listenPort;
                    var listenerId = "".concat(listenProtocol).concat(listenPort);
                    listenersById[listenerId] = (_a = listenersById[listenerId]) !== null && _a !== void 0 ? _a : [];
                    listenersById[listenerId].push(r);
                });
                // Create listeners
                return Object.entries(listenersById).map(function (_a) {
                    var _b;
                    var _c;
                    var listenerId = _a[0], rules = _a[1];
                    var listenProtocol = rules[0].listenProtocol.toUpperCase();
                    var listenPort = rules[0].listenPort;
                    var defaultRule = rules.find(function (r) { return !r.listenConditions; });
                    var customRules = rules.filter(function (r) { return r.listenConditions; });
                    var buildActions = function (r) { return __spreadArray(__spreadArray(__spreadArray([], (!r
                        ? [
                            {
                                type: "fixed-response",
                                fixedResponse: {
                                    statusCode: "403",
                                    contentType: "text/plain",
                                    messageBody: "Forbidden",
                                },
                            },
                        ]
                        : []), true), ((r === null || r === void 0 ? void 0 : r.type) === "forward"
                        ? [
                            {
                                type: "forward",
                                targetGroupArn: targets["".concat(r.container).concat(r.forwardProtocol.toUpperCase()).concat(r.forwardPort)].arn,
                            },
                        ]
                        : []), true), ((r === null || r === void 0 ? void 0 : r.type) === "redirect"
                        ? [
                            {
                                type: "redirect",
                                redirect: {
                                    port: r.redirectPort.toString(),
                                    protocol: r.redirectProtocol.toUpperCase(),
                                    statusCode: "HTTP_301",
                                },
                            },
                        ]
                        : []), true); };
                    var listener = new ((_b = aws_1.lb.Listener).bind.apply(_b, __spreadArray([void 0], (0, component_js_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.listener, "".concat(name, "Listener").concat(listenerId), {
                        loadBalancerArn: loadBalancer.arn,
                        port: listenPort,
                        protocol: listenProtocol,
                        certificateArn: ["HTTPS", "TLS"].includes(listenProtocol)
                            ? cert
                            : undefined,
                        defaultActions: buildActions(defaultRule),
                    }, { parent: self }), false)))();
                    customRules.forEach(function (r) {
                        return new aws_1.lb.ListenerRule("".concat(name, "Listener").concat(listenerId, "Rule").concat((0, naming_js_1.hashStringToPrettyString)(JSON.stringify(r.listenConditions), 4)), {
                            listenerArn: listener.arn,
                            actions: buildActions(r),
                            conditions: [
                                {
                                    pathPattern: r.listenConditions.path
                                        ? { values: [r.listenConditions.path] }
                                        : undefined,
                                    queryStrings: r.listenConditions.query,
                                    httpHeader: r.listenConditions.header
                                        ? {
                                            httpHeaderName: r.listenConditions.header.name,
                                            values: r.listenConditions.header.values,
                                        }
                                        : undefined,
                                },
                            ],
                        }, { parent: self });
                    });
                    return listener;
                });
            });
        }
        function createSsl() {
            if (!lbArgs)
                return (0, pulumi_1.output)(undefined);
            return lbArgs.domain.apply(function (domain) {
                if (!domain)
                    return (0, pulumi_1.output)(undefined);
                if (domain.cert)
                    return (0, pulumi_1.output)(domain.cert);
                return new dns_validated_certificate_js_1.DnsValidatedCertificate("".concat(name, "Ssl"), {
                    domainName: domain.name,
                    alternativeNames: domain.aliases,
                    dns: domain.dns,
                }, { parent: self }).arn;
            });
        }
        function createCloudmapService() {
            return (0, pulumi_1.output)(vpc.cloudmapNamespaceId).apply(function (cloudmapNamespaceId) {
                if (!cloudmapNamespaceId)
                    return;
                return new aws_1.servicediscovery.Service("".concat(name, "CloudmapService"), {
                    name: "".concat(name, ".").concat($app.stage, ".").concat($app.name),
                    namespaceId: cloudmapNamespaceId,
                    forceDestroy: true,
                    dnsConfig: {
                        namespaceId: cloudmapNamespaceId,
                        dnsRecords: __spreadArray(__spreadArray([], (args.serviceRegistry ? [{ ttl: 60, type: "SRV" }] : []), true), [
                            { ttl: 60, type: "A" },
                        ], false),
                    },
                }, { parent: self });
            });
        }
        function createService() {
            return cloudmapService.apply(function (cloudmapService) {
                var _a;
                var _b;
                return new ((_a = aws_1.ecs.Service).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.service, "".concat(name, "Service"), __assign(__assign({ name: name, cluster: clusterArn, taskDefinition: taskDefinition.arn, desiredCount: scaling.min }, (capacity
                    ? {
                        // setting `forceNewDeployment` ensures that the service is not recreated
                        // when the capacity provider config changes.
                        forceNewDeployment: true,
                        capacityProviderStrategies: capacity.apply(function (v) {
                            var _a, _b, _c, _d;
                            return __spreadArray(__spreadArray([], (v.fargate
                                ? [
                                    {
                                        capacityProvider: "FARGATE",
                                        base: (_a = v.fargate) === null || _a === void 0 ? void 0 : _a.base,
                                        weight: (_b = v.fargate) === null || _b === void 0 ? void 0 : _b.weight,
                                    },
                                ]
                                : []), true), (v.spot
                                ? [
                                    {
                                        capacityProvider: "FARGATE_SPOT",
                                        base: (_c = v.spot) === null || _c === void 0 ? void 0 : _c.base,
                                        weight: (_d = v.spot) === null || _d === void 0 ? void 0 : _d.weight,
                                    },
                                ]
                                : []), true);
                        }),
                    }
                    : // @deprecated do not use `launchType`, set `capacityProviderStrategies`
                        // to `[{ capacityProvider: "FARGATE", weight: 1 }]` instead
                        {
                            launchType: "FARGATE",
                        })), { networkConfiguration: {
                        // If the vpc is an SST vpc, services are automatically deployed to the public
                        // subnets. So we need to assign a public IP for the service to be accessible.
                        assignPublicIp: vpc.isSstVpc,
                        subnets: vpc.containerSubnets,
                        securityGroups: vpc.securityGroups,
                    }, deploymentCircuitBreaker: {
                        enable: true,
                        rollback: true,
                    }, loadBalancers: lbArgs &&
                        (0, pulumi_1.all)([lbArgs.rules, targetGroups]).apply(function (_a) {
                            var rules = _a[0], targets = _a[1];
                            return Object.values(targets).map(function (target) { return ({
                                targetGroupArn: target.arn,
                                containerName: target.port.apply(function (port) {
                                    return rules.find(function (r) { return r.forwardPort === port; }).container;
                                }),
                                containerPort: target.port.apply(function (port) { return port; }),
                            }); });
                        }), enableExecuteCommand: true, serviceRegistries: cloudmapService && {
                        registryArn: cloudmapService.arn,
                        port: args.serviceRegistry
                            ? (0, pulumi_1.output)(args.serviceRegistry).port
                            : undefined,
                    }, waitForSteadyState: wait }), { parent: self }), false)))();
            });
        }
        function createAutoScaling() {
            var _a;
            var _b;
            var target = new ((_a = aws_1.appautoscaling.Target).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.autoScalingTarget, "".concat(name, "AutoScalingTarget"), {
                serviceNamespace: "ecs",
                scalableDimension: "ecs:service:DesiredCount",
                resourceId: (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["service/", "/", ""], ["service/", "/", ""])), clusterName, service.name),
                maxCapacity: scaling.max,
                minCapacity: scaling.min,
            }, { parent: self }), false)))();
            (0, pulumi_1.output)(scaling.cpuUtilization).apply(function (cpuUtilization) {
                if (cpuUtilization === false)
                    return;
                new aws_1.appautoscaling.Policy("".concat(name, "AutoScalingCpuPolicy"), {
                    serviceNamespace: target.serviceNamespace,
                    scalableDimension: target.scalableDimension,
                    resourceId: target.resourceId,
                    policyType: "TargetTrackingScaling",
                    targetTrackingScalingPolicyConfiguration: {
                        predefinedMetricSpecification: {
                            predefinedMetricType: "ECSServiceAverageCPUUtilization",
                        },
                        targetValue: cpuUtilization,
                    },
                }, { parent: self });
            });
            (0, pulumi_1.output)(scaling.memoryUtilization).apply(function (memoryUtilization) {
                if (memoryUtilization === false)
                    return;
                new aws_1.appautoscaling.Policy("".concat(name, "AutoScalingMemoryPolicy"), {
                    serviceNamespace: target.serviceNamespace,
                    scalableDimension: target.scalableDimension,
                    resourceId: target.resourceId,
                    policyType: "TargetTrackingScaling",
                    targetTrackingScalingPolicyConfiguration: {
                        predefinedMetricSpecification: {
                            predefinedMetricType: "ECSServiceAverageMemoryUtilization",
                        },
                        targetValue: memoryUtilization,
                    },
                }, { parent: self });
            });
            (0, pulumi_1.all)([scaling.requestCount, targetGroups]).apply(function (_a) {
                var requestCount = _a[0], targetGroups = _a[1];
                if (requestCount === false)
                    return;
                if (!targetGroups)
                    return;
                var targetGroup = Object.values(targetGroups)[0];
                new aws_1.appautoscaling.Policy("".concat(name, "AutoScalingRequestCountPolicy"), {
                    serviceNamespace: target.serviceNamespace,
                    scalableDimension: target.scalableDimension,
                    resourceId: target.resourceId,
                    policyType: "TargetTrackingScaling",
                    targetTrackingScalingPolicyConfiguration: {
                        predefinedMetricSpecification: {
                            predefinedMetricType: "ALBRequestCountPerTarget",
                            resourceLabel: (0, pulumi_1.all)([
                                loadBalancer === null || loadBalancer === void 0 ? void 0 : loadBalancer.arn,
                                targetGroup.arn,
                            ]).apply(function (_a) {
                                var _b;
                                var loadBalancerArn = _a[0], targetGroupArn = _a[1];
                                // arn:...:loadbalancer/app/frank-MyServiceLoadBalan/005af2ad12da1e52
                                // => app/frank-MyServiceLoadBalan/005af2ad12da1e52
                                var lbPart = (_b = loadBalancerArn === null || loadBalancerArn === void 0 ? void 0 : loadBalancerArn.split(":").pop()) === null || _b === void 0 ? void 0 : _b.split("/").slice(1).join("/");
                                // arn:...:targetgroup/HTTP20250103004618450100000001/e0811b8cf3a60762
                                // => targetgroup/HTTP20250103004618450100000001
                                var tgPart = targetGroupArn === null || targetGroupArn === void 0 ? void 0 : targetGroupArn.split(":").pop();
                                return "".concat(lbPart, "/").concat(tgPart);
                            }),
                        },
                        targetValue: requestCount,
                    },
                }, { parent: self });
            });
            return target;
        }
        function createDnsRecords() {
            if (!lbArgs)
                return;
            lbArgs.domain.apply(function (domain) {
                if (!(domain === null || domain === void 0 ? void 0 : domain.dns))
                    return;
                for (var _i = 0, _a = __spreadArray([domain.name], domain.aliases, true); _i < _a.length; _i++) {
                    var recordName = _a[_i];
                    var namePrefix = recordName === domain.name ? name : "".concat(name).concat(recordName);
                    domain.dns.createAlias(namePrefix, {
                        name: recordName,
                        aliasName: loadBalancer.dnsName,
                        aliasZone: loadBalancer.zoneId,
                    }, { parent: self });
                }
            });
        }
        function registerReceiver() {
            (0, pulumi_1.all)([containers]).apply(function (_a) {
                var val = _a[0];
                var _loop_1 = function (container) {
                    var title = val.length == 1 ? name : "".concat(name).concat(container.name);
                    new dev_command_js_1.DevCommand("".concat(title, "Dev"), {
                        link: args.link,
                        dev: __assign({ title: title, autostart: true, directory: (function () {
                                if (!container.image)
                                    return "";
                                if (typeof container.image === "string")
                                    return "";
                                if (container.image.context)
                                    return container.image.context;
                                return "";
                            })() }, container.dev),
                        environment: __assign(__assign({}, container.environment), { AWS_REGION: region }),
                        aws: {
                            role: taskRole.arn,
                        },
                    });
                };
                for (var _i = 0, val_1 = val; _i < val_1.length; _i++) {
                    var container = val_1[_i];
                    _loop_1(container);
                }
            });
        }
        return _this;
    }
    Object.defineProperty(Service.prototype, "url", {
        /**
         * The URL of the service.
         *
         * If `public.domain` is set, this is the URL with the custom domain.
         * Otherwise, it's the auto-generated load balancer URL.
         */
        get: function () {
            var errorMessage = "Cannot access the URL because no public ports are exposed.";
            if (this.dev) {
                if (!this.devUrl)
                    throw new error_js_1.VisibleError(errorMessage);
                return this.devUrl;
            }
            if (!this._url)
                throw new error_js_1.VisibleError(errorMessage);
            return this._url;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Service.prototype, "service", {
        /**
         * The name of the Cloud Map service. This is useful for service discovery.
         */
        get: function () {
            var _this = this;
            return (0, pulumi_1.all)([this.cloudmapNamespace, this.cloudmapService]).apply(function (_a) {
                var namespace = _a[0], service = _a[1];
                if (!namespace)
                    throw new error_js_1.VisibleError("Cannot access the AWS Cloud Map service name for the \"".concat(_this._name, "\" Service. Cloud Map is not configured for the cluster."));
                return _this.dev
                    ? (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["dev.", ""], ["dev.", ""])), namespace) : (0, pulumi_1.interpolate)(templateObject_3 || (templateObject_3 = __makeTemplateObject(["", ".", ""], ["", ".", ""])), service.name, namespace);
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Service.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var self = this;
            return {
                /**
                 * The Amazon ECS Service.
                 */
                get service() {
                    if (self.dev)
                        throw new error_js_1.VisibleError("Cannot access `nodes.service` in dev mode.");
                    return self._service;
                },
                /**
                 * The Amazon ECS Execution Role.
                 */
                executionRole: this.executionRole,
                /**
                 * The Amazon ECS Task Role.
                 */
                taskRole: this.taskRole,
                /**
                 * The Amazon ECS Task Definition.
                 */
                get taskDefinition() {
                    if (self.dev)
                        throw new error_js_1.VisibleError("Cannot access `nodes.taskDefinition` in dev mode.");
                    return self.taskDefinition;
                },
                /**
                 * The Amazon Elastic Load Balancer.
                 */
                get loadBalancer() {
                    if (self.dev)
                        throw new error_js_1.VisibleError("Cannot access `nodes.loadBalancer` in dev mode.");
                    if (!self.loadBalancer)
                        throw new error_js_1.VisibleError("Cannot access `nodes.loadBalancer` when no public ports are exposed.");
                    return self.loadBalancer;
                },
                /**
                 * The Amazon Application Auto Scaling target.
                 */
                get autoScalingTarget() {
                    if (self.dev)
                        throw new error_js_1.VisibleError("Cannot access `nodes.autoScalingTarget` in dev mode.");
                    return self.autoScalingTarget;
                },
                /**
                 * The Amazon Cloud Map service.
                 */
                get cloudmapService() {
                    console.log("NODES GETTER");
                    if (self.dev)
                        throw new error_js_1.VisibleError("Cannot access `nodes.cloudmapService` in dev mode.");
                    return (0, pulumi_1.output)(self.cloudmapService).apply(function (service) {
                        if (!service)
                            throw new error_js_1.VisibleError("Cannot access \"nodes.cloudmapService\" for the \"".concat(self._name, "\" Service. Cloud Map is not configured for the cluster."));
                        return service;
                    });
                },
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Service.prototype.getSSTLink = function () {
        var _this = this;
        return {
            properties: {
                url: this.dev ? this.devUrl : this._url,
                service: (0, pulumi_1.output)(this.cloudmapNamespace).apply(function (namespace) {
                    return namespace ? _this.service : undefined;
                }),
            },
        };
    };
    return Service;
}(component_js_1.Component));
exports.Service = Service;
function protocolType(protocol) {
    return ["http", "https"].includes(protocol)
        ? "application"
        : "network";
}
var __pulumiType = "sst:aws:Service";
// @ts-expect-error
Service.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2, templateObject_3;
