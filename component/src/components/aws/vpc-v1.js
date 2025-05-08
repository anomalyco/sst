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
/**
 * The `Vpc` component lets you add a VPC to your app, but it has been deprecated because
 * it does not support modifying the number of Availability Zones (AZs) after VPC creation.
 *
 * For existing usage, rename `sst.aws.Vpc` to `sst.aws.Vpc.v1`. For new VPCs, use
 * the latest [`Vpc`](/docs/component/aws/vpc) component instead.
 *
 * :::caution
 * This component has been deprecated.
 * :::
 *
 * This creates a VPC with 2 Availability Zones by default. It also creates the following
 * resources:
 *
 * 1. A security group.
 * 2. A public subnet in each AZ.
 * 3. A private subnet in each AZ.
 * 4. An Internet Gateway, all the traffic from the public subnets are routed through it.
 * 5. A NAT Gateway in each AZ. All the traffic from the private subnets are routed to the
 *    NAT Gateway in the same AZ.
 *
 * :::note
 * By default, this creates two NAT Gateways, one in each AZ. And it roughly costs $33 per
 * NAT Gateway per month.
 * :::
 *
 * NAT Gateways are billed per hour and per gigabyte of data processed. By default,
 * this creates a NAT Gateway in each AZ. And this would be roughly $33 per NAT
 * Gateway per month. Make sure to [review the pricing](https://aws.amazon.com/vpc/pricing/).
 *
 * @example
 *
 * #### Create a VPC
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Vpc.v1("MyVPC");
 * ```
 *
 * #### Create it with 3 Availability Zones
 *
 * ```ts title="sst.config.ts" {2}
 * new sst.aws.Vpc.v1("MyVPC", {
 *   az: 3
 * });
 * ```
 */
