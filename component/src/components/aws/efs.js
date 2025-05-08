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
exports.Efs = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_js_1 = require("../component.js");
var aws_1 = require("@pulumi/aws");
var vpc_js_1 = require("./vpc.js");
var error_js_1 = require("../error.js");
/**
 * The `Efs` component lets you add [Amazon Elastic File System (EFS)](https://docs.aws.amazon.com/efs/latest/ug/whatisefs.html) to your app.
 *
 * @example
 *
 * #### Create the file system
 *
 * ```js title="sst.config.ts" {2}
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const efs = new sst.aws.Efs("MyEfs", { vpc });
 * ```
 *
 * This needs a VPC.
 *
 * #### Attach it to a Lambda function
 *
 * ```ts title="sst.config.ts" {4}
 * new sst.aws.Function("MyFunction", {
 *   vpc,
 *   handler: "lambda.handler",
 *   volume: { efs, path: "/mnt/efs" }
 * });
 * ```
 *
 * This is now mounted at `/mnt/efs` in the Lambda function.
 *
 * #### Attach it to a container
 *
 * ```ts title="sst.config.ts" {7}
 * const cluster = new sst.aws.Cluster("MyCluster", { vpc });
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   public: {
 *     ports: [{ listen: "80/http" }],
 *   },
 *   volumes: [
 *     { efs, path: "/mnt/efs" }
 *   ]
 * });
 * ```
 *
 * Mounted at `/mnt/efs` in the container.
 *
 * ---
 *
 * ### Cost
 *
 * By default this component uses _Regional (Multi-AZ) with Elastic Throughput_. The pricing is
 * pay-per-use.
 *
 * - For storage: $0.30 per GB per month
 * - For reads: $0.03 per GB per month
 * - For writes: $0.06 per GB per month
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [EFS pricing](https://aws.amazon.com/efs/pricing/) for more details.
 */
