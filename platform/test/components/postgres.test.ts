import { beforeAll, beforeEach, describe, expect, it } from "vitest";
import "../../src/global.d.ts";
import * as pulumi from "@pulumi/pulumi";

const resources: pulumi.runtime.MockResourceArgs[] = [];

// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
};
global.$util = pulumi;
// @ts-ignore
global.$cli = { state: { version: {} } };
// @ts-ignore
global.$dev = false;

pulumi.runtime.setMocks(
  {
    newResource: (args: pulumi.runtime.MockResourceArgs) => {
      resources.push(args);
      return {
        id: `${args.name}_id`,
        state: args.inputs,
      };
    },
    call: (args: pulumi.runtime.MockCallArgs) => args.inputs,
  },
  "project",
  "stack",
  false,
);

describe("Postgres", () => {
  let Postgres: typeof import("../../src/components/aws/postgres").Postgres;

  beforeAll(async () => {
    Postgres = (await import("../../src/components/aws/postgres")).Postgres;
  });

  beforeEach(() => {
    resources.length = 0;
  });

  async function createParameterGroup(version?: string) {
    const postgres = new Postgres("MyPostgres", {
      version,
      vpc: {
        subnets: ["subnet-123"],
      },
    });

    await new Promise<void>((resolve) => {
      pulumi.all([postgres.id]).apply(() => resolve());
    });

    const parameterGroup = resources.find(
      (resource) => resource.type === "aws:rds/parameterGroup:ParameterGroup",
    );

    expect(parameterGroup).toBeDefined();
    return parameterGroup!.inputs;
  }

  it("uses a family-scoped parameter group name when version is pinned", async () => {
    const parameterGroup = await createParameterGroup("17.1");

    expect(parameterGroup.family).toBe("postgres17");
    expect(parameterGroup.name).toBe(
      "app-test-mypostgresparametergroup-postgres17",
    );
  });

  it("uses the major version when the pinned version omits minor", async () => {
    const parameterGroup = await createParameterGroup("16");

    expect(parameterGroup.family).toBe("postgres16");
    expect(parameterGroup.name).toBe(
      "app-test-mypostgresparametergroup-postgres16",
    );
  });

  it("keeps the existing auto-named parameter group when version is not pinned", async () => {
    const parameterGroup = await createParameterGroup();

    expect(parameterGroup.family).toBe("postgres17");
    expect(parameterGroup.name).toMatch(
      /^app-test-mypostgresparametergroup-[a-z]{8}$/,
    );
  });

  it("switches from auto-named to family-scoped when pinning a new major", async () => {
    const unpinned = await createParameterGroup();

    resources.length = 0;

    const pinned = await createParameterGroup("18");

    expect(unpinned.name).toMatch(
      /^app-test-mypostgresparametergroup-[a-z]{8}$/,
    );
    expect(pinned.family).toBe("postgres18");
    expect(pinned.name).toBe("app-test-mypostgresparametergroup-postgres18");
    expect(pinned.name).not.toBe(unpinned.name);
  });
});
