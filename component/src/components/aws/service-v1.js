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
var fs_1 = require("fs");
var path_1 = require("path");
var pulumi_1 = require("@pulumi/pulumi");
var docker_build_1 = require("@pulumi/docker-build");
var component_js_1 = require("../component.js");
var size_js_1 = require("../size.js");
var cpu_js_1 = require("../cpu.js");
var dns_js_1 = require("./dns.js");
var error_js_1 = require("../error.js");
var dns_validated_certificate_js_1 = require("./dns-validated-certificate.js");
var link_js_1 = require("../link.js");
var bootstrap_js_1 = require("./helpers/bootstrap.js");
var cluster_v1_js_1 = require("./cluster-v1.js");
var logging_js_1 = require("./logging.js");
var linkable_js_1 = require("./linkable.js");
var aws_1 = require("@pulumi/aws");
var vpc_js_1 = require("./vpc.js");
/**
 * The `Service` component is internally used by the `Cluster` component to deploy services to
 * [Amazon ECS](https://aws.amazon.com/ecs/). It uses [AWS Fargate](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/AWS_Fargate.html).
 *
 * :::note
 * This component is not meant to be created directly.
 * :::
 *
 * This component is returned by the `addService` method of the `Cluster` component.
 */
var Service = /** @class */ (function (_super) {
    __extends(Service, _super);
    function Service(name, args, opts) {
        var _a, _b, _c;
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var self = _this;
        var cluster = (0, pulumi_1.output)(args.cluster);
        var vpc = normalizeVpc();
        var region = normalizeRegion();
        var architecture = normalizeArchitecture();
        var imageArgs = normalizeImage();
        var cpu = normalizeCpu();
        var memory = normalizeMemory();
        var storage = normalizeStorage();
        var scaling = normalizeScaling();
        var logging = normalizeLogging();
        var pub = normalizePublic();
        var linkData = buildLinkData();
        var linkPermissions = buildLinkPermissions();
        var taskRole = createTaskRole();
        _this.taskRole = taskRole;
        if ($dev) {
            _this.devUrl = !pub ? undefined : (0, pulumi_1.output)((_b = (_a = args.dev) === null || _a === void 0 ? void 0 : _a.url) !== null && _b !== void 0 ? _b : linkable_js_1.URL_UNAVAILABLE);
            registerReceiver();
            return _this;
        }
        var bootstrapData = region.apply(function (region) { return bootstrap_js_1.bootstrap.forRegion(region); });
        var executionRole = createExecutionRole();
        var image = createImage();
        var logGroup = createLogGroup();
        var taskDefinition = createTaskDefinition();
        var certificateArn = createSsl();
        var _d = createLoadBalancer(), loadBalancer = _d.loadBalancer, targets = _d.targets;
        var service = createService();
        createAutoScaling();
        createDnsRecords();
        _this.service = service;
        _this.taskDefinition = taskDefinition;
        _this.loadBalancer = loadBalancer;
        _this.domain = (pub === null || pub === void 0 ? void 0 : pub.domain)
            ? pub.domain.apply(function (domain) { return domain === null || domain === void 0 ? void 0 : domain.name; })
            : (0, pulumi_1.output)(undefined);
        _this._url = !self.loadBalancer
            ? undefined
            : (0, pulumi_1.all)([self.domain, (_c = self.loadBalancer) === null || _c === void 0 ? void 0 : _c.dnsName]).apply(function (_a) {
                var domain = _a[0], loadBalancer = _a[1];
                return domain ? "https://".concat(domain, "/") : "http://".concat(loadBalancer);
            });
        registerHint();
        registerReceiver();
        function normalizeVpc() {
            // "vpc" is a Vpc component
            if (args.vpc instanceof vpc_js_1.Vpc) {
                var result_1 = {
                    id: args.vpc.id,
                    publicSubnets: args.vpc.publicSubnets,
                    privateSubnets: args.vpc.privateSubnets,
                    securityGroups: args.vpc.securityGroups,
                };
                return args.vpc.nodes.natGateways.apply(function (natGateways) {
                    if (natGateways.length === 0)
                        throw new error_js_1.VisibleError("The VPC configured for the service does not have NAT enabled. Enable NAT by configuring \"nat\" on the \"sst.aws.Vpc\" component.");
                    return result_1;
                });
            }
            // "vpc" is object
            return (0, pulumi_1.output)(args.vpc);
        }
        function normalizeRegion() {
            return (0, aws_1.getRegionOutput)(undefined, { parent: self }).name;
        }
        function normalizeArchitecture() {
            var _a;
            return (0, pulumi_1.output)((_a = args.architecture) !== null && _a !== void 0 ? _a : "x86_64").apply(function (v) { return v; });
        }
        function normalizeImage() {
            var _a;
            return (0, pulumi_1.all)([(_a = args.image) !== null && _a !== void 0 ? _a : {}, architecture]).apply(function (_a) {
                var _b;
                var image = _a[0], architecture = _a[1];
                return (__assign(__assign({}, image), { context: (_b = image.context) !== null && _b !== void 0 ? _b : ".", platform: architecture === "arm64"
                        ? docker_build_1.Platform.Linux_arm64
                        : docker_build_1.Platform.Linux_amd64 }));
            });
        }
        function normalizeCpu() {
            var _a;
            return (0, pulumi_1.output)((_a = args.cpu) !== null && _a !== void 0 ? _a : "0.25 vCPU").apply(function (v) {
                if (!cluster_v1_js_1.supportedCpus[v]) {
                    throw new Error("Unsupported CPU: ".concat(v, ". The supported values for CPU are ").concat(Object.keys(cluster_v1_js_1.supportedCpus).join(", ")));
                }
                return v;
            });
        }
        function normalizeMemory() {
            var _a;
            return (0, pulumi_1.all)([cpu, (_a = args.memory) !== null && _a !== void 0 ? _a : "0.5 GB"]).apply(function (_a) {
                var cpu = _a[0], v = _a[1];
                if (!(v in cluster_v1_js_1.supportedMemories[cpu])) {
                    throw new Error("Unsupported memory: ".concat(v, ". The supported values for memory for a ").concat(cpu, " CPU are ").concat(Object.keys(cluster_v1_js_1.supportedMemories[cpu]).join(", ")));
                }
                return v;
            });
        }
        function normalizeStorage() {
            var _a;
            return (0, pulumi_1.output)((_a = args.storage) !== null && _a !== void 0 ? _a : "21 GB").apply(function (v) {
                var storage = (0, size_js_1.toGBs)(v);
                if (storage < 21 || storage > 200)
                    throw new Error("Unsupported storage: ".concat(v, ". The supported value for storage is between \"21 GB\" and \"200 GB\""));
                return v;
            });
        }
        function normalizeScaling() {
            return (0, pulumi_1.output)(args.scaling).apply(function (v) {
                var _a, _b, _c, _d;
                return ({
                    min: (_a = v === null || v === void 0 ? void 0 : v.min) !== null && _a !== void 0 ? _a : 1,
                    max: (_b = v === null || v === void 0 ? void 0 : v.max) !== null && _b !== void 0 ? _b : 1,
                    cpuUtilization: (_c = v === null || v === void 0 ? void 0 : v.cpuUtilization) !== null && _c !== void 0 ? _c : 70,
                    memoryUtilization: (_d = v === null || v === void 0 ? void 0 : v.memoryUtilization) !== null && _d !== void 0 ? _d : 70,
                });
            });
        }
        function normalizeLogging() {
            return (0, pulumi_1.output)(args.logging).apply(function (logging) {
                var _a;
                return (__assign(__assign({}, logging), { retention: (_a = logging === null || logging === void 0 ? void 0 : logging.retention) !== null && _a !== void 0 ? _a : "1 month" }));
            });
        }
        function normalizePublic() {
            if (!args.public)
                return;
            var ports = (0, pulumi_1.output)(args.public).apply(function (pub) {
                // validate ports
                if (!pub.ports || pub.ports.length === 0)
                    throw new error_js_1.VisibleError("You must provide the ports to expose via \"public.ports\".");
                // parse protocols and ports
                var ports = pub.ports.map(function (v) {
                    var listenParts = v.listen.split("/");
                    var forwardParts = v.forward ? v.forward.split("/") : listenParts;
                    return {
                        listenPort: parseInt(listenParts[0]),
                        listenProtocol: listenParts[1],
                        forwardPort: parseInt(forwardParts[0]),
                        forwardProtocol: forwardParts[1],
                    };
                });
                // validate protocols are consistent
                var appProtocols = ports.filter(function (port) {
                    return ["http", "https"].includes(port.listenProtocol) &&
                        ["http", "https"].includes(port.forwardProtocol);
                });
                if (appProtocols.length > 0 && appProtocols.length < ports.length)
                    throw new error_js_1.VisibleError("Protocols must be either all http/https, or all tcp/udp/tcp_udp/tls.");
                // validate certificate exists for https/tls protocol
                ports.forEach(function (port) {
                    if (["https", "tls"].includes(port.listenProtocol) && !pub.domain) {
                        throw new error_js_1.VisibleError("You must provide a custom domain for ".concat(port.listenProtocol.toUpperCase(), " protocol."));
                    }
                });
                return ports;
            });
            var domain = (0, pulumi_1.output)(args.public).apply(function (pub) {
                var _a;
                if (!pub.domain)
                    return undefined;
                // normalize domain
                var domain = typeof pub.domain === "string" ? { name: pub.domain } : pub.domain;
                return {
                    name: domain.name,
                    dns: domain.dns === false ? undefined : (_a = domain.dns) !== null && _a !== void 0 ? _a : (0, dns_js_1.dns)(),
                    cert: domain.cert,
                };
            });
            return { ports: ports, domain: domain };
        }
        function buildLinkData() {
            return (0, pulumi_1.output)(args.link || []).apply(function (links) { return link_js_1.Link.build(links); });
        }
        function buildLinkPermissions() {
            return link_js_1.Link.getInclude("aws.permission", args.link);
        }
        function createImage() {
            var _a;
            // Edit .dockerignore file
            var imageArgsNew = imageArgs.apply(function (imageArgs) {
                var _a;
                var context = path_1.default.join($cli.paths.root, imageArgs.context);
                var dockerfile = (_a = imageArgs.dockerfile) !== null && _a !== void 0 ? _a : "Dockerfile";
                // get .dockerignore file
                var file = (function () {
                    var filePath = path_1.default.join(context, "".concat(dockerfile, ".dockerignore"));
                    if (fs_1.default.existsSync(filePath))
                        return filePath;
                    filePath = path_1.default.join(context, ".dockerignore");
                    if (fs_1.default.existsSync(filePath))
                        return filePath;
                })();
                // add .sst to .dockerignore if not exist
                var content = file ? fs_1.default.readFileSync(file).toString() : "";
                var lines = content.split("\n");
                if (!lines.find(function (line) { return line === ".sst"; })) {
                    fs_1.default.writeFileSync(file !== null && file !== void 0 ? file : path_1.default.join(context, ".dockerignore"), __spreadArray(__spreadArray([], lines, true), ["", "# sst", ".sst"], false).join("\n"));
                }
                return imageArgs;
            });
            // Build image
            return new (docker_build_1.Image.bind.apply(docker_build_1.Image, __spreadArray([void 0], (0, component_js_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.image, "".concat(name, "Image"), {
                context: {
                    location: imageArgsNew.apply(function (v) {
                        return path_1.default.join($cli.paths.root, v.context);
                    }),
                },
                dockerfile: {
                    location: imageArgsNew.apply(function (v) {
                        return v.dockerfile
                            ? path_1.default.join($cli.paths.root, v.dockerfile)
                            : path_1.default.join($cli.paths.root, v.context, "Dockerfile");
                    }),
                },
                buildArgs: imageArgsNew.apply(function (v) { var _a; return (_a = v.args) !== null && _a !== void 0 ? _a : {}; }),
                platforms: [imageArgs.platform],
                tags: [(0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["", ":", ""], ["", ":", ""])), bootstrapData.assetEcrUrl, name)],
                registries: [
                    aws_1.ecr
                        .getAuthorizationTokenOutput({
                        registryId: bootstrapData.assetEcrRegistryId,
                    })
                        .apply(function (authToken) { return ({
                        address: authToken.proxyEndpoint,
                        password: (0, pulumi_1.secret)(authToken.password),
                        username: authToken.userName,
                    }); }),
                ],
                push: true,
            }, { parent: self }), false)))();
        }
        function createLoadBalancer() {
            var _a, _b;
            var _c, _d;
            if (!pub)
                return {};
            var securityGroup = new ((_a = aws_1.ec2.SecurityGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_c = args === null || args === void 0 ? void 0 : args.transform) === null || _c === void 0 ? void 0 : _c.loadBalancerSecurityGroup, "".concat(name, "LoadBalancerSecurityGroup"), {
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
            var loadBalancer = new ((_b = aws_1.lb.LoadBalancer).bind.apply(_b, __spreadArray([void 0], (0, component_js_1.transform)((_d = args.transform) === null || _d === void 0 ? void 0 : _d.loadBalancer, "".concat(name, "LoadBalancer"), {
                internal: false,
                loadBalancerType: pub.ports.apply(function (ports) {
                    return ports[0].listenProtocol.startsWith("http")
                        ? "application"
                        : "network";
                }),
                subnets: vpc.publicSubnets,
                securityGroups: [securityGroup.id],
                enableCrossZoneLoadBalancing: true,
            }, { parent: self }), false)))();
            var ret = (0, pulumi_1.all)([pub.ports, certificateArn]).apply(function (_a) {
                var ports = _a[0], cert = _a[1];
                var listeners = {};
                var targets = {};
                ports.forEach(function (port) {
                    var _a, _b;
                    var _c, _d, _e, _f;
                    var forwardProtocol = port.forwardProtocol.toUpperCase();
                    var forwardPort = port.forwardPort;
                    var targetId = "".concat(forwardProtocol).concat(forwardPort);
                    var target = (_c = targets[targetId]) !== null && _c !== void 0 ? _c : new ((_a = aws_1.lb.TargetGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_d = args.transform) === null || _d === void 0 ? void 0 : _d.target, "".concat(name, "Target").concat(targetId), {
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
                    }, { parent: self }), false)))();
                    targets[targetId] = target;
                    var listenProtocol = port.listenProtocol.toUpperCase();
                    var listenPort = port.listenPort;
                    var listenerId = "".concat(listenProtocol).concat(listenPort);
                    var listener = (_e = listeners[listenerId]) !== null && _e !== void 0 ? _e : new ((_b = aws_1.lb.Listener).bind.apply(_b, __spreadArray([void 0], (0, component_js_1.transform)((_f = args.transform) === null || _f === void 0 ? void 0 : _f.listener, "".concat(name, "Listener").concat(listenerId), {
                        loadBalancerArn: loadBalancer.arn,
                        port: listenPort,
                        protocol: listenProtocol,
                        certificateArn: ["HTTPS", "TLS"].includes(listenProtocol)
                            ? cert
                            : undefined,
                        defaultActions: [
                            {
                                type: "forward",
                                targetGroupArn: target.arn,
                            },
                        ],
                    }, { parent: self }), false)))();
                    listeners[listenerId] = listener;
                });
                return { listeners: listeners, targets: targets };
            });
            return { loadBalancer: loadBalancer, targets: ret.targets };
        }
        function createSsl() {
            if (!pub)
                return (0, pulumi_1.output)(undefined);
            return pub.domain.apply(function (domain) {
                if (!domain)
                    return (0, pulumi_1.output)(undefined);
                if (domain.cert)
                    return (0, pulumi_1.output)(domain.cert);
                return new dns_validated_certificate_js_1.DnsValidatedCertificate("".concat(name, "Ssl"), {
                    domainName: domain.name,
                    dns: domain.dns,
                }, { parent: self }).arn;
            });
        }
        function createLogGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.cloudwatch.LogGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.logGroup, "".concat(name, "LogGroup"), {
                name: (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["/sst/cluster/", "/", ""], ["/sst/cluster/", "/", ""])), cluster.name, name),
                retentionInDays: logging.apply(function (logging) { return logging_js_1.RETENTION[logging.retention]; }),
            }, { parent: self }), false)))();
        }
        function createTaskRole() {
            var _a;
            var _b;
            var policy = (0, pulumi_1.all)([args.permissions || [], linkPermissions]).apply(function (_a) {
                var argsPermissions = _a[0], linkPermissions = _a[1];
                return aws_1.iam.getPolicyDocumentOutput({
                    statements: __spreadArray(__spreadArray([], argsPermissions, true), linkPermissions, true).map(function (item) { return ({
                        effect: (function () {
                            var _a;
                            var effect = (_a = item.effect) !== null && _a !== void 0 ? _a : "allow";
                            return effect.charAt(0).toUpperCase() + effect.slice(1);
                        })(),
                        actions: item.actions,
                        resources: item.resources,
                    }); }),
                });
            });
            return new ((_a = aws_1.iam.Role).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.taskRole, "".concat(name, "TaskRole"), {
                assumeRolePolicy: !$dev
                    ? aws_1.iam.assumeRolePolicyForPrincipal({
                        Service: "ecs-tasks.amazonaws.com",
                    })
                    : aws_1.iam.assumeRolePolicyForPrincipal({
                        AWS: (0, pulumi_1.interpolate)(templateObject_3 || (templateObject_3 = __makeTemplateObject(["arn:aws:iam::", ":root"], ["arn:aws:iam::", ":root"])), (0, aws_1.getCallerIdentityOutput)().accountId),
                    }),
                inlinePolicies: policy.apply(function (_a) {
                    var statements = _a.statements;
                    return statements ? [{ name: "inline", policy: policy.json }] : [];
                }),
            }, { parent: self }), false)))();
        }
        function createExecutionRole() {
            return new aws_1.iam.Role("".concat(name, "ExecutionRole"), {
                assumeRolePolicy: aws_1.iam.assumeRolePolicyForPrincipal({
                    Service: "ecs-tasks.amazonaws.com",
                }),
                managedPolicyArns: [
                    "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy",
                ],
            }, { parent: self });
        }
        function createTaskDefinition() {
            var _a;
            var _b, _c;
            return new ((_a = aws_1.ecs.TaskDefinition).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.taskDefinition, "".concat(name, "Task"), {
                family: (0, pulumi_1.interpolate)(templateObject_4 || (templateObject_4 = __makeTemplateObject(["", "-", ""], ["", "-", ""])), cluster.name, name),
                trackLatest: true,
                cpu: cpu.apply(function (v) { return (0, cpu_js_1.toNumber)(v).toString(); }),
                memory: memory.apply(function (v) { return (0, size_js_1.toMBs)(v).toString(); }),
                networkMode: "awsvpc",
                ephemeralStorage: {
                    sizeInGib: storage.apply(function (v) { return (0, size_js_1.toGBs)(v); }),
                },
                requiresCompatibilities: ["FARGATE"],
                runtimePlatform: {
                    cpuArchitecture: architecture.apply(function (v) { return v.toUpperCase(); }),
                    operatingSystemFamily: "LINUX",
                },
                executionRoleArn: executionRole.arn,
                taskRoleArn: taskRole.arn,
                containerDefinitions: $jsonStringify([
                    {
                        name: name,
                        image: (0, pulumi_1.interpolate)(templateObject_5 || (templateObject_5 = __makeTemplateObject(["", "@", ""], ["", "@", ""])), bootstrapData.assetEcrUrl, image.digest),
                        pseudoTerminal: true,
                        portMappings: pub === null || pub === void 0 ? void 0 : pub.ports.apply(function (ports) {
                            return ports
                                .map(function (port) { return port.forwardPort; })
                                // ensure unique ports
                                .filter(function (value, index, self) { return self.indexOf(value) === index; })
                                .map(function (value) { return ({ containerPort: value }); });
                        }),
                        logConfiguration: {
                            logDriver: "awslogs",
                            options: {
                                "awslogs-group": logGroup.name,
                                "awslogs-region": region,
                                "awslogs-stream-prefix": "/service",
                            },
                        },
                        environment: (0, pulumi_1.all)([(_c = args.environment) !== null && _c !== void 0 ? _c : [], linkData]).apply(function (_a) {
                            var env = _a[0], linkData = _a[1];
                            return __spreadArray(__spreadArray(__spreadArray([], Object.entries(env).map(function (_a) {
                                var name = _a[0], value = _a[1];
                                return ({
                                    name: name,
                                    value: value,
                                });
                            }), true), linkData.map(function (d) { return ({
                                name: "SST_RESOURCE_".concat(d.name),
                                value: JSON.stringify(d.properties),
                            }); }), true), [
                                {
                                    name: "SST_RESOURCE_App",
                                    value: JSON.stringify({
                                        name: $app.name,
                                        stage: $app.stage,
                                    }),
                                },
                            ], false);
                        }),
                    },
                ]),
            }, { parent: self }), false)))();
        }
        function createService() {
            var _a;
            var _b;
            return new ((_a = aws_1.ecs.Service).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.service, "".concat(name, "Service"), {
                name: name,
                cluster: cluster.arn,
                taskDefinition: taskDefinition.arn,
                desiredCount: scaling.min,
                launchType: "FARGATE",
                networkConfiguration: {
                    assignPublicIp: false,
                    subnets: vpc.privateSubnets,
                    securityGroups: vpc.securityGroups,
                },
                deploymentCircuitBreaker: {
                    enable: true,
                    rollback: true,
                },
                loadBalancers: targets &&
                    targets.apply(function (targets) {
                        return Object.values(targets).map(function (target) { return ({
                            targetGroupArn: target.arn,
                            containerName: name,
                            containerPort: target.port.apply(function (port) { return port; }),
                        }); });
                    }),
            }, { parent: self }), false)))();
        }
        function createAutoScaling() {
            var target = new aws_1.appautoscaling.Target("".concat(name, "AutoScalingTarget"), {
                serviceNamespace: "ecs",
                scalableDimension: "ecs:service:DesiredCount",
                resourceId: (0, pulumi_1.interpolate)(templateObject_6 || (templateObject_6 = __makeTemplateObject(["service/", "/", ""], ["service/", "/", ""])), cluster.name, service.name),
                maxCapacity: scaling.max,
                minCapacity: scaling.min,
            }, { parent: self });
            new aws_1.appautoscaling.Policy("".concat(name, "AutoScalingCpuPolicy"), {
                serviceNamespace: target.serviceNamespace,
                scalableDimension: target.scalableDimension,
                resourceId: target.resourceId,
                policyType: "TargetTrackingScaling",
                targetTrackingScalingPolicyConfiguration: {
                    predefinedMetricSpecification: {
                        predefinedMetricType: "ECSServiceAverageCPUUtilization",
                    },
                    targetValue: scaling.cpuUtilization,
                },
            }, { parent: self });
            new aws_1.appautoscaling.Policy("".concat(name, "AutoScalingMemoryPolicy"), {
                serviceNamespace: target.serviceNamespace,
                scalableDimension: target.scalableDimension,
                resourceId: target.resourceId,
                policyType: "TargetTrackingScaling",
                targetTrackingScalingPolicyConfiguration: {
                    predefinedMetricSpecification: {
                        predefinedMetricType: "ECSServiceAverageMemoryUtilization",
                    },
                    targetValue: scaling.memoryUtilization,
                },
            }, { parent: self });
        }
        function createDnsRecords() {
            if (!pub)
                return;
            pub.domain.apply(function (domain) {
                if (!(domain === null || domain === void 0 ? void 0 : domain.dns))
                    return;
                domain.dns.createAlias(name, {
                    name: domain.name,
                    aliasName: loadBalancer.dnsName,
                    aliasZone: loadBalancer.zoneId,
                }, { parent: self });
            });
        }
        function registerHint() {
            self.registerOutputs({ _hint: self._url });
        }
        function registerReceiver() {
            self.registerOutputs({
                _dev: imageArgs.apply(function (imageArgs) {
                    var _a, _b, _c;
                    return ({
                        links: linkData.apply(function (input) { return input.map(function (item) { return item.name; }); }),
                        environment: __assign(__assign({}, args.environment), { AWS_REGION: region }),
                        aws: {
                            role: taskRole.arn,
                        },
                        autostart: (0, pulumi_1.output)((_a = args.dev) === null || _a === void 0 ? void 0 : _a.autostart).apply(function (val) { return val !== null && val !== void 0 ? val : true; }),
                        directory: (0, pulumi_1.output)((_b = args.dev) === null || _b === void 0 ? void 0 : _b.directory).apply(function (dir) {
                            return dir ||
                                path_1.default.join(imageArgs.dockerfile
                                    ? path_1.default.dirname(imageArgs.dockerfile)
                                    : imageArgs.context);
                        }),
                        command: (_c = args.dev) === null || _c === void 0 ? void 0 : _c.command,
                    });
                }),
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
            if ($dev) {
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
                    if ($dev)
                        throw new error_js_1.VisibleError("Cannot access `nodes.service` in dev mode.");
                    return self.service;
                },
                /**
                 * The Amazon ECS Task Role.
                 */
                get taskRole() {
                    return self.taskRole;
                },
                /**
                 * The Amazon ECS Task Definition.
                 */
                get taskDefinition() {
                    if ($dev)
                        throw new error_js_1.VisibleError("Cannot access `nodes.taskDefinition` in dev mode.");
                    return self.taskDefinition;
                },
                /**
                 * The Amazon Elastic Load Balancer.
                 */
                get loadBalancer() {
                    if ($dev)
                        throw new error_js_1.VisibleError("Cannot access `nodes.loadBalancer` in dev mode.");
                    if (!self.loadBalancer)
                        throw new error_js_1.VisibleError("Cannot access `nodes.loadBalancer` when no public ports are exposed.");
                    return self.loadBalancer;
                },
            };
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Service.prototype.getSSTLink = function () {
        return {
            properties: { url: $dev ? this.devUrl : this._url },
        };
    };
    return Service;
}(component_js_1.Component));
exports.Service = Service;
var __pulumiType = "sst:aws:Service";
// @ts-expect-error
Service.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2, templateObject_3, templateObject_4, templateObject_5, templateObject_6;