var Efs = /** @class */ (function (_super) {
    __extends(Efs, _super);
    function Efs(name, args, opts) {
        var _a, _b;
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        if (args && "ref" in args) {
            var ref = args;
            _this._fileSystem = (0, pulumi_1.output)(ref.fileSystem);
            _this._accessPoint = (0, pulumi_1.output)(ref.accessPoint);
            return _this;
        }
        var parent = _this;
        var vpc = normalizeVpc();
        var throughput = (0, pulumi_1.output)((_a = args.throughput) !== null && _a !== void 0 ? _a : "elastic");
        var performance = (0, pulumi_1.output)((_b = args.performance) !== null && _b !== void 0 ? _b : "general-purpose");
        var fileSystem = createFileSystem();
        var securityGroup = createSecurityGroup();
        var mountTargets = createMountTargets();
        var accessPoint = createAccessPoint();
        var waited = mountTargets.apply(function (targets) {
            return (0, pulumi_1.all)(targets.map(function (target) { return target.urn; })).apply(function () { return ({
                fileSystem: fileSystem,
                accessPoint: accessPoint,
            }); });
        });
        _this._fileSystem = waited.fileSystem;
        _this._accessPoint = waited.accessPoint;
        function normalizeVpc() {
            // "vpc" is a Vpc component
            if (args.vpc instanceof vpc_js_1.Vpc) {
                return (0, pulumi_1.output)({
                    id: args.vpc.id,
                    subnets: args.vpc.privateSubnets,
                    cidrBlock: args.vpc.nodes.vpc.cidrBlock,
                });
            }
            // "vpc" is object
            return (0, pulumi_1.output)(args.vpc).apply(function (vpc) {
                // Because `vpc.id` is newly required since v3.3.66, some people might not have
                // it, and they should get a type error. We want to throw a descriptive error.
                if (!vpc.id)
                    throw new error_js_1.VisibleError("Missing \"vpc.id\" for the \"".concat(name, "\" EFS component. The VPC id is required to create the security group for the EFS mount targets."));
                var vpcRef = aws_1.ec2.Vpc.get("".concat(name, "Vpc"), vpc.id, undefined, {
                    parent: parent,
                });
                return {
                    id: vpc.id,
                    subnets: vpc.subnets,
                    cidrBlock: vpcRef.cidrBlock,
                };
            });
        }
        function createFileSystem() {
            var _a;
            var _b;
            return new ((_a = aws_1.efs.FileSystem).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.fileSystem, "".concat(name, "FileSystem"), {
                performanceMode: performance.apply(function (v) {
                    return v === "general-purpose" ? "generalPurpose" : "maxIO";
                }),
                throughputMode: throughput,
                encrypted: true,
            }, { parent: parent }), false)))();
        }
        function createSecurityGroup() {
            var _a;
            var _b;
            return new ((_a = aws_1.ec2.SecurityGroup).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.securityGroup, "".concat(name, "SecurityGroup"), {
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
            }, { parent: parent }), false)))();
        }
        function createMountTargets() {
            return vpc.subnets.apply(function (subnets) {
                return subnets.map(function (subnet) {
                    return new aws_1.efs.MountTarget("".concat(name, "MountTarget").concat(subnet), {
                        fileSystemId: fileSystem.id,
                        subnetId: subnet,
                        securityGroups: [securityGroup.id],
                    }, { parent: parent });
                });
            });
        }
        function createAccessPoint() {
            var _a;
            var _b;
            return new ((_a = aws_1.efs.AccessPoint).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.accessPoint, "".concat(name, "AccessPoint"), {
                fileSystemId: fileSystem.id,
                posixUser: {
                    uid: 0,
                    gid: 0,
                },
                rootDirectory: {
                    path: "/",
                },
            }, { parent: parent }), false)))();
        }
        return _this;
    }
    Object.defineProperty(Efs.prototype, "id", {
        /**
         * The ID of the EFS file system.
         */
        get: function () {
            return this._fileSystem.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Efs.prototype, "accessPoint", {
        /**
         * The ID of the EFS access point.
         */
        get: function () {
            return this._accessPoint.id;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Efs.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon EFS file system.
                 */
                fileSystem: this._fileSystem,
                /**
                 * The Amazon EFS access point.
                 */
                accessPoint: this._accessPoint,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Reference an existing EFS file system with the given file system ID. This is useful when
     * you create a EFS file system in one stage and want to share it in another. It avoids
     * having to create a new EFS file system in the other stage.
     *
     * :::tip
     * You can use the `static get` method to share EFS file systems across stages.
     * :::
     *
     * @param name The name of the component.
     * @param fileSystemID The ID of the existing EFS file system.
     * @param opts? Resource options.
     *
     * @example
     * Imagine you create a EFS file system in the `dev` stage. And in your personal stage
     * `frank`, instead of creating a new file system, you want to share the same file system
     * from `dev`.
     *
     * ```ts title="sst.config.ts"
     * const efs = $app.stage === "frank"
     *   ? sst.aws.Efs.get("MyEfs", "app-dev-myefs")
     *   : new sst.aws.Efs("MyEfs", { vpc });
     * ```
     *
     * Here `app-dev-myefs` is the ID of the file system created in the `dev` stage.
     * You can find this by outputting the file system ID in the `dev` stage.
     *
     * ```ts title="sst.config.ts"
     * return {
     *   id: efs.id
     * };
     * ```
     */
    Efs.get = function (name, fileSystemID, opts) {
        var fileSystem = aws_1.efs.FileSystem.get("".concat(name, "FileSystem"), fileSystemID, undefined, opts);
        var accessPointId = aws_1.efs
            .getAccessPointsOutput({ fileSystemId: fileSystem.id }, opts)
            .apply(function (accessPoints) { return accessPoints.ids[0]; });
        var accessPoint = aws_1.efs.AccessPoint.get("".concat(name, "AccessPoint"), accessPointId, undefined, opts);
        return new Efs(name, {
            ref: true,
            fileSystem: fileSystem,
            accessPoint: accessPoint,
        });
    };
    return Efs;
}(component_js_1.Component));
exports.Efs = Efs;
var __pulumiType = "sst:aws:Efs";
// @ts-expect-error
Efs.__pulumiType = __pulumiType;
