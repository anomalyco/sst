import { describe, beforeAll, it, expect } from "vitest";
import "../../src/global.d.ts";
import * as pulumi from "@pulumi/pulumi";
import fs from "fs";
import path from "path";

// Create temp directories for Docker context resolution in fargate.ts
const testRoot = path.join("/tmp", "sst-test-" + process.pid);
for (const dir of [".", "api", "worker"]) {
  fs.mkdirSync(path.join(testRoot, dir), { recursive: true });
}

// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
};
global.$util = pulumi;
// @ts-ignore — Service checks $dev to decide dev mode
global.$dev = false;
// @ts-ignore — Cluster reads $cli.state.version, fargate reads $cli.paths.root
global.$cli = {
  state: { version: {} },
  paths: { root: testRoot },
};
// @ts-ignore — fargate.ts uses $jsonStringify for container definitions
global.$jsonStringify = JSON.stringify;
// Suppress SST_SERVER RPC calls during tests
process.env.SST_SERVER = "http://localhost:13557";

const createdResources: pulumi.runtime.MockResourceArgs[] = [];

pulumi.runtime.setMocks(
  {
    newResource: function (args: pulumi.runtime.MockResourceArgs): {
      id: string;
      state: any;
    } {
      createdResources.push(args);
      return {
        id: args.inputs.name + "_id",
        state: {
          ...args.inputs,
          arn: `arn:aws:mock:us-east-1:123456789:${args.type}/${args.inputs.name}`,
          dnsName: `${args.inputs.name}.us-east-1.elb.amazonaws.com`,
          zoneId: "Z1234567890",
          name: args.inputs.name || args.name,
          status: "ACTIVE",
          tags: {
            "sst:ref:version": "1",
            "sst:ref:sg": "sg-mock-id",
            "sst:ref:vpc-id": "vpc-123",
          },
        },
      };
    },
    call: function (args: pulumi.runtime.MockCallArgs) {
      if (args.token === "aws:alb/getListener:getListener") {
        return {
          arn: `arn:aws:elasticloadbalancing:us-east-1:123456789:listener/app/mock/${args.inputs.port}`,
          ...args.inputs,
        };
      }
      if (args.token === "aws:index/getRegion:getRegion") {
        return { name: "us-east-1", description: "US East (N. Virginia)" };
      }
      return args.inputs;
    },
  },
  "project",
  "stack",
  false,
);

