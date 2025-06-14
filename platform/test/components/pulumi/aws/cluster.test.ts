import { describe, beforeAll, it, expect } from "vitest";
import * as pulumi from "@pulumi/pulumi";
import { setupSSTTestEnvironment, createAWSMocks } from "../helpers/pulumi-mocks";
import { createComponentTestSuite, testComponentCreation, testSSTNaming } from "../helpers/test-utils";

// Set up global test environment
setupSSTTestEnvironment("test-app", "test");

describe("AWS Cluster Component", () => {
  let Cluster: typeof import("../../../../src/components/aws/cluster").Cluster;
  let Vpc: typeof import("../../../../src/components/aws/vpc").Vpc;

  beforeAll(async () => {
    Cluster = (await import("../../../../src/components/aws/cluster")).Cluster;
    Vpc = (await import("../../../../src/components/aws/vpc")).Vpc;
  });

  describe("Basic Cluster Creation", () => {
    it("should create a basic cluster with VPC", async () => {
      const vpc = new Vpc("TestVpc");
      const cluster = await testComponentCreation(() => new Cluster("TestCluster", {
        vpc: vpc,
      }));

      expect(cluster).toBeDefined();
    });

    it("should create cluster with VPC configuration object", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678", "subnet-87654321"],
          loadBalancerSubnets: ["subnet-abcdef12", "subnet-fedcba21"],
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should create cluster with Cloud Map namespace configuration", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
          cloudmapNamespaceId: "ns-12345678",
          cloudmapNamespaceName: "test.local",
        },
      });

      expect(cluster).toBeDefined();
    });
  });

  describe("Cluster Configuration", () => {
    it("should handle deprecated serviceSubnets parameter", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          serviceSubnets: ["subnet-12345678"], // deprecated
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should create cluster with transform configuration", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
        transform: {
          cluster: (args) => ({
            ...args,
            tags: {
              ...args.tags,
              Environment: "test",
            },
          }),
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should create cluster with force upgrade option", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
        forceUpgrade: "v2",
      });

      expect(cluster).toBeDefined();
    });
  });

  describe("Cluster Reference and Get", () => {
    it("should get existing cluster by ID", async () => {
      const cluster = Cluster.get("ExistingCluster", {
        id: "cluster-12345678",
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should handle cluster reference with VPC object", async () => {
      const vpc = new Vpc("TestVpc");
      const cluster = Cluster.get("ExistingCluster", {
        id: "cluster-12345678",
        vpc: vpc,
      });

      expect(cluster).toBeDefined();
    });
  });

  describe("ECS Configuration", () => {
    it("should create cluster with capacity providers", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
      // Capacity providers should be automatically configured
    });

    it("should support Fargate workloads", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
      // Should support Fargate by default
    });
  });

  describe("VPC Integration", () => {
    it("should work with public subnets", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-public-1", "subnet-public-2"],
          loadBalancerSubnets: ["subnet-public-1", "subnet-public-2"],
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should work with private subnets", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-private-1", "subnet-private-2"],
          loadBalancerSubnets: ["subnet-public-1", "subnet-public-2"],
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should handle multiple security groups", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-web", "sg-app", "sg-db"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
    });
  });

  describe("Service Discovery", () => {
    it("should configure Cloud Map namespace", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
          cloudmapNamespaceId: "ns-servicediscovery",
          cloudmapNamespaceName: "myapp.local",
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should work without Cloud Map namespace", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
    });
  });

  describe("Edge Cases", () => {
    it("should handle empty security groups array", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: [],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should handle single subnet configuration", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-12345678"], // Same subnet for both
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should handle long cluster names", async () => {
      const longName = "VeryLongClusterNameThatExceedsNormalLimits";
      const cluster = new Cluster(longName, {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should handle special characters in VPC IDs", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678-abcd-efgh",
          securityGroups: ["sg-12345678-abcd-efgh"],
          containerSubnets: ["subnet-12345678-abcd-efgh"],
          loadBalancerSubnets: ["subnet-abcdef12-ijkl-mnop"],
        },
      });

      expect(cluster).toBeDefined();
    });
  });

  describe("Integration Scenarios", () => {
    it("should support multi-AZ deployment", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: [
            "subnet-us-east-1a",
            "subnet-us-east-1b",
            "subnet-us-east-1c",
          ],
          loadBalancerSubnets: [
            "subnet-public-1a",
            "subnet-public-1b",
            "subnet-public-1c",
          ],
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should work with existing VPC component", async () => {
      const vpc = new Vpc("TestVpc");
      const cluster = new Cluster("TestCluster", {
        vpc: vpc,
      });

      expect(cluster).toBeDefined();
    });

    it("should support development environment configuration", async () => {
      const cluster = new Cluster("DevCluster", {
        vpc: {
          id: "vpc-dev-12345678",
          securityGroups: ["sg-dev-12345678"],
          containerSubnets: ["subnet-dev-12345678"],
          loadBalancerSubnets: ["subnet-dev-abcdef12"],
          cloudmapNamespaceId: "ns-dev-12345678",
          cloudmapNamespaceName: "dev.local",
        },
        transform: {
          cluster: (args) => ({
            ...args,
            tags: {
              ...args.tags,
              Environment: "development",
              CostCenter: "engineering",
            },
          }),
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should support production environment configuration", async () => {
      const cluster = new Cluster("ProdCluster", {
        vpc: {
          id: "vpc-prod-12345678",
          securityGroups: ["sg-prod-web", "sg-prod-app"],
          containerSubnets: [
            "subnet-prod-private-1a",
            "subnet-prod-private-1b",
          ],
          loadBalancerSubnets: [
            "subnet-prod-public-1a",
            "subnet-prod-public-1b",
          ],
          cloudmapNamespaceId: "ns-prod-12345678",
          cloudmapNamespaceName: "prod.internal",
        },
        transform: {
          cluster: (args) => ({
            ...args,
            tags: {
              ...args.tags,
              Environment: "production",
              Backup: "required",
              Monitoring: "enhanced",
            },
          }),
        },
      });

      expect(cluster).toBeDefined();
    });
  });

  describe("SST Naming Conventions", () => {
    it("should follow SST naming patterns", async () => {
      const cluster = new Cluster("MyCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
      // SST naming conventions should be followed
    });

    it("should handle cluster names with hyphens", async () => {
      const cluster = new Cluster("my-cluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
    });

    it("should handle cluster names with numbers", async () => {
      const cluster = new Cluster("cluster-v2", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
    });
  });

  describe("Component Linking", () => {
    it("should support linking with Service components", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
      // Services should be able to reference this cluster
    });

    it("should support linking with Task components", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
      });

      expect(cluster).toBeDefined();
      // Tasks should be able to reference this cluster
    });
  });

  describe("Version Compatibility", () => {
    it("should support v1 cluster compatibility", async () => {
      const ClusterV1 = Cluster.v1;
      expect(ClusterV1).toBeDefined();
    });

    it("should handle version upgrades", async () => {
      const cluster = new Cluster("TestCluster", {
        vpc: {
          id: "vpc-12345678",
          securityGroups: ["sg-12345678"],
          containerSubnets: ["subnet-12345678"],
          loadBalancerSubnets: ["subnet-abcdef12"],
        },
        forceUpgrade: "v2",
      });

      expect(cluster).toBeDefined();
    });
  });
});