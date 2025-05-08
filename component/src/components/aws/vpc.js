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
exports.Vpc = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var vpc_v1_1 = require("./vpc-v1");
var error_1 = require("../error");
var tls_1 = require("@pulumi/tls");
/**
 * The `Vpc` component lets you add a VPC to your app. It uses [Amazon VPC](https://docs.aws.amazon.com/vpc/). This is useful for services like RDS and Fargate that need to be hosted inside
 * a VPC.
 *
 * This creates a VPC with 2 Availability Zones by default. It also creates the following
 * resources:
 *
 * 1. A default security group blocking all incoming internet traffic.
 * 2. A public subnet in each AZ.
 * 3. A private subnet in each AZ.
 * 4. An Internet Gateway. All the traffic from the public subnets are routed through it.
 * 5. If `nat` is enabled, a NAT Gateway or NAT instance in each AZ. All the traffic from
 *    the private subnets are routed to the NAT in the same AZ.
 *
 * :::note
 * By default, this does not create NAT Gateways or NAT instances.
 * :::
 *
 * @example
 *
 * #### Create a VPC
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Vpc("MyVPC");
 * ```
 *
 * #### Create it with 3 Availability Zones
 *
 * ```ts title="sst.config.ts" {2}
 * new sst.aws.Vpc("MyVPC", {
 *   az: 3
 * });
 * ```
 *
 * #### Enable NAT
 *
 * ```ts title="sst.config.ts" {2}
 * new sst.aws.Vpc("MyVPC", {
 *   nat: "managed"
 * });
 * ```
 *
 * ---
 *
 * ### Cost
 *
 * By default, this component is **free**. Following is the cost to enable the `nat` or `bastion`
 * options.
 *
 * #### Managed NAT
 *
 * If you enable `nat` with the `managed` option, it uses a _NAT Gateway_ per `az` at $0.045 per
 * hour, and $0.045 per GB processed per month.
 *
 * That works out to a minimum of $0.045 x 2 x 24 x 30 or **$65 per month**. Adjust this for the
 * number of `az` and add $0.045 per GB processed per month.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [NAT Gateway pricing](https://aws.amazon.com/vpc/pricing/) for more details. Standard [data
 * transfer charges](https://aws.amazon.com/ec2/pricing/on-demand/#Data_Transfer) apply.
 *
 * #### EC2 NAT
 *
 * If you enable `nat` with the `ec2` option, it uses `t4g.nano` EC2 _On Demand_ instances per
 * `az` at $0.0042 per hour, and $0.09 per GB processed per month for the first 10TB.
 *
 * That works out to a minimum of $0.0042 x 2 x 24 x 30 or **$6 per month**. Adjust this for the
 * `nat.ec2.instance` you are using and add $0.09 per GB processed per month.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [EC2 On-Demand pricing](https://aws.amazon.com/vpc/pricing/) and the
 * [EC2 Data Transfer pricing](https://aws.amazon.com/ec2/pricing/on-demand/#Data_Transfer)
 * for more details.
 *
 * #### Bastion
 *
 * If you enable `bastion`, it uses a single `t4g.nano` EC2 _On Demand_ instance at
 * $0.0042 per hour, and $0.09 per GB processed per month for the first 10TB.
 *
 * That works out to $0.0042 x 24 x 30 or **$3 per month**. Add $0.09 per GB processed per month.
 *
 * However if `nat: "ec2"` is enabled, one of the NAT EC2 instances will be reused; making this
 * **free**.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [EC2 On-Demand pricing](https://aws.amazon.com/vpc/pricing/) and the
 * [EC2 Data Transfer pricing](https://aws.amazon.com/ec2/pricing/on-demand/#Data_Transfer)
 * for more details.
 */
