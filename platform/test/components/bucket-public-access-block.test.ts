import { beforeAll, describe, expect, it, vi } from "vitest";
import * as pulumi from "@pulumi/pulumi";

// @ts-ignore
global.$app = { name: "app", stage: "test" };
global.$util = pulumi;

const retention = new Map<string, boolean | undefined>();
const resources = new Map<string, pulumi.runtime.MockResourceArgs>();

function resolveOutput<T>(value: pulumi.Output<T>) {
  return new Promise<T>((resolve) => {
    value.apply((resolved) => {
      resolve(resolved);
      return resolved;
    });
  });
}

describe("Bucket public access block lifecycle", () => {
  let Bucket: typeof import("../../src/components/aws/bucket").Bucket;

  beforeAll(async () => {
    await pulumi.runtime.setMocks(
      {
        newResource(args) {
          resources.set(args.name, args);
          return { id: `${args.name}_id`, state: args.inputs };
        },
        call(args) {
          return args.inputs;
        },
      },
      "project",
      "stack",
      false,
    );
    await pulumi.runtime.runInPulumiStack(async () => {
      pulumi.runtime.registerStackTransformation((args) => {
        retention.set(args.name, args.opts.retainOnDelete);
        return undefined;
      });
      Bucket = (await import("../../src/components/aws/bucket")).Bucket;
      return {};
    });
  });

  it.each([false, true])(
    "retains the block without changing bucket retention %s",
    async (retainOnDelete) => {
      const name = retainOnDelete ? "RetainedBucket" : "RemovedBucket";
      const bucket = new Bucket(name, {}, { retainOnDelete });
      await resolveOutput(bucket.name);

      expect(retention.get(`${name}PublicAccessBlock`)).toBe(true);
      expect(retention.get(`${name}Bucket`)).toBe(retainOnDelete);
    },
  );

  it("retains the block when no parent retention option is provided", async () => {
    const bucket = new Bucket("DefaultBucket");
    await resolveOutput(bucket.name);

    expect(retention.get("DefaultBucketPublicAccessBlock")).toBe(true);
    expect(retention.get("DefaultBucketBucket")).toBeUndefined();
  });

  it("lets a transform explicitly opt into deleting the block", async () => {
    const bucket = new Bucket(
      "OverrideBucket",
      {
        transform: {
          publicAccessBlock: (_args, opts) => {
            opts.retainOnDelete = false;
          },
        },
      },
      { retainOnDelete: true },
    );
    await resolveOutput(bucket.name);

    expect(retention.get("OverrideBucketPublicAccessBlock")).toBe(false);
    expect(retention.get("OverrideBucketBucket")).toBe(true);
  });

  it.each(["cloudfront", "public"] as const)(
    "preserves the public access block settings for %s access",
    async (access) => {
      const name = access === "public" ? "PublicBucket" : "CloudfrontBucket";
      const bucket = new Bucket(name, { access });
      await resolveOutput(bucket.name);
      await vi.waitFor(() => {
        expect(resources.has(`${name}PublicAccessBlock`)).toBe(true);
      });

      expect(resources.get(`${name}PublicAccessBlock`)?.inputs).toMatchObject({
        blockPublicAcls: true,
        blockPublicPolicy: access !== "public",
        ignorePublicAcls: true,
        restrictPublicBuckets: access !== "public",
      });
      expect(retention.get(`${name}PublicAccessBlock`)).toBe(true);
    },
  );
});
