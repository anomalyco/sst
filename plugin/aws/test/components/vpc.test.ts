import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy, assertions, generators } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

/**
 * Test VPC component migration and network configuration
 * Tests network architecture changes and AZ handling
 */
describe("VPC Component", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("VPC v2 migration", () => {
    it("should create VPC v2 component without migration warnings for new installations", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("TestVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16",
          availabilityZones: ["us-east-1a", "us-east-1b"],
          subnets: {
            public: {
              cidr: "10.0.0.0/24"
            },
            private: {
              cidr: "10.0.1.0/24"
            }
          }
        });

        vpc.registerVersion({
          new: "2.0.0",
          message: "VPC v2 with improved network architecture"
        });

        expect(vpc).toBeDefined();
        expect(vpc.originalName).toBe("TestVPC"); expect(vpc.name).toMatch(/test-app-test-testvpc-/);
        expect(vpc.args.cidr).toBe("10.0.0.0/16");
        expect(vpc.args.availabilityZones).toHaveLength(2);
      });
    });

    it("should require migration from v1 to v2 without forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("TestVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16"
        });

        expect(() => {
          vpc.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "VPC v2 introduces breaking changes to network architecture. Please review your subnet configuration."
          });
        }).toThrow(/Migration required for.*testvpc.*: VPC v2 introduces breaking changes/);
      });
    });

    it("should allow migration from v1 to v2 with forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("TestVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16",
          availabilityZones: ["us-east-1a", "us-east-1b"]
        });

        expect(() => {
          vpc.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "VPC v2 introduces breaking changes to network architecture. Please review your subnet configuration.",
            forceUpgrade: true
          });
        }).not.toThrow();

        const history = vpc.getVersionHistory();
        expect(history).toHaveLength(1);
        expect(history[0].forceUpgrade).toBe(true);
      });
    });
  });

  describe("VPC creation and configuration", () => {
    it("should create VPC with basic configuration", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("BasicVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16"
        });

        expect(vpc).toBeDefined();
        expect(vpc.args.cidr).toBe("10.0.0.0/16");
      });
    });

    it("should create VPC with comprehensive configuration", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("ComprehensiveVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16",
          enableDnsHostnames: true,
          enableDnsSupport: true,
          availabilityZones: ["us-east-1a", "us-east-1b", "us-east-1c"],
          subnets: {
            public: {
              cidr: "10.0.0.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b"]
            },
            private: {
              cidr: "10.0.1.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b"]
            },
            database: {
              cidr: "10.0.2.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b"]
            }
          },
          natGateways: {
            strategy: "single"
          }
        });

        expect(vpc.args.enableDnsHostnames).toBe(true);
        expect(vpc.args.enableDnsSupport).toBe(true);
        expect(vpc.args.availabilityZones).toHaveLength(3);
        expect(vpc.args.subnets.public).toBeDefined();
        expect(vpc.args.subnets.private).toBeDefined();
        expect(vpc.args.subnets.database).toBeDefined();
        expect(vpc.args.natGateways.strategy).toBe("single");
      });
    });

    it("should handle minimal VPC configuration", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("MinimalVPC", "aws:vpc:component");

        expect(vpc).toBeDefined();
        expect(vpc.args.cidr).toBeUndefined();
      });
    });
  });

  describe("VPC availability zone handling", () => {
    it("should handle explicit availability zones", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("ExplicitAZVPC", "aws:vpc:component", {
          availabilityZones: ["us-east-1a", "us-east-1b", "us-east-1c"]
        });

        expect(vpc.args.availabilityZones).toHaveLength(3);
        expect(vpc.args.availabilityZones).toContain("us-east-1a");
        expect(vpc.args.availabilityZones).toContain("us-east-1b");
        expect(vpc.args.availabilityZones).toContain("us-east-1c");
      });
    });

    it("should handle automatic availability zone selection", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("AutoAZVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16"
          // No explicit AZs - should use automatic selection
        });

        expect(vpc.args.availabilityZones).toBeUndefined();
      });
    });

    it("should handle single availability zone", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("SingleAZVPC", "aws:vpc:component", {
          availabilityZones: ["us-east-1a"]
        });

        expect(vpc.args.availabilityZones).toHaveLength(1);
        expect(vpc.args.availabilityZones[0]).toBe("us-east-1a");
      });
    });

    it("should handle cross-region availability zones", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("CrossRegionVPC", "aws:vpc:component", {
          availabilityZones: ["us-west-2a", "us-west-2b", "us-west-2c"]
        });

        expect(vpc.args.availabilityZones).toHaveLength(3);
        expect(vpc.args.availabilityZones.every(az => az.startsWith("us-west-2"))).toBe(true);
      });
    });
  });

  describe("VPC subnet configuration", () => {
    it("should handle public subnet configuration", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("PublicSubnetVPC", "aws:vpc:component", {
          subnets: {
            public: {
              cidr: "10.0.0.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b"]
            }
          }
        });

        expect(vpc.args.subnets.public.cidr).toBe("10.0.0.0/24");
        expect(vpc.args.subnets.public.availabilityZones).toHaveLength(2);
      });
    });

    it("should handle private subnet configuration", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("PrivateSubnetVPC", "aws:vpc:component", {
          subnets: {
            private: {
              cidr: "10.0.1.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b"]
            }
          }
        });

        expect(vpc.args.subnets.private.cidr).toBe("10.0.1.0/24");
        expect(vpc.args.subnets.private.availabilityZones).toHaveLength(2);
      });
    });

    it("should handle database subnet configuration", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("DatabaseSubnetVPC", "aws:vpc:component", {
          subnets: {
            database: {
              cidr: "10.0.2.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b", "us-east-1c"]
            }
          }
        });

        expect(vpc.args.subnets.database.cidr).toBe("10.0.2.0/24");
        expect(vpc.args.subnets.database.availabilityZones).toHaveLength(3);
      });
    });

    it("should handle multiple subnet types", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("MultiSubnetVPC", "aws:vpc:component", {
          subnets: {
            public: {
              cidr: "10.0.0.0/24"
            },
            private: {
              cidr: "10.0.1.0/24"
            },
            database: {
              cidr: "10.0.2.0/24"
            },
            cache: {
              cidr: "10.0.3.0/24"
            }
          }
        });

        expect(Object.keys(vpc.args.subnets)).toHaveLength(4);
        expect(vpc.args.subnets.public).toBeDefined();
        expect(vpc.args.subnets.private).toBeDefined();
        expect(vpc.args.subnets.database).toBeDefined();
        expect(vpc.args.subnets.cache).toBeDefined();
      });
    });

    it("should handle empty subnet configuration", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("EmptySubnetVPC", "aws:vpc:component", {
          subnets: {}
        });

        expect(vpc.args.subnets).toEqual({});
      });
    });
  });

  describe("VPC NAT Gateway configuration", () => {
    it("should handle single NAT Gateway strategy", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("SingleNATVPC", "aws:vpc:component", {
          natGateways: {
            strategy: "single"
          }
        });

        expect(vpc.args.natGateways.strategy).toBe("single");
      });
    });

    it("should handle per-AZ NAT Gateway strategy", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("PerAZNATVPC", "aws:vpc:component", {
          natGateways: {
            strategy: "per-az"
          }
        });

        expect(vpc.args.natGateways.strategy).toBe("per-az");
      });
    });

    it("should handle no NAT Gateway configuration", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("NoNATVPC", "aws:vpc:component", {
          natGateways: {
            strategy: "none"
          }
        });

        expect(vpc.args.natGateways.strategy).toBe("none");
      });
    });

    it("should handle undefined NAT Gateway configuration", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("DefaultNATVPC", "aws:vpc:component");

        expect(vpc.args.natGateways).toBeUndefined();
      });
    });
  });

  describe("VPC DNS configuration", () => {
    it("should handle DNS hostnames enabled", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("DNSHostnamesVPC", "aws:vpc:component", {
          enableDnsHostnames: true
        });

        expect(vpc.args.enableDnsHostnames).toBe(true);
      });
    });

    it("should handle DNS support enabled", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("DNSSupportVPC", "aws:vpc:component", {
          enableDnsSupport: true
        });

        expect(vpc.args.enableDnsSupport).toBe(true);
      });
    });

    it("should handle both DNS options enabled", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("FullDNSVPC", "aws:vpc:component", {
          enableDnsHostnames: true,
          enableDnsSupport: true
        });

        expect(vpc.args.enableDnsHostnames).toBe(true);
        expect(vpc.args.enableDnsSupport).toBe(true);
      });
    });

    it("should handle DNS options disabled", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("NoDNSVPC", "aws:vpc:component", {
          enableDnsHostnames: false,
          enableDnsSupport: false
        });

        expect(vpc.args.enableDnsHostnames).toBe(false);
        expect(vpc.args.enableDnsSupport).toBe(false);
      });
    });
  });

  describe("VPC naming", () => {
    it("should generate valid VPC names", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("MyTestVPC", "aws:vpc:component");
        const physicalName = vpc.generatePhysicalName("vpc");
        
        assertions.validAWSName(physicalName);
        expect(physicalName.value).toMatch(/test-app-test-vpc-/);
      });
    });

    it("should handle VPC names with special characters", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("My-Test_VPC.v2", "aws:vpc:component");
        const physicalName = vpc.generatePhysicalName("vpc");
        
        assertions.validAWSName(physicalName);
        expect(physicalName.value).toMatch(/vpc/);
      });
    });
  });

  describe("VPC v1 legacy support", () => {
    it("should provide access to v1 VPC component", async () => {
      await withTestEnvironment(async () => {
        const VpcV1 = class extends MockAWSComponent {
          constructor(name: string, args: any = {}) {
            super(name, "aws:vpc:v1:component", args);
            this.registerVersion({
              new: "1.0.0",
              message: "VPC v1 legacy component"
            });
          }
        };

        const vpcV1 = new VpcV1("LegacyVPC", {
          cidr: "10.0.0.0/16",
          publicSubnets: ["10.0.1.0/24", "10.0.2.0/24"],
          privateSubnets: ["10.0.3.0/24", "10.0.4.0/24"]
        });

        expect(vpcV1).toBeDefined();
        expect(vpcV1.type).toBe("aws:vpc:v1:component");
        expect(vpcV1.args.publicSubnets).toHaveLength(2);
        expect(vpcV1.args.privateSubnets).toHaveLength(2);
      });
    });

    it("should maintain v1 component functionality", async () => {
      await withTestEnvironment(async () => {
        const vpcV1 = new MockAWSComponent("LegacyVPC", "aws:vpc:v1:component", {
          cidr: "10.0.0.0/16",
          publicSubnets: ["10.0.1.0/24"],
          privateSubnets: ["10.0.2.0/24"],
          enableNatGateway: true,
          singleNatGateway: true
        });

        vpcV1.registerVersion({
          new: "1.0.0",
          message: "VPC v1 with legacy subnet configuration"
        });

        expect(vpcV1.args.enableNatGateway).toBe(true);
        expect(vpcV1.args.singleNatGateway).toBe(true);
        expect(vpcV1.args.publicSubnets).toHaveLength(1);
        expect(vpcV1.args.privateSubnets).toHaveLength(1);
      });
    });
  });

  describe("VPC integration scenarios", () => {
    it("should integrate with Service component", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("ServiceVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16",
          subnets: {
            private: {
              cidr: "10.0.1.0/24"
            }
          }
        });

        const service = new MockAWSComponent("TestService", "aws:service:component", {
          vpc: {
            vpcId: vpc.generatePhysicalName("vpc"),
            subnetIds: vpc.generatePhysicalName("private-subnets"),
            securityGroupIds: vpc.generatePhysicalName("security-group")
          }
        });

        assertions.validOutput(service.args.vpc.vpcId);
        assertions.validOutput(service.args.vpc.subnetIds);
        assertions.validOutput(service.args.vpc.securityGroupIds);
      });
    });

    it("should integrate with RDS database", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("DatabaseVPC", "aws:vpc:component", {
          subnets: {
            database: {
              cidr: "10.0.2.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b"]
            }
          }
        });

        const database = new MockAWSComponent("TestDatabase", "aws:postgres:component", {
          vpc: {
            subnetIds: vpc.generatePhysicalName("database-subnets"),
            securityGroupIds: vpc.generatePhysicalName("database-security-group")
          }
        });

        assertions.validOutput(database.args.vpc.subnetIds);
        assertions.validOutput(database.args.vpc.securityGroupIds);
      });
    });

    it("should integrate with Lambda functions", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("LambdaVPC", "aws:vpc:component", {
          subnets: {
            private: {
              cidr: "10.0.1.0/24"
            }
          }
        });

        const lambdaFunction = new MockAWSComponent("VPCFunction", "aws:function:component", {
          handler: "index.handler",
          vpc: {
            securityGroups: [vpc.generatePhysicalName("lambda-security-group")],
            subnets: [vpc.generatePhysicalName("private-subnets")]
          }
        });

        expect(lambdaFunction.args.vpc.securityGroups).toHaveLength(1);
        expect(lambdaFunction.args.vpc.subnets).toHaveLength(1);
        assertions.validOutput(lambdaFunction.args.vpc.securityGroups[0]);
        assertions.validOutput(lambdaFunction.args.vpc.subnets[0]);
      });
    });
  });

  describe("VPC error handling", () => {
    it("should handle invalid CIDR blocks", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("InvalidCIDRVPC", "aws:vpc:component", {
          cidr: "invalid-cidr"
        });

        expect(vpc.args.cidr).toBe("invalid-cidr");
      });
    });

    it("should handle invalid availability zones", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("InvalidAZVPC", "aws:vpc:component", {
          availabilityZones: ["invalid-az-1", "invalid-az-2"]
        });

        expect(vpc.args.availabilityZones).toContain("invalid-az-1");
        expect(vpc.args.availabilityZones).toContain("invalid-az-2");
      });
    });

    it("should handle conflicting subnet CIDRs", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("ConflictingSubnetsVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16",
          subnets: {
            public: {
              cidr: "10.0.1.0/24"
            },
            private: {
              cidr: "10.0.1.0/24" // Same CIDR as public
            }
          }
        });

        expect(vpc.args.subnets.public.cidr).toBe("10.0.1.0/24");
        expect(vpc.args.subnets.private.cidr).toBe("10.0.1.0/24");
      });
    });

    it("should handle invalid NAT Gateway strategy", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("InvalidNATVPC", "aws:vpc:component", {
          natGateways: {
            strategy: "invalid-strategy"
          }
        });

        expect(vpc.args.natGateways.strategy).toBe("invalid-strategy");
      });
    });
  });

  describe("VPC security group configuration", () => {
    it("should handle default security group configuration", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("SecurityGroupVPC", "aws:vpc:component", {
          securityGroups: {
            default: {
              ingress: [
                {
                  protocol: "tcp",
                  fromPort: 80,
                  toPort: 80,
                  cidrBlocks: ["0.0.0.0/0"]
                }
              ],
              egress: [
                {
                  protocol: "-1",
                  fromPort: 0,
                  toPort: 0,
                  cidrBlocks: ["0.0.0.0/0"]
                }
              ]
            }
          }
        });

        expect(vpc.args.securityGroups.default.ingress).toHaveLength(1);
        expect(vpc.args.securityGroups.default.egress).toHaveLength(1);
        expect(vpc.args.securityGroups.default.ingress[0].fromPort).toBe(80);
      });
    });

    it("should handle multiple security groups", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("MultiSecurityGroupVPC", "aws:vpc:component", {
          securityGroups: {
            web: {
              ingress: [
                { protocol: "tcp", fromPort: 80, toPort: 80, cidrBlocks: ["0.0.0.0/0"] },
                { protocol: "tcp", fromPort: 443, toPort: 443, cidrBlocks: ["0.0.0.0/0"] }
              ]
            },
            database: {
              ingress: [
                { protocol: "tcp", fromPort: 5432, toPort: 5432, sourceSecurityGroupId: "sg-web" }
              ]
            }
          }
        });

        expect(Object.keys(vpc.args.securityGroups)).toHaveLength(2);
        expect(vpc.args.securityGroups.web.ingress).toHaveLength(2);
        expect(vpc.args.securityGroups.database.ingress).toHaveLength(1);
      });
    });
  });
});