var Vpc = /** @class */ (function (_super) {
    __extends(Vpc, _super);
    function Vpc(name, args, opts) {
        if (args === void 0) { args = {}; }
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var _version = 2;
        var _refVersion = 2;
        var self = _this;
        if (args && "ref" in args) {
            var ref = reference();
            _this.vpc = ref.vpc;
            _this.internetGateway = ref.internetGateway;
            _this.securityGroup = ref.securityGroup;
            _this._publicSubnets = (0, pulumi_1.output)(ref.publicSubnets);
            _this._privateSubnets = (0, pulumi_1.output)(ref.privateSubnets);
            _this.publicRouteTables = (0, pulumi_1.output)(ref.publicRouteTables);
            _this.privateRouteTables = (0, pulumi_1.output)(ref.privateRouteTables);
            _this.natGateways = (0, pulumi_1.output)(ref.natGateways);
            _this.natInstances = (0, pulumi_1.output)(ref.natInstances);
            _this.elasticIps = ref.elasticIps;
            _this.bastionInstance = ref.bastionInstance;
            _this.cloudmapNamespace = ref.cloudmapNamespace;
            _this.privateKeyValue = (0, pulumi_1.output)(ref.privateKeyValue);
            registerOutputs();
            return _this;
        }
        registerVersion();
        var zones = normalizeAz();
        var nat = normalizeNat();
        var partition = (0, aws_1.getPartitionOutput)({}, opts).partition;
        var vpc = createVpc();
        var _a = createKeyPair(), keyPair = _a.keyPair, privateKeyValue = _a.privateKeyValue;
        var internetGateway = createInternetGateway();
        var securityGroup = createSecurityGroup();
        var _b = createPublicSubnets(), publicSubnets = _b.publicSubnets, publicRouteTables = _b.publicRouteTables;
        var elasticIps = createElasticIps();
        var natGateways = createNatGateways();
        var natInstances = createNatInstances();
        var _c = createPrivateSubnets(), privateSubnets = _c.privateSubnets, privateRouteTables = _c.privateRouteTables;
        var bastionInstance = createBastion();
        var cloudmapNamespace = createCloudmapNamespace();
        _this.vpc = vpc;
        _this.internetGateway = internetGateway;
        _this.securityGroup = securityGroup;
        _this.natGateways = natGateways;
        _this.natInstances = natInstances;
        _this.elasticIps = elasticIps;
        _this._publicSubnets = publicSubnets;
        _this._privateSubnets = privateSubnets;
        _this.publicRouteTables = publicRouteTables;
        _this.privateRouteTables = privateRouteTables;
        _this.bastionInstance = (0, pulumi_1.output)(bastionInstance);
        _this.cloudmapNamespace = cloudmapNamespace;
        _this.privateKeyValue = (0, pulumi_1.output)(privateKeyValue);
        registerOutputs();
        function reference() {
            var ref = args;
            var vpc = aws_1.ec2.Vpc.get("".concat(name, "Vpc"), ref.vpcId, undefined, {
                parent: self,
            });
            var vpcId = vpc.tags.apply(function (tags) {
                registerVersion((tags === null || tags === void 0 ? void 0 : tags["sst:component-version"])
                    ? parseInt(tags["sst:component-version"])
                    : undefined);
                if ((tags === null || tags === void 0 ? void 0 : tags["sst:ref-version"]) !== _refVersion.toString()) {
                    throw new error_1.VisibleError([
                        "There have been some minor changes to the \"Vpc\" component that's being referenced by \"".concat(name, "\".\n"),
                        "To update, you'll need to redeploy the stage where the VPC was created. And then redeploy this stage.",
                    ].join("\n"));
                }
                return (0, pulumi_1.output)(ref.vpcId);
            });
            var internetGateway = aws_1.ec2.InternetGateway.get("".concat(name, "InstanceGateway"), aws_1.ec2.getInternetGatewayOutput({
                filters: [{ name: "attachment.vpc-id", values: [vpcId] }],
            }, { parent: self }).internetGatewayId, undefined, { parent: self });
            var securityGroup = aws_1.ec2.SecurityGroup.get("".concat(name, "SecurityGroup"), aws_1.ec2
                .getSecurityGroupsOutput({
                filters: [
                    { name: "group-name", values: ["default"] },
                    { name: "vpc-id", values: [vpcId] },
                ],
            }, { parent: self })
                .ids.apply(function (ids) {
                if (!ids.length) {
                    throw new error_1.VisibleError("Security group not found in VPC ".concat(vpcId));
                }
                return ids[0];
            }), undefined, { parent: self });
            var privateSubnets = aws_1.ec2
                .getSubnetsOutput({
                filters: [
                    { name: "vpc-id", values: [vpcId] },
                    { name: "tag:Name", values: ["*Private*"] },
                ],
            }, { parent: self })
                .ids.apply(function (ids) {
                return ids.map(function (id, i) {
                    return aws_1.ec2.Subnet.get("".concat(name, "PrivateSubnet").concat(i + 1), id, undefined, {
                        parent: self,
                    });
                });
            });
            var privateRouteTables = privateSubnets.apply(function (subnets) {
                return subnets.map(function (subnet, i) {
                    return aws_1.ec2.RouteTable.get("".concat(name, "PrivateRouteTable").concat(i + 1), aws_1.ec2.getRouteTableOutput({ subnetId: subnet.id }, { parent: self })
                        .routeTableId, undefined, { parent: self });
                });
            });
            var publicSubnets = aws_1.ec2
                .getSubnetsOutput({
                filters: [
                    { name: "vpc-id", values: [vpcId] },
                    { name: "tag:Name", values: ["*Public*"] },
                ],
            }, { parent: self })
                .ids.apply(function (ids) {
                return ids.map(function (id, i) {
                    return aws_1.ec2.Subnet.get("".concat(name, "PublicSubnet").concat(i + 1), id, undefined, {
                        parent: self,
                    });
                });
            });
            var publicRouteTables = publicSubnets.apply(function (subnets) {
                return subnets.map(function (subnet, i) {
                    return aws_1.ec2.RouteTable.get("".concat(name, "PublicRouteTable").concat(i + 1), aws_1.ec2.getRouteTableOutput({ subnetId: subnet.id }, { parent: self })
                        .routeTableId, undefined, { parent: self });
                });
            });
            var natGateways = publicSubnets.apply(function (subnets) {
                var natGatewayIds = subnets.map(function (subnet, i) {
                    return aws_1.ec2
                        .getNatGatewaysOutput({
                        filters: [
                            { name: "subnet-id", values: [subnet.id] },
                            { name: "state", values: ["available"] },
                        ],
                    }, { parent: self })
                        .ids.apply(function (ids) { return ids[0]; });
                });
                return (0, pulumi_1.output)(natGatewayIds).apply(function (ids) {
                    return ids
                        .filter(function (id) { return id; })
                        .map(function (id, i) {
                        return aws_1.ec2.NatGateway.get("".concat(name, "NatGateway").concat(i + 1), id, undefined, {
                            parent: self,
                        });
                    });
                });
            });
            var elasticIps = natGateways.apply(function (nats) {
                return nats.map(function (nat, i) {
                    return aws_1.ec2.Eip.get("".concat(name, "ElasticIp").concat(i + 1), nat.allocationId, undefined, { parent: self });
                });
            });
            var natInstances = aws_1.ec2
                .getInstancesOutput({
                filters: [
                    { name: "tag:sst:is-nat", values: ["true"] },
                    { name: "vpc-id", values: [vpcId] },
                ],
            }, { parent: self })
                .ids.apply(function (ids) {
                return ids.map(function (id, i) {
                    return aws_1.ec2.Instance.get("".concat(name, "NatInstance").concat(i + 1), id, undefined, {
                        parent: self,
                    });
                });
            });
            var bastionInstance = aws_1.ec2
                .getInstancesOutput({
                filters: [
                    { name: "tag:sst:is-bastion", values: ["true"] },
                    { name: "vpc-id", values: [vpcId] },
                ],
            }, { parent: self })
                .ids.apply(function (ids) {
                return ids.length
                    ? aws_1.ec2.Instance.get("".concat(name, "BastionInstance"), ids[0], undefined, {
                        parent: self,
                    })
                    : undefined;
            });
            // Note: can also use servicediscovery.getDnsNamespaceOutput() here, ie.
            // ```ts
            // const namespaceId = servicediscovery.getDnsNamespaceOutput({
            //   name: "sst",
            //   type: "DNS_PRIVATE",
            // }).id;
            // ```
            // but if user deployed multiple VPCs into the same account. This will error because
            // there are multiple results. Even though `getDnsNamespaceOutput()` takes tags in args,
            // the tags are not used for lookup.
            var zone = (0, pulumi_1.output)(vpcId).apply(function (vpcId) {
                return aws_1.route53.getZone({
                    name: "sst",
                    privateZone: true,
                    vpcId: vpcId,
                }, { parent: self });
            });
            var namespaceId = zone.linkedServiceDescription.apply(function (description) {
                var _a;
                var match = (_a = description.match(/:namespace\/(ns-[a-z1-9]*)/)) === null || _a === void 0 ? void 0 : _a[1];
                if (!match) {
                    throw new error_1.VisibleError("Cloud Map namespace not found for VPC ".concat(vpcId));
                }
                return match;
            });
            var cloudmapNamespace = aws_1.servicediscovery.PrivateDnsNamespace.get("".concat(name, "CloudmapNamespace"), namespaceId, { vpc: vpcId }, { parent: self });
            var privateKeyValue = bastionInstance.apply(function (v) {
                if (!v)
                    return;
                var param = aws_1.ssm.Parameter.get("".concat(name, "PrivateKeyValue"), (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["/sst/vpc/", "/private-key-value"], ["/sst/vpc/", "/private-key-value"])), vpcId), undefined, { parent: self });
                return param.value;
            });
            return {
                vpc: vpc,
                internetGateway: internetGateway,
                securityGroup: securityGroup,
                publicSubnets: publicSubnets,
                publicRouteTables: publicRouteTables,
                privateSubnets: privateSubnets,
                privateRouteTables: privateRouteTables,
                natGateways: natGateways,
                natInstances: natInstances,
                elasticIps: elasticIps,
                bastionInstance: bastionInstance,
                cloudmapNamespace: cloudmapNamespace,
                privateKeyValue: privateKeyValue,
            };
        }
        function registerVersion(overrideVersion) {
            self.registerVersion({
                new: _version,
                old: overrideVersion !== null && overrideVersion !== void 0 ? overrideVersion : $cli.state.version[name],
                message: [
                    "There is a new version of \"Vpc\" that has breaking changes.",
                    "",
                    "To continue using the previous version, rename \"Vpc\" to \"Vpc.v".concat($cli.state.version[name], "\". Or recreate this component to update - https://sst.dev/docs/components/#versioning"),
                ].join("\n"),
            });
        }
        function registerOutputs() {
            self.registerOutputs({
                _tunnel: (0, pulumi_1.all)([
                    self.bastionInstance,
                    self.privateKeyValue,
                    self._privateSubnets,
                    self._publicSubnets,
                ]).apply(function (_a) {
                    var bastion = _a[0], privateKeyValue = _a[1], privateSubnets = _a[2], publicSubnets = _a[3];
                    if (!bastion)
                        return;
                    return {
                        ip: bastion.publicIp,
                        username: "ec2-user",
                        privateKey: privateKeyValue,
                        subnets: __spreadArray(__spreadArray([], privateSubnets, true), publicSubnets, true).map(function (s) { return s.cidrBlock; }),
                    };
                }),
            });
        }
        function normalizeAz() {
            return (0, pulumi_1.output)(args.az).apply(function (az) {
                var _a;
                if (Array.isArray(az))
                    return (0, pulumi_1.output)(az);
                var zones = (0, aws_1.getAvailabilityZonesOutput)({
                    state: "available",
                }, { parent: self });
                return (0, pulumi_1.all)([zones, (_a = args.az) !== null && _a !== void 0 ? _a : 2]).apply(function (_a) {
                    var zones = _a[0], az = _a[1];
                    return Array(az)
                        .fill(0)
                        .map(function (_, i) { return zones.names[i]; });
                });
            });
        }
        function normalizeNat() {
            return (0, pulumi_1.all)([args.nat, zones]).apply(function (_a) {
                var _b;
                var nat = _a[0], zones = _a[1];
                if (nat === "managed") {
                    return { type: "managed" };
                }
                if (nat === "ec2") {
                    return {
                        type: "ec2",
                        ec2: { instance: "t4g.nano", ami: undefined },
                    };
                }
                if (nat) {
                    if (nat.ec2 && nat.type === "managed")
                        throw new error_1.VisibleError("\"nat.type\" cannot be \"managed\" when \"nat.ec2\" is specified");
                    if (!nat.type)
                        throw new error_1.VisibleError("Missing \"nat.type\" for the \"".concat(name, "\" VPC. It is required when \"nat.ec2\" is not specified"));
                    if (nat.ip && nat.ip.length !== zones.length)
                        throw new error_1.VisibleError("The number of Elastic IP allocation IDs must match the number of AZs.");
                    return nat.ec2 || nat.type === "ec2"
                        ? {
                            type: "ec2",
                            ip: nat.ip,
                            ec2: (_b = nat.ec2) !== null && _b !== void 0 ? _b : { instance: "t4g.nano" },
                        }
                        : {
                            type: "managed",
                            ip: nat.ip,
                        };
                }
                return undefined;
            });
        }
        function createVpc() {
            var _a;
            var _b;
            return new ((_a = aws_1.ec2.Vpc).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.vpc, "".concat(name, "Vpc"), {
                cidrBlock: "10.0.0.0/16",
                enableDnsSupport: true,
                enableDnsHostnames: true,
                tags: {
                    Name: "".concat($app.name, "-").concat($app.stage, "-").concat(name, " VPC"),
                    "sst:component-version": _version.toString(),
                    "sst:ref-version": _refVersion.toString(),
                },
            }, { parent: self }), false)))();
        }
        function createKeyPair() {
            var ret = (0, pulumi_1.output)(args.bastion).apply(function (bastion) {
                if (!bastion)
                    return {};
                var tlsPrivateKey = new tls_1.PrivateKey("".concat(name, "TlsPrivateKey"), {
                    algorithm: "RSA",
                    rsaBits: 4096,
                }, { parent: self });
                new aws_1.ssm.Parameter("".concat(name, "PrivateKeyValue"), {
                    name: (0, pulumi_1.interpolate)(templateObject_2 || (templateObject_2 = __makeTemplateObject(["/sst/vpc/", "/private-key-value"], ["/sst/vpc/", "/private-key-value"])), vpc.id),
                    description: "Bastion host private key",
                    type: aws_1.ssm.ParameterType.SecureString,
                    value: tlsPrivateKey.privateKeyOpenssh,
                }, { parent: self });
                var keyPair = new aws_1.ec2.KeyPair("".concat(name, "KeyPair"), {
                    publicKey: tlsPrivateKey.publicKeyOpenssh,
                }, { parent: self });
                return { keyPair: keyPair, privateKeyValue: tlsPrivateKey.privateKeyOpenssh };
            });
            return {
                keyPair: (0, pulumi_1.output)(ret.keyPair),
                privateKeyValue: (0, pulumi_1.output)(ret.privateKeyValue),
            };
        }
        function createInternetGateway() {
            var _a;
            var _b;
            return new ((_a = aws_1.ec2.InternetGateway).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.internetGateway, "".concat(name, "InternetGateway"), {
                vpcId: vpc.id,
            }, { parent: self }), false)))();
        }
        function createSecurityGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.ec2.DefaultSecurityGroup).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.securityGroup, "".concat(name, "SecurityGroup"), {
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
                        // Restricts inbound traffic to only within the VPC
                        cidrBlocks: [vpc.cidrBlock],
                    },
                ],
            }, { parent: self }), false)))();
        }
        function createElasticIps() {
            return (0, pulumi_1.all)([nat, publicSubnets]).apply(function (_a) {
                var nat = _a[0], subnets = _a[1];
                if (!nat)
                    return [];
                if (nat === null || nat === void 0 ? void 0 : nat.ip)
                    return [];
                return subnets.map(function (_, i) {
                    var _a;
                    var _b;
                    return new ((_a = aws_1.ec2.Eip).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.elasticIp, "".concat(name, "ElasticIp").concat(i + 1), {
                        vpc: true,
                    }, { parent: self }), false)))();
                });
            });
        }
        function createNatGateways() {
            return (0, pulumi_1.all)([nat, publicSubnets, elasticIps]).apply(function (_a) {
                var nat = _a[0], subnets = _a[1], elasticIps = _a[2];
                if ((nat === null || nat === void 0 ? void 0 : nat.type) !== "managed")
                    return [];
                return subnets.map(function (subnet, i) {
                    var _a;
                    var _b, _c, _d;
                    return new ((_a = aws_1.ec2.NatGateway).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.natGateway, "".concat(name, "NatGateway").concat(i + 1), {
                        subnetId: subnet.id,
                        allocationId: (_d = (_c = elasticIps[i]) === null || _c === void 0 ? void 0 : _c.id) !== null && _d !== void 0 ? _d : nat.ip[i],
                    }, { parent: self }), false)))();
                });
            });
        }
        function createNatInstances() {
            return nat.apply(function (nat) {
                var _a;
                var _b, _c;
                if ((nat === null || nat === void 0 ? void 0 : nat.type) !== "ec2")
                    return (0, pulumi_1.output)([]);
                var sg = new ((_a = aws_1.ec2.SecurityGroup).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.natSecurityGroup, "".concat(name, "NatInstanceSecurityGroup"), {
                    vpcId: vpc.id,
                    ingress: [
                        {
                            protocol: "-1",
                            fromPort: 0,
                            toPort: 0,
                            cidrBlocks: ["0.0.0.0/0"],
                        },
                    ],
                    egress: [
                        {
                            protocol: "-1",
                            fromPort: 0,
                            toPort: 0,
                            cidrBlocks: ["0.0.0.0/0"],
                        },
                    ],
                }, { parent: self }), false)))();
                var role = new aws_1.iam.Role("".concat(name, "NatInstanceRole"), {
                    assumeRolePolicy: aws_1.iam.getPolicyDocumentOutput({
                        statements: [
                            {
                                actions: ["sts:AssumeRole"],
                                principals: [
                                    {
                                        type: "Service",
                                        identifiers: ["ec2.amazonaws.com"],
                                    },
                                ],
                            },
                        ],
                    }).json,
                    managedPolicyArns: [
                        (0, pulumi_1.interpolate)(templateObject_3 || (templateObject_3 = __makeTemplateObject(["arn:", ":iam::aws:policy/AmazonSSMManagedInstanceCore"], ["arn:", ":iam::aws:policy/AmazonSSMManagedInstanceCore"])), partition),
                    ],
                }, { parent: self });
                var instanceProfile = new aws_1.iam.InstanceProfile("".concat(name, "NatInstanceProfile"), { role: role.name }, { parent: self });
                var ami = (_c = nat.ec2.ami) !== null && _c !== void 0 ? _c : aws_1.ec2.getAmiOutput({
                    owners: ["568608671756"], // AWS account ID for fck-nat AMI
                    filters: [
                        {
                            name: "name",
                            // The AMI has the SSM agent pre-installed
                            values: ["fck-nat-al2023-*"],
                        },
                        {
                            name: "architecture",
                            values: ["arm64"],
                        },
                    ],
                    mostRecent: true,
                }, { parent: self }).id;
                return (0, pulumi_1.all)([
                    zones,
                    publicSubnets,
                    elasticIps,
                    keyPair,
                    args.bastion,
                ]).apply(function (_a) {
                    var zones = _a[0], publicSubnets = _a[1], elasticIps = _a[2], keyPair = _a[3], bastion = _a[4];
                    return zones.map(function (_, i) {
                        var _a;
                        var _b, _c, _d;
                        var instance = new ((_a = aws_1.ec2.Instance).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.natInstance, "".concat(name, "NatInstance").concat(i + 1), {
                            instanceType: nat.ec2.instance,
                            ami: ami,
                            subnetId: publicSubnets[i].id,
                            vpcSecurityGroupIds: [sg.id],
                            iamInstanceProfile: instanceProfile.name,
                            sourceDestCheck: false,
                            keyName: keyPair === null || keyPair === void 0 ? void 0 : keyPair.keyName,
                            tags: __assign({ Name: "".concat(name, " NAT Instance"), "sst:is-nat": "true" }, (bastion && i === 0 ? { "sst:is-bastion": "true" } : {})),
                        }, { parent: self }), false)))();
                        new aws_1.ec2.EipAssociation("".concat(name, "NatInstanceEipAssociation").concat(i + 1), {
                            instanceId: instance.id,
                            allocationId: (_d = (_c = elasticIps[i]) === null || _c === void 0 ? void 0 : _c.id) !== null && _d !== void 0 ? _d : nat.ip[i],
                        });
                        return instance;
                    });
                });
            });
        }
        function createPublicSubnets() {
            var ret = zones.apply(function (zones) {
                return zones.map(function (zone, i) {
                    var _a, _b;
                    var _c, _d;
                    var subnet = new ((_a = aws_1.ec2.Subnet).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.publicSubnet, "".concat(name, "PublicSubnet").concat(i + 1), {
                        vpcId: vpc.id,
                        cidrBlock: "10.0.".concat(8 * i, ".0/22"),
                        availabilityZone: zone,
                        mapPublicIpOnLaunch: true,
                    }, { parent: self }), false)))();
                    var routeTable = new ((_b = aws_1.ec2.RouteTable).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_d = args.transform) === null || _d === void 0 ? void 0 : _d.publicRouteTable, "".concat(name, "PublicRouteTable").concat(i + 1), {
                        vpcId: vpc.id,
                        routes: [
                            {
                                cidrBlock: "0.0.0.0/0",
                                gatewayId: internetGateway.id,
                            },
                        ],
                    }, { parent: self }), false)))();
                    new aws_1.ec2.RouteTableAssociation("".concat(name, "PublicRouteTableAssociation").concat(i + 1), {
                        subnetId: subnet.id,
                        routeTableId: routeTable.id,
                    }, { parent: self });
                    return { subnet: subnet, routeTable: routeTable };
                });
            });
            return {
                publicSubnets: ret.apply(function (ret) { return ret.map(function (r) { return r.subnet; }); }),
                publicRouteTables: ret.apply(function (ret) { return ret.map(function (r) { return r.routeTable; }); }),
            };
        }
        function createPrivateSubnets() {
            var ret = zones.apply(function (zones) {
                return zones.map(function (zone, i) {
                    var _a, _b;
                    var _c, _d;
                    var subnet = new ((_a = aws_1.ec2.Subnet).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_c = args.transform) === null || _c === void 0 ? void 0 : _c.privateSubnet, "".concat(name, "PrivateSubnet").concat(i + 1), {
                        vpcId: vpc.id,
                        cidrBlock: "10.0.".concat(8 * i + 4, ".0/22"),
                        availabilityZone: zone,
                    }, { parent: self }), false)))();
                    var routeTable = new ((_b = aws_1.ec2.RouteTable).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_d = args.transform) === null || _d === void 0 ? void 0 : _d.privateRouteTable, "".concat(name, "PrivateRouteTable").concat(i + 1), {
                        vpcId: vpc.id,
                        routes: (0, pulumi_1.all)([natGateways, natInstances]).apply(function (_a) {
                            var natGateways = _a[0], natInstances = _a[1];
                            return __spreadArray(__spreadArray([], (natGateways[i]
                                ? [
                                    {
                                        cidrBlock: "0.0.0.0/0",
                                        natGatewayId: natGateways[i].id,
                                    },
                                ]
                                : []), true), (natInstances[i]
                                ? [
                                    {
                                        cidrBlock: "0.0.0.0/0",
                                        networkInterfaceId: natInstances[i].primaryNetworkInterfaceId,
                                    },
                                ]
                                : []), true);
                        }),
                    }, { parent: self }), false)))();
                    new aws_1.ec2.RouteTableAssociation("".concat(name, "PrivateRouteTableAssociation").concat(i + 1), {
                        subnetId: subnet.id,
                        routeTableId: routeTable.id,
                    }, { parent: self });
                    return { subnet: subnet, routeTable: routeTable };
                });
            });
            return {
                privateSubnets: ret.apply(function (ret) { return ret.map(function (r) { return r.subnet; }); }),
                privateRouteTables: ret.apply(function (ret) { return ret.map(function (r) { return r.routeTable; }); }),
            };
        }
        function createBastion() {
            return (0, pulumi_1.all)([args.bastion, natInstances, keyPair]).apply(function (_a) {
                var _b, _c;
                var _d, _e;
                var bastion = _a[0], natInstances = _a[1], keyPair = _a[2];
                if (!bastion)
                    return undefined;
                if (natInstances.length)
                    return natInstances[0];
                var sg = new ((_b = aws_1.ec2.SecurityGroup).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_d = args.transform) === null || _d === void 0 ? void 0 : _d.bastionSecurityGroup, "".concat(name, "BastionSecurityGroup"), {
                    vpcId: vpc.id,
                    ingress: [
                        {
                            protocol: "tcp",
                            fromPort: 22,
                            toPort: 22,
                            cidrBlocks: ["0.0.0.0/0"],
                        },
                    ],
                    egress: [
                        {
                            protocol: "-1",
                            fromPort: 0,
                            toPort: 0,
                            cidrBlocks: ["0.0.0.0/0"],
                        },
                    ],
                }, { parent: self }), false)))();
                var role = new aws_1.iam.Role("".concat(name, "BastionRole"), {
                    assumeRolePolicy: aws_1.iam.getPolicyDocumentOutput({
                        statements: [
                            {
                                actions: ["sts:AssumeRole"],
                                principals: [
                                    {
                                        type: "Service",
                                        identifiers: ["ec2.amazonaws.com"],
                                    },
                                ],
                            },
                        ],
                    }).json,
                    managedPolicyArns: [
                        (0, pulumi_1.interpolate)(templateObject_4 || (templateObject_4 = __makeTemplateObject(["arn:", ":iam::aws:policy/AmazonSSMManagedInstanceCore"], ["arn:", ":iam::aws:policy/AmazonSSMManagedInstanceCore"])), partition),
                    ],
                }, { parent: self });
                var instanceProfile = new aws_1.iam.InstanceProfile("".concat(name, "BastionProfile"), { role: role.name }, { parent: self });
                var ami = aws_1.ec2.getAmiOutput({
                    owners: ["amazon"],
                    filters: [
                        {
                            name: "name",
                            // The AMI has the SSM agent pre-installed
                            values: ["al2023-ami-20*"],
                        },
                        {
                            name: "architecture",
                            values: ["arm64"],
                        },
                    ],
                    mostRecent: true,
                }, { parent: self });
                return new ((_c = aws_1.ec2.Instance).bind.apply(_c, __spreadArray([void 0], (0, component_1.transform)((_e = args.transform) === null || _e === void 0 ? void 0 : _e.bastionInstance, "".concat(name, "BastionInstance"), {
                    instanceType: "t4g.nano",
                    ami: ami.id,
                    subnetId: publicSubnets.apply(function (v) { return v[0].id; }),
                    vpcSecurityGroupIds: [sg.id],
                    iamInstanceProfile: instanceProfile.name,
                    keyName: keyPair === null || keyPair === void 0 ? void 0 : keyPair.keyName,
                    tags: {
                        "sst:is-bastion": "true",
                    },
                }, { parent: self }), false)))();
            });
        }
        function createCloudmapNamespace() {
            return new aws_1.servicediscovery.PrivateDnsNamespace("".concat(name, "CloudmapNamespace"), {
                name: "sst",
                vpc: vpc.id,
            }, { parent: self });
        }
        return _this;
    }
    Object.defineProperty(Vpc.prototype, "id", {
        /**
         * The VPC ID.
         */
        get: function () {
            return this.vpc.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Vpc.prototype, "publicSubnets", {
        /**
         * A list of public subnet IDs in the VPC.
         */
        get: function () {
            return this._publicSubnets.apply(function (subnets) {
                return subnets.map(function (subnet) { return subnet.id; });
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Vpc.prototype, "privateSubnets", {
        /**
         * A list of private subnet IDs in the VPC.
         */
        get: function () {
            return this._privateSubnets.apply(function (subnets) {
                return subnets.map(function (subnet) { return subnet.id; });
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Vpc.prototype, "securityGroups", {
        /**
         * A list of VPC security group IDs.
         */
        get: function () {
            return (0, pulumi_1.output)(this.securityGroup).apply(function (v) { return [v.id]; });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Vpc.prototype, "bastion", {
        /**
         * The bastion instance ID.
         */
        get: function () {
            return this.bastionInstance.apply(function (v) {
                if (!v) {
                    throw new error_1.VisibleError("VPC bastion is not enabled. Enable it with \"bastion: true\".");
                }
                return v.id;
            });
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Vpc.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon EC2 VPC.
                 */
                vpc: this.vpc,
                /**
                 * The Amazon EC2 Internet Gateway.
                 */
                internetGateway: this.internetGateway,
                /**
                 * The Amazon EC2 Security Group.
                 */
                securityGroup: this.securityGroup,
                /**
                 * The Amazon EC2 NAT Gateway.
                 */
                natGateways: this.natGateways,
                /**
                 * The Amazon EC2 NAT instances.
                 */
                natInstances: this.natInstances,
                /**
                 * The Amazon EC2 Elastic IP.
                 */
                elasticIps: this.elasticIps,
                /**
                 * The Amazon EC2 public subnet.
                 */
                publicSubnets: this._publicSubnets,
                /**
                 * The Amazon EC2 private subnet.
                 */
                privateSubnets: this._privateSubnets,
                /**
                 * The Amazon EC2 route table for the public subnet.
                 */
                publicRouteTables: this.publicRouteTables,
                /**
                 * The Amazon EC2 route table for the private subnet.
                 */
                privateRouteTables: this.privateRouteTables,
                /**
                 * The Amazon EC2 bastion instance.
                 */
                bastionInstance: this.bastionInstance,
                /**
                 * The AWS Cloudmap namespace.
                 */
                cloudmapNamespace: this.cloudmapNamespace,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Reference an existing VPC with the given ID. This is useful when you
     * create a VPC in one stage and want to share it in another stage. It avoids having to
     * create a new VPC in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share VPCs across stages.
     * :::
     *
     * @param name The name of the component.
     * @param vpcId The ID of the existing VPC.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a VPC in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new VPC, you want to share the VPC from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const vpc = $app.stage === "frank"
     *   ? sst.aws.Vpc.get("MyVPC", "vpc-0be8fa4de860618bb")
     *   : new sst.aws.Vpc("MyVPC");
     * ```
     *
     * Here `vpc-0be8fa4de860618bb` is the ID of the VPC created in the `dev` stage.
     * You can find this by outputting the VPC ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   vpc: vpc.id
     * };
     * ```
     */
    Vpc.get = function (name, vpcId, opts) {
        return new Vpc(name, {
            ref: true,
            vpcId: vpcId,
        }, opts);
    };
    /** @internal */
    Vpc.prototype.getSSTLink = function () {
        return {
            properties: {
                bastion: this.bastionInstance.apply(function (v) { return v === null || v === void 0 ? void 0 : v.id; }),
            },
        };
    };
    Vpc.v1 = vpc_v1_1.Vpc;
    return Vpc;
}(component_1.Component));
exports.Vpc = Vpc;
var __pulumiType = "sst:aws:Vpc";
// @ts-expect-error
Vpc.__pulumiType = __pulumiType;
var templateObject_1, templateObject_2, templateObject_3, templateObject_4;