var Vpc = /** @class */ (function (_super) {
    __extends(Vpc, _super);
    function Vpc(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        if (args && "ref" in args) {
            var ref = args;
            _this.vpc = ref.vpc;
            _this.internetGateway = ref.internetGateway;
            _this.securityGroup = ref.securityGroup;
            _this._publicSubnets = (0, pulumi_1.output)(ref.publicSubnets);
            _this._privateSubnets = (0, pulumi_1.output)(ref.privateSubnets);
            _this.publicRouteTables = (0, pulumi_1.output)(ref.publicRouteTables);
            _this.privateRouteTables = (0, pulumi_1.output)(ref.privateRouteTables);
            _this.natGateways = (0, pulumi_1.output)(ref.natGateways);
            _this.elasticIps = ref.elasticIps;
            return _this;
        }
        var parent = _this;
        var zones = normalizeAz();
        var vpc = createVpc();
        var internetGateway = createInternetGateway();
        var securityGroup = createSecurityGroup();
        var _a = createPublicSubnets(), publicSubnets = _a.publicSubnets, publicRouteTables = _a.publicRouteTables;
        var _b = createNatGateways(), elasticIps = _b.elasticIps, natGateways = _b.natGateways;
        var _c = createPrivateSubnets(), privateSubnets = _c.privateSubnets, privateRouteTables = _c.privateRouteTables;
        _this.vpc = vpc;
        _this.internetGateway = internetGateway;
        _this.securityGroup = securityGroup;
        _this.natGateways = natGateways;
        _this.elasticIps = elasticIps;
        _this._publicSubnets = publicSubnets;
        _this._privateSubnets = privateSubnets;
        _this.publicRouteTables = publicRouteTables;
        _this.privateRouteTables = privateRouteTables;
        function normalizeAz() {
            var _a;
            var zones = (0, aws_1.getAvailabilityZonesOutput)({
                state: "available",
            });
            return (0, pulumi_1.all)([zones, (_a = args === null || args === void 0 ? void 0 : args.az) !== null && _a !== void 0 ? _a : 2]).apply(function (_a) {
                var zones = _a[0], az = _a[1];
                return Array(az)
                    .fill(0)
                    .map(function (_, i) { return zones.names[i]; });
            });
        }
        function createVpc() {
            var _a;
            var _b;
            return new ((_a = aws_1.ec2.Vpc).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.vpc, "".concat(name, "Vpc"), {
                cidrBlock: "10.0.0.0/16",
                enableDnsSupport: true,
                enableDnsHostnames: true,
            }, { parent: parent }), false)))();
        }
        function createInternetGateway() {
            var _a;
            var _b;
            return new ((_a = aws_1.ec2.InternetGateway).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.internetGateway, "".concat(name, "InternetGateway"), {
                vpcId: vpc.id,
            }, { parent: parent }), false)))();
        }
        function createSecurityGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.ec2.SecurityGroup).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args === null || args === void 0 ? void 0 : args.transform) === null || _b === void 0 ? void 0 : _b.securityGroup, "".concat(name, "SecurityGroup"), {
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
            }, { parent: parent }), false)))();
        }
        function createNatGateways() {
            var ret = publicSubnets.apply(function (subnets) {
                return subnets.map(function (subnet, i) {
                    var _a, _b;
                    var _c, _d;
                    var elasticIp = new ((_a = aws_1.ec2.Eip).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_c = args === null || args === void 0 ? void 0 : args.transform) === null || _c === void 0 ? void 0 : _c.elasticIp, "".concat(name, "ElasticIp").concat(i + 1), {
                        vpc: true,
                    }, { parent: parent }), false)))();
                    var natGateway = new ((_b = aws_1.ec2.NatGateway).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_d = args === null || args === void 0 ? void 0 : args.transform) === null || _d === void 0 ? void 0 : _d.natGateway, "".concat(name, "NatGateway").concat(i + 1), {
                        subnetId: subnet.id,
                        allocationId: elasticIp.id,
                    }, { parent: parent }), false)))();
                    return { elasticIp: elasticIp, natGateway: natGateway };
                });
            });
            return {
                elasticIps: ret.apply(function (ret) { return ret.map(function (r) { return r.elasticIp; }); }),
                natGateways: ret.apply(function (ret) { return ret.map(function (r) { return r.natGateway; }); }),
            };
        }
        function createPublicSubnets() {
            var ret = zones.apply(function (zones) {
                return zones.map(function (zone, i) {
                    var _a, _b;
                    var _c, _d;
                    var subnet = new ((_a = aws_1.ec2.Subnet).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_c = args === null || args === void 0 ? void 0 : args.transform) === null || _c === void 0 ? void 0 : _c.publicSubnet, "".concat(name, "PublicSubnet").concat(i + 1), {
                        vpcId: vpc.id,
                        cidrBlock: "10.0.".concat(i + 1, ".0/24"),
                        availabilityZone: zone,
                        mapPublicIpOnLaunch: true,
                    }, { parent: parent }), false)))();
                    var routeTable = new ((_b = aws_1.ec2.RouteTable).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_d = args === null || args === void 0 ? void 0 : args.transform) === null || _d === void 0 ? void 0 : _d.publicRouteTable, "".concat(name, "PublicRouteTable").concat(i + 1), {
                        vpcId: vpc.id,
                        routes: [
                            {
                                cidrBlock: "0.0.0.0/0",
                                gatewayId: internetGateway.id,
                            },
                        ],
                    }, { parent: parent }), false)))();
                    new aws_1.ec2.RouteTableAssociation("".concat(name, "PublicRouteTableAssociation").concat(i + 1), {
                        subnetId: subnet.id,
                        routeTableId: routeTable.id,
                    }, { parent: parent });
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
                    var subnet = new ((_a = aws_1.ec2.Subnet).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_c = args === null || args === void 0 ? void 0 : args.transform) === null || _c === void 0 ? void 0 : _c.privateSubnet, "".concat(name, "PrivateSubnet").concat(i + 1), {
                        vpcId: vpc.id,
                        cidrBlock: "10.0.".concat(zones.length + i + 1, ".0/24"),
                        availabilityZone: zone,
                    }, { parent: parent }), false)))();
                    var routeTable = new ((_b = aws_1.ec2.RouteTable).bind.apply(_b, __spreadArray([void 0], (0, component_1.transform)((_d = args === null || args === void 0 ? void 0 : args.transform) === null || _d === void 0 ? void 0 : _d.privateRouteTable, "".concat(name, "PrivateRouteTable").concat(i + 1), {
                        vpcId: vpc.id,
                        routes: [
                            {
                                cidrBlock: "0.0.0.0/0",
                                natGatewayId: natGateways[i].id,
                            },
                        ],
                    }, { parent: parent }), false)))();
                    new aws_1.ec2.RouteTableAssociation("".concat(name, "PrivateRouteTableAssociation").concat(i + 1), {
                        subnetId: subnet.id,
                        routeTableId: routeTable.id,
                    }, { parent: parent });
                    return { subnet: subnet, routeTable: routeTable };
                });
            });
            return {
                privateSubnets: ret.apply(function (ret) { return ret.map(function (r) { return r.subnet; }); }),
                privateRouteTables: ret.apply(function (ret) { return ret.map(function (r) { return r.routeTable; }); }),
            };
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
            return [this.securityGroup.id];
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
     * @param vpcID The ID of the existing VPC.
     *
     * @example
     * Imagine you create a VPC in the `dev` stage. And in your personal stage `frank`,
     * instead of creating a new VPC, you want to share the VPC from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const vpc = $app.stage === "frank"
     *   ? sst.aws.Vpc.v1.get("MyVPC", "vpc-0be8fa4de860618bb")
     *   : new sst.aws.Vpc.v1("MyVPC");
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
    Vpc.get = function (name, vpcID) {
        var vpc = aws_1.ec2.Vpc.get("".concat(name, "Vpc"), vpcID);
        var internetGateway = aws_1.ec2.InternetGateway.get("".concat(name, "InstanceGateway"), aws_1.ec2.getInternetGatewayOutput({
            filters: [{ name: "attachment.vpc-id", values: [vpc.id] }],
        }).internetGatewayId);
        var securityGroup = aws_1.ec2.SecurityGroup.get("".concat(name, "SecurityGroup"), aws_1.ec2
            .getSecurityGroupsOutput({
            filters: [
                { name: "group-name", values: ["*SecurityGroup*"] },
                { name: "vpc-id", values: [vpc.id] },
            ],
        })
            .ids.apply(function (ids) {
            if (!ids.length)
                throw new Error("Security group not found in VPC ".concat(vpcID));
            return ids[0];
        }));
        var privateSubnets = aws_1.ec2
            .getSubnetsOutput({
            filters: [
                { name: "vpc-id", values: [vpc.id] },
                { name: "tag:Name", values: ["*Private*"] },
            ],
        })
            .ids.apply(function (ids) {
            return ids.map(function (id, i) { return aws_1.ec2.Subnet.get("".concat(name, "PrivateSubnet").concat(i + 1), id); });
        });
        var privateRouteTables = privateSubnets.apply(function (subnets) {
            return subnets.map(function (subnet, i) {
                return aws_1.ec2.RouteTable.get("".concat(name, "PrivateRouteTable").concat(i + 1), aws_1.ec2.getRouteTableOutput({ subnetId: subnet.id }).routeTableId);
            });
        });
        var publicSubnets = aws_1.ec2
            .getSubnetsOutput({
            filters: [
                { name: "vpc-id", values: [vpc.id] },
                { name: "tag:Name", values: ["*Public*"] },
            ],
        })
            .ids.apply(function (ids) {
            return ids.map(function (id, i) { return aws_1.ec2.Subnet.get("".concat(name, "PublicSubnet").concat(i + 1), id); });
        });
        var publicRouteTables = publicSubnets.apply(function (subnets) {
            return subnets.map(function (subnet, i) {
                return aws_1.ec2.RouteTable.get("".concat(name, "PublicRouteTable").concat(i + 1), aws_1.ec2.getRouteTableOutput({ subnetId: subnet.id }).routeTableId);
            });
        });
        var natGateways = publicSubnets.apply(function (subnets) {
            return subnets.map(function (subnet, i) {
                return aws_1.ec2.NatGateway.get("".concat(name, "NatGateway").concat(i + 1), aws_1.ec2.getNatGatewayOutput({ subnetId: subnet.id }).id);
            });
        });
        var elasticIps = natGateways.apply(function (nats) {
            return nats.map(function (nat, i) {
                return aws_1.ec2.Eip.get("".concat(name, "ElasticIp").concat(i + 1), nat.allocationId);
            });
        });
        return new Vpc(name, {
            ref: true,
            vpc: vpc,
            internetGateway: internetGateway,
            securityGroup: securityGroup,
            privateSubnets: privateSubnets,
            privateRouteTables: privateRouteTables,
            publicSubnets: publicSubnets,
            publicRouteTables: publicRouteTables,
            natGateways: natGateways,
            elasticIps: elasticIps,
        });
    };
    return Vpc;
}(component_1.Component));
exports.Vpc = Vpc;
var __pulumiType = "sst:aws:Vpc";
// @ts-expect-error
Vpc.__pulumiType = __pulumiType;
