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

  function resolveOutput<T>(value: pulumi.Output<T>) {
    return new Promise<T>((resolve) => {
      value.apply((resolved) => {
        resolve(resolved);
        return resolved;
      });
    });
  }

  beforeAll(async () => {
    Secret = (await import("../../src/components/secret")).Secret;
    Link = (await import("../../src/components/link")).Link;
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
});
