import { beforeAll, describe, expect, it } from "vitest";
import * as pulumi from "@pulumi/pulumi";

// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
};
// @ts-ignore
global.$util = pulumi;

pulumi.runtime.setMocks(
  {
    newResource: function (args: pulumi.runtime.MockResourceArgs): {
      id: string;
      state: any;
    } {
      return {
        id: args.name + "_id",
        state: args.inputs,
      };
    },
    call: function (args: pulumi.runtime.MockCallArgs) {
      return args.inputs;
    },
  },
  "project",
  "stack",
  false,
);

describe("Link", () => {
  let Secret: typeof import("../../src/components/secret").Secret;
  let Link: typeof import("../../src/components/link").Link;
  let Linkable: typeof import("../../src/components/linkable").Linkable;

  async function resolveOutput<T>(value: pulumi.Output<T>) {
    // Observe rejected values and metadata so invalid links fail instead of hanging.
    const result = value as pulumi.Output<T> & {
      promise(): Promise<T>;
      isKnown: Promise<boolean>;
      isSecret: Promise<boolean>;
    };
    const [resolved] = await Promise.all([
      result.promise(),
      result.isKnown,
      result.isSecret,
    ]);
    return resolved;
  }

  beforeAll(async () => {
    Secret = (await import("../../src/components/secret")).Secret;
    Link = (await import("../../src/components/link")).Link;
    Linkable = (await import("../../src/components/linkable")).Linkable;
  });

  it.each([
    null,
    false,
    42,
    "Uploads",
    {},
    { getSSTLink: true, urn: pulumi.output("urn") },
    { getSSTLink: () => ({ properties: {} }) },
    { getSSTLink: () => ({ properties: {} }), urn: "urn" },
  ])(
    "rejects invalid explicit links without breaking discovery: %j",
    (link) => {
      expect(Link.isLinkable(link)).toBe(false);
      expect(() => Link.build([link])).toThrow(
        "An invalid resource was passed into a `link` array.",
      );
    },
  );

  it("preserves the undefined-link error", () => {
    expect(() => Link.build([undefined])).toThrow(
      "An undefined link was passed into a `link` array.",
    );
  });

  it("rejects an unwrapped Pulumi resource mixed with valid links", () => {
    const secret = new Secret("MixedSecret", "test");
    const resource = new pulumi.ComponentResource("test:index:Resource", "Raw");

    expect(Link.isLinkable(resource)).toBe(false);
    expect(() => Link.build([secret, resource])).toThrow("sst.Linkable.wrap()");
  });

  it("rejects invalid links when resolving properties", async () => {
    await expect(
      resolveOutput(Link.getProperties(Promise.resolve([{}]))),
    ).rejects.toThrow("An invalid resource was passed into a `link` array.");
  });

  it("rejects invalid links when extracting permissions", async () => {
    await expect(
      resolveOutput(Link.getInclude("aws.permission", Promise.resolve([{}]))),
    ).rejects.toThrow("An invalid resource was passed into a `link` array.");
  });

  it("normalizes type in build output", async () => {
    const secret = new Secret("MySecret", "test");
    const built = await resolveOutput(pulumi.output(Link.build([secret])));

    expect(built[0].name).toBe("MySecret");
    expect(built[0].properties.type).toBe("sst.sst.Secret");
  });

  it("normalizes type in env properties", async () => {
    const secret = new Secret("MyOtherSecret", "test");
    const properties = await resolveOutput(Link.getProperties([secret]));

    expect(properties.MyOtherSecret.type).toBe("sst.sst.Secret");
  });

  it("keeps valid links without matching includes and extracts permissions", async () => {
    const secret = new Secret("PermissionSecret", "test");
    const permission = {
      type: "aws.permission",
      actions: ["s3:GetObject"],
      resources: ["arn:aws:s3:::uploads/*"],
    };
    const storage = new Linkable("Storage", {
      properties: { name: "uploads" },
      include: [permission, { type: "environment", env: { EXAMPLE: "value" } }],
    });

    expect(
      await resolveOutput(Link.getInclude("aws.permission", [secret, storage])),
    ).toEqual([permission]);
    expect(await resolveOutput(Link.getProperties([secret, storage]))).toEqual({
      PermissionSecret: { type: "sst.sst.Secret", value: "test" },
      Storage: { type: "sst.sst.Linkable", name: "uploads" },
    });
  });
});
