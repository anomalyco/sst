"use strict";
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
exports.supportedMemories = exports.supportedCpus = void 0;
exports.normalizeArchitecture = normalizeArchitecture;
exports.normalizeCpu = normalizeCpu;
exports.normalizeMemory = normalizeMemory;
exports.normalizeStorage = normalizeStorage;
exports.normalizeContainers = normalizeContainers;
exports.createTaskRole = createTaskRole;
exports.createExecutionRole = createExecutionRole;
exports.createTaskDefinition = createTaskDefinition;
var fs_1 = require("fs");
var path_1 = require("path");
var pulumi_1 = require("@pulumi/pulumi");
var pulumi_2 = require("@pulumi/pulumi");
var efs_1 = require("./efs");
var logging_1 = require("./logging");
var size_1 = require("../size");
var error_1 = require("../error");
var docker_build_1 = require("@pulumi/docker-build");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var link_1 = require("../link");
var bootstrap_1 = require("./helpers/bootstrap");
var container_builder_1 = require("./helpers/container-builder");
var cpu_1 = require("../cpu");
var duration_1 = require("../duration");
exports.supportedCpus = {
    "0.25 vCPU": 256,
    "0.5 vCPU": 512,
    "1 vCPU": 1024,
    "2 vCPU": 2048,
    "4 vCPU": 4096,
    "8 vCPU": 8192,
    "16 vCPU": 16384,
};
exports.supportedMemories = {
    "0.25 vCPU": {
        "0.5 GB": 512,
        "1 GB": 1024,
        "2 GB": 2048,
    },
    "0.5 vCPU": {
        "1 GB": 1024,
        "2 GB": 2048,
        "3 GB": 3072,
        "4 GB": 4096,
    },
    "1 vCPU": {
        "2 GB": 2048,
        "3 GB": 3072,
        "4 GB": 4096,
        "5 GB": 5120,
        "6 GB": 6144,
        "7 GB": 7168,
        "8 GB": 8192,
    },
    "2 vCPU": {
        "4 GB": 4096,
        "5 GB": 5120,
        "6 GB": 6144,
        "7 GB": 7168,
        "8 GB": 8192,
        "9 GB": 9216,
        "10 GB": 10240,
        "11 GB": 11264,
        "12 GB": 12288,
        "13 GB": 13312,
        "14 GB": 14336,
        "15 GB": 15360,
        "16 GB": 16384,
    },
    "4 vCPU": {
        "8 GB": 8192,
        "9 GB": 9216,
        "10 GB": 10240,
        "11 GB": 11264,
        "12 GB": 12288,
        "13 GB": 13312,
        "14 GB": 14336,
        "15 GB": 15360,
        "16 GB": 16384,
        "17 GB": 17408,
        "18 GB": 18432,
        "19 GB": 19456,
        "20 GB": 20480,
        "21 GB": 21504,
        "22 GB": 22528,
        "23 GB": 23552,
        "24 GB": 24576,
        "25 GB": 25600,
        "26 GB": 26624,
        "27 GB": 27648,
        "28 GB": 28672,
        "29 GB": 29696,
        "30 GB": 30720,
    },
    "8 vCPU": {
        "16 GB": 16384,
        "20 GB": 20480,
        "24 GB": 24576,
        "28 GB": 28672,
        "32 GB": 32768,
        "36 GB": 36864,
        "40 GB": 40960,
        "44 GB": 45056,
        "48 GB": 49152,
        "52 GB": 53248,
        "56 GB": 57344,
        "60 GB": 61440,
    },
    "16 vCPU": {
        "32 GB": 32768,
        "40 GB": 40960,
        "48 GB": 49152,
        "56 GB": 57344,
        "64 GB": 65536,
        "72 GB": 73728,
        "80 GB": 81920,
        "88 GB": 90112,
        "96 GB": 98304,
        "104 GB": 106496,
        "112 GB": 114688,
        "120 GB": 122880,
    },
};
function normalizeArchitecture(args) {
    var _a;
    return (0, pulumi_2.output)((_a = args.architecture) !== null && _a !== void 0 ? _a : "x86_64").apply(function (v) { return v; });
}
function normalizeCpu(args) {
    var _a;
    return (0, pulumi_2.output)((_a = args.cpu) !== null && _a !== void 0 ? _a : "0.25 vCPU").apply(function (v) {
        if (!exports.supportedCpus[v]) {
            throw new Error("Unsupported CPU: ".concat(v, ". The supported values for CPU are ").concat(Object.keys(exports.supportedCpus).join(", ")));
        }
        return v;
    });
}
function normalizeMemory(cpu, args) {
    var _a;
    return (0, pulumi_2.all)([cpu, (_a = args.memory) !== null && _a !== void 0 ? _a : "0.5 GB"]).apply(function (_a) {
        var cpu = _a[0], v = _a[1];
        if (!(v in exports.supportedMemories[cpu])) {
            throw new Error("Unsupported memory: ".concat(v, ". The supported values for memory for a ").concat(cpu, " CPU are ").concat(Object.keys(exports.supportedMemories[cpu]).join(", ")));
        }
        return v;
    });
}
function normalizeStorage(args) {
    var _a;
    return (0, pulumi_2.output)((_a = args.storage) !== null && _a !== void 0 ? _a : "20 GB").apply(function (v) {
        var storage = (0, size_1.toGBs)(v);
        if (storage < 20 || storage > 200)
            throw new Error("Unsupported storage: ".concat(v, ". The supported value for storage is between \"20 GB\" and \"200 GB\""));
        return v;
    });
}
function normalizeContainers(type, args, name, architecture) {
    var _a;
    if (args.containers &&
        (args.image ||
            args.logging ||
            args.environment ||
            args.volumes ||
            args.health ||
            args.ssm)) {
        throw new error_1.VisibleError(type === "service"
            ? "You cannot provide both \"containers\" and \"image\", \"logging\", \"environment\", \"volumes\", \"health\" or \"ssm\"."
            : "You cannot provide both \"containers\" and \"image\", \"logging\", \"environment\", \"volumes\" or \"ssm\".");
    }
    // Standardize containers
    var containers = (_a = args.containers) !== null && _a !== void 0 ? _a : [
        {
            name: name,
            cpu: undefined,
            memory: undefined,
            image: args.image,
            logging: args.logging,
            environment: args.environment,
            ssm: args.ssm,
            volumes: args.volumes,
            command: args.command,
            entrypoint: args.entrypoint,
            health: type === "service" ? args.health : undefined,
            dev: type === "service" ? args.dev : undefined,
        },
    ];
    // Normalize container props
    return (0, pulumi_2.output)(containers).apply(function (containers) {
        return containers.map(function (v) {
            return __assign(__assign({}, v), { volumes: normalizeVolumes(), image: normalizeImage(), logging: normalizeLogging() });
            function normalizeVolumes() {
                return (0, pulumi_2.output)(v.volumes).apply(function (volumes) {
                    return volumes === null || volumes === void 0 ? void 0 : volumes.map(function (volume) { return ({
                        path: volume.path,
                        efs: volume.efs instanceof efs_1.Efs
                            ? {
                                fileSystem: volume.efs.id,
                                accessPoint: volume.efs.accessPoint,
                            }
                            : volume.efs,
                    }); });
                });
            }
            function normalizeImage() {
                return (0, pulumi_2.all)([v.image, architecture]).apply(function (_a) {
                    var _b;
                    var image = _a[0], architecture = _a[1];
                    if (typeof image === "string")
                        return image;
                    return __assign(__assign({}, image), { context: (_b = image === null || image === void 0 ? void 0 : image.context) !== null && _b !== void 0 ? _b : ".", platform: architecture === "arm64"
                            ? docker_build_1.Platform.Linux_arm64
                            : docker_build_1.Platform.Linux_amd64 });
                });
            }
            function normalizeLogging() {
                return (0, pulumi_2.all)([v.logging, args.cluster.nodes.cluster.name]).apply(function (_a) {
                    var _b, _c;
                    var logging = _a[0], clusterName = _a[1];
                    return (__assign(__assign({}, logging), { retention: (_b = logging === null || logging === void 0 ? void 0 : logging.retention) !== null && _b !== void 0 ? _b : "1 month", name: (_c = logging === null || logging === void 0 ? void 0 : logging.name) !== null && _c !== void 0 ? _c : "/sst/cluster/".concat(clusterName, "/").concat(name, "/").concat(v.name) }));
                });
            }
        });
    });
}
function createTaskRole(name, args, opts, parent, dev, additionalPermissions) {
    var _a;
    var _b;
    if (args.taskRole)
        return aws_1.iam.Role.get("".concat(name, "TaskRole"), args.taskRole, {}, { parent: parent });
    var policy = (0, pulumi_2.all)([
        args.permissions || [],
        link_1.Link.getInclude("aws.permission", args.link),
        additionalPermissions,
    ]).apply(function (_a) {
        var argsPermissions = _a[0], linkPermissions = _a[1], additionalPermissions = _a[2];
        return aws_1.iam.getPolicyDocumentOutput({
            statements: __spreadArray(__spreadArray(__spreadArray(__spreadArray([], argsPermissions, true), linkPermissions, true), (additionalPermissions !== null && additionalPermissions !== void 0 ? additionalPermissions : []), true), [
                {
                    actions: [
                        "ssmmessages:CreateControlChannel",
                        "ssmmessages:CreateDataChannel",
                        "ssmmessages:OpenControlChannel",
                        "ssmmessages:OpenDataChannel",
                    ],
                    resources: ["*"],
                },
            ], false).map(function (item) { return ({
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
    return new ((_a = aws_1.iam.Role).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.taskRole, "".concat(name, "TaskRole"), {
        assumeRolePolicy: aws_1.iam.assumeRolePolicyForPrincipal(__assign({ Service: "ecs-tasks.amazonaws.com" }, (dev ? { AWS: (0, aws_1.getCallerIdentityOutput)({}, opts).accountId } : {}))),
        inlinePolicies: policy.apply(function (_a) {
            var statements = _a.statements;
            return statements ? [{ name: "inline", policy: policy.json }] : [];
        }),
    }, { parent: parent }), false)))();
}
function createExecutionRole(name, args, opts, parent) {
    var _a;
    var _b;
    if (args.executionRole)
        return aws_1.iam.Role.get("".concat(name, "ExecutionRole"), args.executionRole, {}, { parent: parent });
    return new ((_a = aws_1.iam.Role).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.executionRole, "".concat(name, "ExecutionRole"), {
        assumeRolePolicy: aws_1.iam.assumeRolePolicyForPrincipal({
            Service: "ecs-tasks.amazonaws.com",
        }),
        managedPolicyArns: [
            (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["arn:", ":iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"], ["arn:", ":iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"])), (0, aws_1.getPartitionOutput)({}, opts).partition),
        ],
        inlinePolicies: [
            {
                name: "inline",
                policy: aws_1.iam.getPolicyDocumentOutput({
                    statements: [
                        {
                            sid: "ReadSsmAndSecrets",
                            actions: [
                                "ssm:GetParameters",
                                "ssm:GetParameter",
                                "ssm:GetParameterHistory",
                                "secretsmanager:GetSecretValue",
                            ],
                            resources: ["*"],
                        },
                    ],
                }).json,
            },
        ],
    }, { parent: parent }), false)))();
}
function createTaskDefinition(name, args, opts, parent, containers, architecture, cpu, memory, storage, taskRole, executionRole) {
    var clusterName = args.cluster.nodes.cluster.name;
    var region = (0, aws_1.getRegionOutput)({}, opts).name;
    var bootstrapData = region.apply(function (region) { return bootstrap_1.bootstrap.forRegion(region); });
    var linkEnvs = link_1.Link.propertiesToEnv(link_1.Link.getProperties(args.link));
    var containerDefinitions = (0, pulumi_2.output)(containers).apply(function (containers) {
        return containers.map(function (container) {
            var _a, _b, _c, _d, _e, _f;
            return ({
                name: container.name,
                image: (function () {
                    var _a, _b, _c;
                    if (typeof container.image === "string")
                        return (0, pulumi_2.output)(container.image);
                    var containerImage = container.image;
                    var contextPath = path_1.default.join($cli.paths.root, container.image.context);
                    var dockerfile = (_a = container.image.dockerfile) !== null && _a !== void 0 ? _a : "Dockerfile";
                    var dockerfilePath = container.image.dockerfile
                        ? path_1.default.join($cli.paths.root, container.image.dockerfile)
                        : path_1.default.join($cli.paths.root, container.image.context, "Dockerfile");
                    var dockerIgnorePath = fs_1.default.existsSync(path_1.default.join(contextPath, "".concat(dockerfile, ".dockerignore")))
                        ? path_1.default.join(contextPath, "".concat(dockerfile, ".dockerignore"))
                        : path_1.default.join(contextPath, ".dockerignore");
                    // add .sst to .dockerignore if not exist
                    var lines = fs_1.default.existsSync(dockerIgnorePath)
                        ? fs_1.default.readFileSync(dockerIgnorePath).toString().split("\n")
                        : [];
                    if (!lines.find(function (line) { return line === ".sst"; })) {
                        fs_1.default.writeFileSync(dockerIgnorePath, __spreadArray(__spreadArray([], lines, true), ["", "# sst", ".sst"], false).join("\n"));
                    }
                    // Build image
                    var image = container_builder_1.imageBuilder.apply(void 0, (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.image, "".concat(name, "Image").concat(container.name), {
                        context: { location: contextPath },
                        dockerfile: { location: dockerfilePath },
                        buildArgs: linkEnvs.apply(function (linkEnvs) { return (__assign(__assign({}, containerImage.args), linkEnvs)); }),
                        target: container.image.target,
                        platforms: [container.image.platform],
                        tags: __spreadArray([container.name], ((_c = container.image.tags) !== null && _c !== void 0 ? _c : []), true).map(function (tag) { return (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["", ":", ""], ["", ":", ""])), bootstrapData.assetEcrUrl, tag); }),
                        registries: [
                            aws_1.ecr
                                .getAuthorizationTokenOutput({
                                registryId: bootstrapData.assetEcrRegistryId,
                            }, { parent: parent })
                                .apply(function (authToken) { return ({
                                address: authToken.proxyEndpoint,
                                password: (0, pulumi_1.secret)(authToken.password),
                                username: authToken.userName,
                            }); }),
                        ],
                        cacheFrom: [
                            {
                                registry: {
                                    ref: (0, pulumi_1.interpolate)(templateObject_3 || (templateObject_3 = __makeTemplateObject(["", ":", "-cache"], ["", ":", "-cache"])), bootstrapData.assetEcrUrl, container.name),
                                },
                            },
                        ],
                        cacheTo: [
                            {
                                registry: {
                                    ref: (0, pulumi_1.interpolate)(templateObject_4 || (templateObject_4 = __makeTemplateObject(["", ":", "-cache"], ["", ":", "-cache"])), bootstrapData.assetEcrUrl, container.name),
                                    imageManifest: true,
                                    ociMediaTypes: true,
                                    mode: "max",
                                },
                            },
                        ],
                        push: true,
                    }, { parent: parent }));
                    return (0, pulumi_1.interpolate)(templateObject_5 || (templateObject_5 = __makeTemplateObject(["", "@", ""], ["", "@", ""])), bootstrapData.assetEcrUrl, image.digest);
                })(),
                cpu: container.cpu ? (0, cpu_1.toNumber)(container.cpu) : undefined,
                memory: container.memory ? (0, size_1.toMBs)(container.memory) : undefined,
                command: container.command,
                entrypoint: container.entrypoint,
                healthCheck: container.health && {
                    command: container.health.command,
                    startPeriod: (0, duration_1.toSeconds)((_a = container.health.startPeriod) !== null && _a !== void 0 ? _a : "0 seconds"),
                    timeout: (0, duration_1.toSeconds)((_b = container.health.timeout) !== null && _b !== void 0 ? _b : "5 seconds"),
                    interval: (0, duration_1.toSeconds)((_c = container.health.interval) !== null && _c !== void 0 ? _c : "30 seconds"),
                    retries: (_d = container.health.retries) !== null && _d !== void 0 ? _d : 3,
                },
                pseudoTerminal: true,
                portMappings: [{ containerPortRange: "1-65535" }],
                logConfiguration: {
                    logDriver: "awslogs",
                    options: {
                        "awslogs-group": (function () {
                            var _a;
                            var _b;
                            return new ((_a = aws_1.cloudwatch.LogGroup).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.logGroup, "".concat(name, "LogGroup").concat(container.name), {
                                name: container.logging.name,
                                retentionInDays: logging_1.RETENTION[container.logging.retention],
                            }, { parent: parent, ignoreChanges: ["name"] }), false)))();
                        })().name,
                        "awslogs-region": region,
                        "awslogs-stream-prefix": "/service",
                    },
                },
                environment: linkEnvs.apply(function (linkEnvs) {
                    return Object.entries(__assign(__assign({}, container.environment), linkEnvs)).map(function (_a) {
                        var name = _a[0], value = _a[1];
                        return ({ name: name, value: value });
                    });
                }),
                linuxParameters: {
                    initProcessEnabled: true,
                },
                mountPoints: (_e = container.volumes) === null || _e === void 0 ? void 0 : _e.map(function (volume) { return ({
                    sourceVolume: volume.efs.accessPoint,
                    containerPath: volume.path,
                }); }),
                secrets: Object.entries((_f = container.ssm) !== null && _f !== void 0 ? _f : {}).map(function (_a) {
                    var name = _a[0], valueFrom = _a[1];
                    return ({
                        name: name,
                        valueFrom: valueFrom,
                    });
                }),
            });
        });
    });
    return storage.apply(function (storage) {
        var _a;
        var _b;
        return new ((_a = aws_1.ecs.TaskDefinition).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.taskDefinition, "".concat(name, "Task"), {
            family: (0, pulumi_1.interpolate)(templateObject_6 || (templateObject_6 = __makeTemplateObject(["", "-", ""], ["", "-", ""])), clusterName, name),
            trackLatest: true,
            cpu: cpu.apply(function (v) { return (0, cpu_1.toNumber)(v).toString(); }),
            memory: memory.apply(function (v) { return (0, size_1.toMBs)(v).toString(); }),
            networkMode: "awsvpc",
            ephemeralStorage: (function () {
                var sizeInGib = (0, size_1.toGBs)(storage);
                return sizeInGib === 20 ? undefined : { sizeInGib: sizeInGib };
            })(),
            requiresCompatibilities: ["FARGATE"],
            runtimePlatform: {
                cpuArchitecture: architecture.apply(function (v) { return v.toUpperCase(); }),
                operatingSystemFamily: "LINUX",
            },
            executionRoleArn: executionRole.arn,
            taskRoleArn: taskRole.arn,
            volumes: (0, pulumi_2.output)(containers).apply(function (containers) {
                var uniqueAccessPoints = new Set();
                return containers.flatMap(function (container) {
                    var _a;
                    return ((_a = container.volumes) !== null && _a !== void 0 ? _a : []).flatMap(function (volume) {
                        if (uniqueAccessPoints.has(volume.efs.accessPoint))
                            return [];
                        uniqueAccessPoints.add(volume.efs.accessPoint);
                        return {
                            name: volume.efs.accessPoint,
                            efsVolumeConfiguration: {
                                fileSystemId: volume.efs.fileSystem,
                                transitEncryption: "ENABLED",
                                authorizationConfig: {
                                    accessPointId: volume.efs.accessPoint,
                                },
                            },
                        };
                    });
                });
            }),
            containerDefinitions: $jsonStringify(containerDefinitions),
        }, { parent: parent }), false)))();
    });
}
var templateObject_1, templateObject_2, templateObject_3, templateObject_4, templateObject_5, templateObject_6;
