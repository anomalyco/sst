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
exports.Version = exports.Component = exports.outputId = void 0;
exports.transform = transform;
exports.$transform = $transform;
exports.$asset = $asset;
exports.$lazy = $lazy;
exports.$print = $print;
exports.parseComponentVersion = parseComponentVersion;
var pulumi_1 = require("@pulumi/pulumi");
var naming_js_1 = require("./naming.js");
var error_js_1 = require("./error.js");
var path_1 = require("path");
var fs_1 = require("fs");
// Previously, `this.api.id` was used as the ID. `this.api.id` was of type Output<string>
// the value evaluates to the mistake id.
// In the future version, we will release a breaking change to fix this.
exports.outputId = "Calling [toString] on an [Output<T>] is not supported.\n\nTo get the value of an Output<T> as an Output<string> consider either:\n1: o.apply(v => `prefix${v}suffix`)\n2: pulumi.interpolate `prefix${v}suffix`\n\nSee https://www.pulumi.com/docs/concepts/inputs-outputs for more details.\nThis function may throw in a future version of @pulumi/pulumi.";
function transform(transform, name, args, opts) {
    // Case: transform is a function
    if (typeof transform === "function") {
        transform(args, opts, name);
        return [name, args, opts];
    }
    // Case: no transform
    // Case: transform is an argument
    return [name, __assign(__assign({}, args), transform), opts];
}
var Component = /** @class */ (function (_super) {
    __extends(Component, _super);
    function Component(type, name, args, opts) {
        var _this = this;
        var _a, _b;
        var transforms = (_a = ComponentTransforms.get(type)) !== null && _a !== void 0 ? _a : [];
        for (var _i = 0, transforms_1 = transforms; _i < transforms_1.length; _i++) {
            var transform_1 = transforms_1[_i];
            transform_1({ name: name, props: args, opts: opts });
        }
        _this = _super.call(this, type, name, args, __assign({ transformations: __spreadArray([
                // Ensure logical and physical names are prefixed
                function (args) {
                    var _a;
                    var _b;
                    // Ensure component names do not contain spaces
                    if (name.includes(" "))
                        throw new Error("Invalid component name \"".concat(name, "\" (").concat(args.type, "). Component names cannot contain spaces."));
                    // Ensure names are prefixed with parent's name
                    if (args.type !== type &&
                        // @ts-expect-error
                        !args.name.startsWith(args.opts.parent.__name)) {
                        throw new Error("In \"".concat(name, "\" component, the logical name of \"").concat(args.name, "\" (").concat(args.type, ") is not prefixed with parent's name ").concat(
                        // @ts-expect-error
                        args.opts.parent.__name));
                    }
                    // Ensure physical names are prefixed with app/stage
                    // note: We are setting the default names here instead of inline when creating
                    //       the resource is b/c the physical name is inferred from the logical name.
                    //       And it's convenient to access the logical name here.
                    if (args.type.startsWith("sst:"))
                        return;
                    if ([
                        // resources manually named
                        "aws:cloudwatch/logGroup:LogGroup",
                        "aws:ecs/service:Service",
                        "aws:ecs/taskDefinition:TaskDefinition",
                        "aws:lb/targetGroup:TargetGroup",
                        "aws:servicediscovery/privateDnsNamespace:PrivateDnsNamespace",
                        "aws:servicediscovery/service:Service",
                        // resources not prefixed
                        "pulumi-nodejs:dynamic:Resource",
                        "random:index/randomId:RandomId",
                        "random:index/randomPassword:RandomPassword",
                        "command:local:Command",
                        "tls:index/privateKey:PrivateKey",
                        "aws:acm/certificate:Certificate",
                        "aws:acm/certificateValidation:CertificateValidation",
                        "aws:apigateway/basePathMapping:BasePathMapping",
                        "aws:apigateway/deployment:Deployment",
                        "aws:apigateway/domainName:DomainName",
                        "aws:apigateway/integration:Integration",
                        "aws:apigateway/integrationResponse:IntegrationResponse",
                        "aws:apigateway/method:Method",
                        "aws:apigateway/methodResponse:MethodResponse",
                        "aws:apigateway/resource:Resource",
                        "aws:apigateway/response:Response",
                        "aws:apigateway/stage:Stage",
                        "aws:apigateway/usagePlanKey:UsagePlanKey",
                        "aws:apigatewayv2/apiMapping:ApiMapping",
                        "aws:apigatewayv2/domainName:DomainName",
                        "aws:apigatewayv2/integration:Integration",
                        "aws:apigatewayv2/route:Route",
                        "aws:apigatewayv2/stage:Stage",
                        "aws:appautoscaling/target:Target",
                        "aws:appsync/dataSource:DataSource",
                        "aws:appsync/domainName:DomainName",
                        "aws:appsync/domainNameApiAssociation:DomainNameApiAssociation",
                        "aws:appsync/function:Function",
                        "aws:appsync/resolver:Resolver",
                        "aws:ec2/routeTableAssociation:RouteTableAssociation",
                        "aws:ecs/clusterCapacityProviders:ClusterCapacityProviders",
                        "aws:efs/fileSystem:FileSystem",
                        "aws:efs/mountTarget:MountTarget",
                        "aws:efs/accessPoint:AccessPoint",
                        "aws:iam/accessKey:AccessKey",
                        "aws:iam/instanceProfile:InstanceProfile",
                        "aws:iam/policy:Policy",
                        "aws:iam/userPolicy:UserPolicy",
                        "aws:cloudfront/cachePolicy:CachePolicy",
                        "aws:cloudfront/distribution:Distribution",
                        "aws:cognito/identityPoolRoleAttachment:IdentityPoolRoleAttachment",
                        "aws:cognito/identityProvider:IdentityProvider",
                        "aws:cognito/userPoolClient:UserPoolClient",
                        "aws:lambda/eventSourceMapping:EventSourceMapping",
                        "aws:lambda/functionEventInvokeConfig:FunctionEventInvokeConfig",
                        "aws:lambda/functionUrl:FunctionUrl",
                        "aws:lambda/invocation:Invocation",
                        "aws:lambda/permission:Permission",
                        "aws:lambda/provisionedConcurrencyConfig:ProvisionedConcurrencyConfig",
                        "aws:lb/listener:Listener",
                        "aws:lb/listenerRule:ListenerRule",
                        "aws:opensearch/domainPolicy:DomainPolicy",
                        "aws:rds/proxyDefaultTargetGroup:ProxyDefaultTargetGroup",
                        "aws:rds/proxyTarget:ProxyTarget",
                        "aws:route53/record:Record",
                        "aws:s3/bucketCorsConfigurationV2:BucketCorsConfigurationV2",
                        "aws:s3/bucketNotification:BucketNotification",
                        "aws:s3/bucketObject:BucketObject",
                        "aws:s3/bucketObjectv2:BucketObjectv2",
                        "aws:s3/bucketPolicy:BucketPolicy",
                        "aws:s3/bucketPublicAccessBlock:BucketPublicAccessBlock",
                        "aws:s3/bucketVersioningV2:BucketVersioningV2",
                        "aws:s3/bucketWebsiteConfigurationV2:BucketWebsiteConfigurationV2",
                        "aws:secretsmanager/secretVersion:SecretVersion",
                        "aws:ses/domainIdentityVerification:DomainIdentityVerification",
                        "aws:sesv2/configurationSetEventDestination:ConfigurationSetEventDestination",
                        "aws:sesv2/emailIdentity:EmailIdentity",
                        "aws:sns/topicPolicy:TopicPolicy",
                        "aws:sns/topicSubscription:TopicSubscription",
                        "aws:sqs/queuePolicy:QueuePolicy",
                        "aws:ssm/parameter:Parameter",
                        "cloudflare:index/record:Record",
                        "cloudflare:index/workerCronTrigger:WorkerCronTrigger",
                        "cloudflare:index/workerDomain:WorkerDomain",
                        "docker-build:index:Image",
                        "vercel:index/dnsRecord:DnsRecord",
                    ].includes(args.type))
                        return;
                    var namingRules = {
                        "aws:apigateway/apiKey:ApiKey": ["name", 1024],
                        "aws:apigateway/authorizer:Authorizer": ["name", 128],
                        "aws:apigateway/restApi:RestApi": ["name", 128],
                        "aws:apigateway/usagePlan:UsagePlan": ["name", 65536], // no length limit
                        "aws:apigatewayv2/api:Api": ["name", 128],
                        "aws:apigatewayv2/authorizer:Authorizer": ["name", 128],
                        "aws:apigatewayv2/vpcLink:VpcLink": ["name", 128],
                        "aws:appautoscaling/policy:Policy": ["name", 255],
                        "aws:appsync/graphQLApi:GraphQLApi": ["name", 65536],
                        "aws:cloudwatch/eventBus:EventBus": ["name", 256],
                        "aws:cloudwatch/eventTarget:EventTarget": ["targetId", 64],
                        "aws:cloudwatch/eventRule:EventRule": ["name", 64],
                        "aws:cloudfront/function:Function": ["name", 64],
                        "aws:cloudfront/keyValueStore:KeyValueStore": ["name", 64],
                        "aws:cognito/identityPool:IdentityPool": ["identityPoolName", 128],
                        "aws:cognito/userPool:UserPool": ["name", 128],
                        "aws:dynamodb/table:Table": ["name", 255],
                        "aws:ec2/keyPair:KeyPair": ["keyName", 255],
                        "aws:ec2/eip:Eip": ["tags", 255],
                        "aws:ec2/instance:Instance": ["tags", 255],
                        "aws:ec2/internetGateway:InternetGateway": ["tags", 255],
                        "aws:ec2/natGateway:NatGateway": ["tags", 255],
                        "aws:ec2/routeTable:RouteTable": ["tags", 255],
                        "aws:ec2/securityGroup:SecurityGroup": ["tags", 255],
                        "aws:ec2/defaultSecurityGroup:DefaultSecurityGroup": ["tags", 255],
                        "aws:ec2/subnet:Subnet": ["tags", 255],
                        "aws:ec2/vpc:Vpc": ["tags", 255],
                        "aws:ecs/cluster:Cluster": ["name", 255],
                        "aws:elasticache/parameterGroup:ParameterGroup": [
                            "name",
                            255,
                            { lower: true },
                        ],
                        "aws:elasticache/replicationGroup:ReplicationGroup": [
                            "replicationGroupId",
                            40,
                            { lower: true },
                        ],
                        "aws:elasticache/subnetGroup:SubnetGroup": [
                            "name",
                            255,
                            { lower: true },
                        ],
                        "aws:iam/role:Role": ["name", 64],
                        "aws:iam/user:User": ["name", 64],
                        "aws:iot/authorizer:Authorizer": ["name", 128],
                        "aws:iot/topicRule:TopicRule": [
                            "name",
                            128,
                            { replace: function (name) { return name.replaceAll("-", "_"); } },
                        ],
                        "aws:kinesis/stream:Stream": ["name", 255],
                        // AWS Load Balancer name allows 32 chars, but an 8 char suffix
                        // ie. "-1234567" is automatically added
                        "aws:lb/loadBalancer:LoadBalancer": ["name", 24],
                        "aws:lambda/function:Function": ["name", 64],
                        "aws:opensearch/domain:Domain": ["domainName", 28, { lower: true }],
                        "aws:rds/cluster:Cluster": [
                            "clusterIdentifier",
                            63,
                            { lower: true },
                        ],
                        "aws:rds/clusterInstance:ClusterInstance": [
                            "identifier",
                            63,
                            { lower: true },
                        ],
                        "aws:rds/instance:Instance": ["identifier", 63, { lower: true }],
                        "aws:rds/proxy:Proxy": ["name", 60, { lower: true }],
                        "aws:rds/clusterParameterGroup:ClusterParameterGroup": [
                            "name",
                            255,
                            { lower: true },
                        ],
                        "aws:rds/parameterGroup:ParameterGroup": [
                            "name",
                            255,
                            { lower: true },
                        ],
                        "aws:rds/subnetGroup:SubnetGroup": ["name", 255, { lower: true }],
                        "aws:s3/bucketV2:BucketV2": ["bucket", 63, { lower: true }],
                        "aws:secretsmanager/secret:Secret": ["name", 512],
                        "aws:sesv2/configurationSet:ConfigurationSet": [
                            "configurationSetName",
                            64,
                            { lower: true },
                        ],
                        "aws:sns/topic:Topic": [
                            "name",
                            256,
                            {
                                suffix: function () {
                                    return (0, pulumi_1.output)(args.props.fifoTopic).apply(function (fifo) {
                                        return fifo ? ".fifo" : "";
                                    });
                                },
                            },
                        ],
                        "aws:sqs/queue:Queue": [
                            "name",
                            80,
                            {
                                suffix: function () {
                                    return (0, pulumi_1.output)(args.props.fifoQueue).apply(function (fifo) {
                                        return fifo ? ".fifo" : "";
                                    });
                                },
                            },
                        ],
                        "cloudflare:index/d1Database:D1Database": [
                            "name",
                            64,
                            { lower: true },
                        ],
                        "cloudflare:index/r2Bucket:R2Bucket": ["name", 64, { lower: true }],
                        "cloudflare:index/workerScript:WorkerScript": [
                            "name",
                            64,
                            { lower: true },
                        ],
                        "cloudflare:index/queue:Queue": ["name", 64, { lower: true }],
                        "cloudflare:index/workersKvNamespace:WorkersKvNamespace": [
                            "title",
                            64,
                            { lower: true },
                        ],
                    };
                    var rule = namingRules[args.type];
                    if (!rule)
                        throw new error_js_1.VisibleError("In \"".concat(name, "\" component, the physical name of \"").concat(args.name, "\" (").concat(args.type, ") is not prefixed"));
                    // name is already set
                    var nameField = rule[0];
                    var length = rule[1];
                    var options = rule[2];
                    if (args.props[nameField] && args.props[nameField] !== "")
                        return;
                    // Handle prefix field is tags
                    if (nameField === "tags") {
                        return {
                            props: __assign(__assign({}, args.props), { tags: __assign(__assign({}, args.tags), { Name: (0, naming_js_1.prefixName)(length, args.name) }) }),
                            opts: args.opts,
                        };
                    }
                    // Handle prefix field is name
                    var suffix = (options === null || options === void 0 ? void 0 : options.suffix) ? options.suffix() : (0, pulumi_1.output)("");
                    return {
                        props: __assign(__assign({}, args.props), (_a = {}, _a[nameField] = suffix.apply(function (suffix) {
                            var v = (options === null || options === void 0 ? void 0 : options.lower)
                                ? (0, naming_js_1.physicalName)(length, args.name, suffix).toLowerCase()
                                : (0, naming_js_1.physicalName)(length, args.name, suffix);
                            if (options === null || options === void 0 ? void 0 : options.replace)
                                v = options.replace(v);
                            return v;
                        }), _a)),
                        opts: __assign(__assign({}, args.opts), { ignoreChanges: __spreadArray(__spreadArray([], ((_b = args.opts.ignoreChanges) !== null && _b !== void 0 ? _b : []), true), [nameField], false) }),
                    };
                },
                // Set child resources `retainOnDelete` if set on component
                function (args) {
                    var _a;
                    return ({
                        props: args.props,
                        opts: __assign(__assign({}, args.opts), { retainOnDelete: (_a = args.opts.retainOnDelete) !== null && _a !== void 0 ? _a : opts === null || opts === void 0 ? void 0 : opts.retainOnDelete }),
                    });
                }
            ], ((_b = opts === null || opts === void 0 ? void 0 : opts.transformations) !== null && _b !== void 0 ? _b : []), true) }, opts)) || this;
        _this.componentType = type;
        _this.componentName = name;
        return _this;
    }
    /** @internal */
    Component.prototype.registerVersion = function (input) {
        var _a, _b;
        // Check component version
        var oldVersion = input.old;
        var newVersion = (_a = input.new) !== null && _a !== void 0 ? _a : 1;
        if (oldVersion) {
            var className = this.componentType.replaceAll(":", ".");
            // Invalid forceUpgrade value
            if (input.forceUpgrade && input.forceUpgrade !== "v".concat(newVersion)) {
                throw new error_js_1.VisibleError([
                    "The value of \"forceUpgrade\" does not match the version of \"".concat(className, "\" component."),
                    "Set \"forceUpgrade\" to \"v".concat(newVersion, "\" to upgrade to the new version."),
                ].join("\n"));
            }
            // Version upgraded without forceUpgrade
            if (oldVersion < newVersion && !input.forceUpgrade) {
                throw new error_js_1.VisibleError((_b = input.message) !== null && _b !== void 0 ? _b : "");
            }
            // Version downgraded
            if (oldVersion > newVersion) {
                throw new error_js_1.VisibleError([
                    "It seems you are trying to use an older version of \"".concat(className, "\"."),
                    "You need to recreate this component to rollback - https://sst.dev/docs/components/#versioning",
                ].join("\n"));
            }
        }
        // Set version
        if (newVersion > 1) {
            new Version(this.componentName, newVersion, { parent: this });
        }
    };
    return Component;
}(pulumi_1.ComponentResource));
exports.Component = Component;
var ComponentTransforms = new Map();
function $transform(resource, cb) {
    // @ts-expect-error
    var type = resource.__pulumiType;
    if (type.startsWith("sst:")) {
        var transforms = ComponentTransforms.get(type);
        if (!transforms) {
            transforms = [];
            ComponentTransforms.set(type, transforms);
        }
        transforms.push(function (input) {
            cb(input.props, input.opts, input.name);
            return input;
        });
        return;
    }
    pulumi_1.runtime.registerStackTransformation(function (input) {
        if (input.type !== type)
            return;
        cb(input.props, input.opts, input.name);
        return input;
    });
}
function $asset(assetPath) {
    var fullPath = path_1.default.isAbsolute(assetPath)
        ? assetPath
        : path_1.default.join($cli.paths.root, assetPath);
    try {
        return (0, fs_1.statSync)(fullPath).isDirectory()
            ? new pulumi_1.asset.FileArchive(fullPath)
            : new pulumi_1.asset.FileAsset(fullPath);
    }
    catch (e) {
        throw new error_js_1.VisibleError("Asset not found: ".concat(fullPath));
    }
}
function $lazy(fn) {
    var _this = this;
    return (0, pulumi_1.output)(undefined)
        .apply(function () { return __awaiter(_this, void 0, void 0, function () { return __generator(this, function (_a) {
        return [2 /*return*/, (0, pulumi_1.output)(fn())];
    }); }); })
        .apply(function (x) { return x; });
}
function $print() {
    var msg = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        msg[_i] = arguments[_i];
    }
    return (0, pulumi_1.all)(msg).apply(function (msg) { return console.log.apply(console, msg); });
}
var Version = /** @class */ (function (_super) {
    __extends(Version, _super);
    function Version(target, version, opts) {
        var _this = _super.call(this, "sst:sst:Version", target + "Version", {}, opts) || this;
        _this.registerOutputs({ target: target, version: version });
        return _this;
    }
    return Version;
}(pulumi_1.ComponentResource));
exports.Version = Version;
function parseComponentVersion(version) {
    var _a = version.split("."), major = _a[0], minor = _a[1];
    return { major: parseInt(major), minor: parseInt(minor) };
}
