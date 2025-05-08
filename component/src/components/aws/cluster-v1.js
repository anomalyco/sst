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
exports.Cluster = exports.supportedMemories = exports.supportedCpus = void 0;
var component_js_1 = require("../component.js");
var service_v1_js_1 = require("./service-v1.js");
var aws_1 = require("@pulumi/aws");
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
/**
 * The `Cluster` component lets you create a cluster of containers and add services to them.
 * It uses [Amazon ECS](https://aws.amazon.com/ecs/) on [AWS Fargate](https://aws.amazon.com/fargate/).
 *
 * For existing usage, rename `sst.aws.Cluster` to `sst.aws.Cluster.v1`. For new Clusters, use
 * the latest [`Cluster`](/docs/component/aws/cluster) component instead.
 *
 * :::caution
 * This component has been deprecated .
 * :::
 *
 * @example
 *
 * #### Create a Cluster
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const cluster = new sst.aws.Cluster.v1("MyCluster", { vpc });
 * ```
 *
 * #### Add a service
 *
 * ```ts title="sst.config.ts"
 * cluster.addService("MyService");
 * ```
 *
 * #### Add a public custom domain
 *
 * ```ts title="sst.config.ts"
 * cluster.addService("MyService", {
 *   public: {
 *     domain: "example.com",
 *     ports: [
 *       { listen: "80/http" },
 *       { listen: "443/https", forward: "80/http" },
 *     ]
 *   }
 * });
 * ```
 *
 * #### Enable auto-scaling
 *
 * ```ts title="sst.config.ts"
 * cluster.addService("MyService", {
 *   scaling: {
 *     min: 4,
 *     max: 16,
 *     cpuUtilization: 50,
 *     memoryUtilization: 50,
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your service. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * cluster.addService("MyService", {
 *   link: [bucket],
 * });
 * ```
 *
 * If your service is written in Node.js, you can use the [SDK](/docs/reference/sdk/)
 * to access the linked resources.
 *
 * ```ts title="app.ts"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 */
var Cluster = /** @class */ (function (_super) {
    __extends(Cluster, _super);
    function Cluster(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var cluster = createCluster();
        _this.args = args;
        _this.cluster = cluster;
        function createCluster() {
            var _a;
            var _b;
            return new ((_a = aws_1.ecs.Cluster).bind.apply(_a, __spreadArray([void 0], (0, component_js_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.cluster, "".concat(name, "Cluster"), {}, { parent: parent }), false)))();
        }
        return _this;
    }
    Object.defineProperty(Cluster.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            return {
                /**
                 * The Amazon ECS Cluster.
                 */
                cluster: this.cluster,
            };
        },
        enumerable: false,
        configurable: true
    });
    /**
     * Add a service to the cluster.
     *
     * @param name Name of the service.
     * @param args Configure the service.
     *
     * @example
     *
     * ```ts title="sst.config.ts"
     * cluster.addService("MyService");
     * ```
     *
     * Set a custom domain for the service.
     *
     * ```js {2} title="sst.config.ts"
     * cluster.addService("MyService", {
     *   domain: "example.com"
     * });
     * ```
     *
     * #### Enable auto-scaling
     *
     * ```ts title="sst.config.ts"
     * cluster.addService("MyService", {
     *   scaling: {
     *     min: 4,
     *     max: 16,
     *     cpuUtilization: 50,
     *     memoryUtilization: 50,
     *   }
     * });
     * ```
     */
    Cluster.prototype.addService = function (name, args) {
        // Do not prefix the service to allow `Resource.MyService` to work.
        return new service_v1_js_1.Service(name, __assign({ cluster: {
                name: this.cluster.name,
                arn: this.cluster.arn,
            }, vpc: this.args.vpc }, args));
    };
    return Cluster;
}(component_js_1.Component));
exports.Cluster = Cluster;
var __pulumiType = "sst:aws:Cluster";
// @ts-expect-error
Cluster.__pulumiType = __pulumiType;