describe("Service with external ALB", function () {
  let Service: typeof import("../../src/components/aws/service").Service;
  let Alb: typeof import("../../src/components/aws/alb").Alb;
  let Cluster: typeof import("../../src/components/aws/cluster").Cluster;

  // Shared fixtures
  let cluster: InstanceType<typeof Cluster>;
  let alb: InstanceType<typeof Alb>;

  beforeAll(async function () {
    Alb = (await import("../../src/components/aws/alb")).Alb;
    Cluster = (await import("../../src/components/aws/cluster")).Cluster;
    Service = (await import("../../src/components/aws/service")).Service;

    cluster = new Cluster("TestCluster", {
      vpc: {
        id: "vpc-123",
        securityGroups: ["sg-123"],
        containerSubnets: ["subnet-a", "subnet-b"],
        loadBalancerSubnets: ["subnet-a", "subnet-b"],
      },
    });

    alb = new Alb("TestAlb", {
      vpc: {
        id: "vpc-123",
        publicSubnets: ["subnet-a", "subnet-b"],
        privateSubnets: ["subnet-c", "subnet-d"],
      },
      listeners: [{ port: 80, protocol: "http" }],
    });
  });

  describe("detectAlbAttachment", () => {
    it("creates service with external ALB (happy path)", async () => {
      const service = new Service("AlbService", {
        cluster,
        image: { context: "." },
        loadBalancer: {
          instance: alb,
          rules: [
            {
              listen: "80/http",
              forward: "3000/http",
              conditions: { path: "/api/*" },
              priority: 100,
            },
          ],
        },
      });

      // Service should use the ALB's URL
      pulumi.all([service.url]).apply(([url]) => {
        expect(url).toBeDefined();
      });
    });

    it("creates service with standard loadBalancer (no ALB instance)", async () => {
      const service = new Service("InlineService", {
        cluster,
        image: { context: "." },
        loadBalancer: {
          ports: [{ listen: "80/http", forward: "3000/http" }],
        },
      });

      // Should still create successfully with inline LB
      expect(service).toBeDefined();
    });

    it("ignores non-Alb instance and falls through to standard loadBalancer", async () => {
      // A plain object with `instance` that is NOT an Alb should not
      // trigger the external ALB path — it falls through to the standard
      // loadBalancer handling (which requires `ports`).
      const fakeAlb = { instance: { notAnAlb: true } };

      const service = new Service("NonAlbService", {
        cluster,
        image: { context: "." },
        loadBalancer: {
          ...(fakeAlb as any),
          ports: [{ listen: "80/http", forward: "3000/http" }],
        },
      });

      expect(service).toBeDefined();
    });
  });

  // Note: Empty rules, out-of-range priorities, duplicate priorities, and
  // missing conditions now throw synchronously. Container name validation
  // still fires inside .apply() (async Pulumi resolution) since container
  // names are resolved asynchronously.

  describe("validation: priority", () => {
    it("creates rules with valid priority", async () => {
      const service = new Service("ValidPriorityService", {
        cluster,
        image: { context: "." },
        loadBalancer: {
          instance: alb,
          rules: [
            {
              listen: "80/http",
              forward: "3000/http",
              conditions: { path: "/valid/*" },
              priority: 100,
            },
          ],
        },
      });

      expect(service).toBeDefined();
    });

    it("creates rules with multiple priorities on same listener", async () => {
      const service = new Service("MultiPriorityService", {
        cluster,
        image: { context: "." },
        loadBalancer: {
          instance: alb,
          rules: [
            {
              listen: "80/http",
              forward: "3000/http",
              conditions: { path: "/api/*" },
              priority: 100,
            },
            {
              listen: "80/http",
              forward: "3000/http",
              conditions: { path: "/health" },
              priority: 200,
            },
          ],
        },
      });

      expect(service).toBeDefined();
    });
  });

  describe("health check defaults", () => {
    it("creates target group without explicit health config", async () => {
      createdResources.length = 0;

      new Service("NoHealthService", {
        cluster,
        image: { context: "." },
        loadBalancer: {
          instance: alb,
          rules: [
            {
              listen: "80/http",
              forward: "3000/http",
              conditions: { path: "/no-health/*" },
              priority: 300,
            },
          ],
        },
      });

      // Target groups are now created synchronously (plain loop, no apply)
      expect(true).toBe(true); // Construction didn't throw
    });

    it("creates target group with explicit health config", async () => {
      const service = new Service("HealthService", {
        cluster,
        image: { context: "." },
        loadBalancer: {
          instance: alb,
          rules: [
            {
              listen: "80/http",
              forward: "3000/http",
              conditions: { path: "/with-health/*" },
              priority: 400,
            },
          ],
          health: {
            "3000/http": {
              path: "/health",
              interval: "30 seconds",
              timeout: "5 seconds",
              healthyThreshold: 3,
              unhealthyThreshold: 2,
              successCodes: "200-299",
            },
          },
        },
      });

      expect(service).toBeDefined();
    });
  });

  describe("multiple containers", () => {
    it("creates service with container field in rules", async () => {
      const service = new Service("MultiContainerService", {
        cluster,
        containers: [
          {
            name: "api",
            image: { context: "./api" },
          },
          {
            name: "worker",
            image: { context: "./worker" },
          },
        ],
        loadBalancer: {
          instance: alb,
          rules: [
            {
              listen: "80/http",
              forward: "3000/http",
              container: "api",
              conditions: { path: "/api/*" },
              priority: 500,
            },
          ],
        },
      });

      expect(service).toBeDefined();
    });
  });

  describe("VPC validation", () => {
    it("verifies ALB and cluster VPCs can be compared", async () => {
      const mismatchedAlb = new Alb("MismatchedAlb", {
        vpc: {
          id: "vpc-999",
          publicSubnets: ["subnet-x"],
          privateSubnets: ["subnet-y"],
        },
        listeners: [{ port: 80, protocol: "http" }],
      });

      // Verify the ALB and cluster expose different VPC IDs.
      // The actual VisibleError is thrown inside .apply() at deploy time
      // and cannot be caught synchronously in unit tests.
      const [albVpcId, clusterVpcId] = await new Promise<[string, string]>(
        (resolve) => {
          pulumi
            .all([mismatchedAlb._vpc, pulumi.output("vpc-123")])
            .apply(([a, b]) => resolve([a, b]));
        },
      );

      expect(albVpcId).toBe("vpc-999");
      expect(clusterVpcId).toBe("vpc-123");
      expect(albVpcId).not.toBe(clusterVpcId);
    });

    it("passes when ALB and cluster share the same VPC", async () => {
      // alb fixture uses vpc-123, same as the cluster
      const service = new Service("SameVpcService", {
        cluster,
        image: { context: "." },
        loadBalancer: {
          instance: alb,
          rules: [
            {
              listen: "80/http",
              forward: "3000/http",
              conditions: { path: "/same-vpc/*" },
              priority: 700,
            },
          ],
        },
      });

      expect(service).toBeDefined();
    });
  });

  describe("Alb.get() with Service", () => {
    it("creates service attached to a referenced ALB", async () => {
      const refAlb = Alb.get(
        "RefAlbForService",
        "arn:aws:elasticloadbalancing:us-east-1:123456789:loadbalancer/app/my-alb/abc123",
      );

      const service = new Service("RefAlbService", {
        cluster,
        image: { context: "." },
        loadBalancer: {
          instance: refAlb,
          rules: [
            {
              listen: "80/http",
              forward: "3000/http",
              conditions: { path: "/ref/*" },
              priority: 600,
            },
          ],
        },
      });

      expect(service).toBeDefined();
    });
  });
});
