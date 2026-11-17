import { describe, expect, it, vi } from "vitest";
import * as pulumi from "@pulumi/pulumi";

vi.mock(
  "../../src/components/aws/providers/distribution-deployment-waiter.js",
  () => ({
    DistributionDeploymentWaiter: class {
      isDone = pulumi.output(true);
    },
  }),
);

// @ts-ignore
global.$app = { name: "app", stage: "test" };
global.$util = pulumi;

pulumi.runtime.setMocks(
  {
    newResource: (args: pulumi.runtime.MockResourceArgs) => ({
      id: `${args.inputs.name}_id`,
      state: {
        ...args.inputs,
        domainName: `${args.inputs.name}.cloudfront.net`,
        hostedZoneId: "Z2FDTNDATAQYW2",
        aliases: args.inputs.aliases ?? [],
        etag: "etag",
      },
    }),
    call: (args: pulumi.runtime.MockCallArgs) => args.inputs,
  },
  "project",
  "stack",
  false,
);

describe("Cdn", () => {
  it("overrides the default behavior cache policy", async () => {
    const { Cdn } = await import("../../src/components/aws/cdn");
    const cdn = new Cdn("TestCdn", {
      origins: [
        {
          originId: "default",
          domainName: "example.com",
          customOriginConfig: {
            httpPort: 80,
            httpsPort: 443,
            originProtocolPolicy: "https-only",
            originSslProtocols: ["TLSv1.2"],
          },
        },
      ],
      cachePolicy: "managed-policy",
      defaultCacheBehavior: {
        targetOriginId: "default",
        viewerProtocolPolicy: "redirect-to-https",
        allowedMethods: ["GET", "HEAD"],
        cachedMethods: ["GET", "HEAD"],
        forwardedValues: {
          queryString: true,
          cookies: {
            forward: "none",
          },
        },
        defaultTtl: 60,
        cachePolicyId: "sst-policy",
      },
    });

    await new Promise<void>((resolve, reject) => {
      cdn.nodes.distribution.apply((distribution) => {
        pulumi.output(distribution.defaultCacheBehavior).apply((behavior) => {
          try {
            expect(behavior).toMatchObject({
              targetOriginId: "default",
              cachePolicyId: "managed-policy",
            });
            expect(behavior.forwardedValues).toBeUndefined();
            expect(behavior.defaultTtl).toBeUndefined();
            resolve();
          } catch (error) {
            reject(error);
          }
        });
      });
    });
  });
});
