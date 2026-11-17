import { beforeEach, describe, expect, it, vi } from "vitest";

const ACCOUNT_TYPE = "aws:apigateway/account:Account";
const ROLE_TYPE = "aws:iam/role:Role";
const EXISTING_ROLE = "arn:aws:iam::111111111111:role/existing";

type Registered = { type: string; name: string; read: boolean };

// Each test loads a fresh copy of the helper and of @pulumi/pulumi, so the
// module-level caches start empty and the mocks bind to the runtime in use.
async function load(accountRoleArn: string) {
  vi.resetModules();
  const registered: Registered[] = [];
  const pulumi = await import("@pulumi/pulumi");
  // @ts-ignore
  global.$app = { name: "app", stage: "test" };
  // @ts-ignore
  global.$util = pulumi;
  pulumi.runtime.setMocks(
    {
      newResource: (args: any) => {
        registered.push({ type: args.type, name: args.name, read: !!args.id });
        if (args.type === ACCOUNT_TYPE && args.id) {
          return {
            id: args.id,
            state: { cloudwatchRoleArn: accountRoleArn },
          };
        }
        return { id: `${args.name}_id`, state: args.inputs };
      },
      call: (args: any) => args.inputs,
    },
    "project",
    "stack",
    false,
  );
  const aws = await import("@pulumi/aws");
  const { setupApiGatewayAccount } = await import(
    "../../src/components/aws/helpers/apigateway-account"
  );
  return { pulumi, aws, setupApiGatewayAccount, registered };
}

function settle(value: any) {
  return new Promise<void>((resolve) =>
    value.apply((resource: any) => {
      resource.urn.apply(() => resolve());
      return resource;
    }),
  );
}

const reads = (registered: Registered[]) =>
  registered.filter((r) => r.type === ACCOUNT_TYPE && r.read);
const setups = (registered: Registered[]) =>
  registered.filter((r) => r.type === ACCOUNT_TYPE && !r.read);
const roles = (registered: Registered[]) =>
  registered.filter((r) => r.type === ROLE_TYPE);

describe("setupApiGatewayAccount", () => {
  let gateways: string[];

  beforeEach(() => {
    gateways = ["GatewayA", "GatewayB", "GatewayC"];
  });

  it("reads the account once per provider, under the first gateway's name", async () => {
    const { setupApiGatewayAccount, registered } = await load(EXISTING_ROLE);

    const results = gateways.map((name) => setupApiGatewayAccount(name, {}));
    await Promise.all(results.map(settle));

    expect(reads(registered).map((r) => r.name)).toEqual([
      "GatewayAAPIGatewayAccount",
    ]);
    expect(setups(registered)).toHaveLength(0);
    expect(roles(registered)).toHaveLength(0);
  });

  it("reads once for each distinct provider, with unique names", async () => {
    const { aws, setupApiGatewayAccount, registered } =
      await load(EXISTING_ROLE);
    const east = new aws.Provider("east", { region: "us-east-1" });
    const west = new aws.Provider("west", { region: "us-west-2" });

    const results = [
      setupApiGatewayAccount("GatewayA", { provider: east }),
      setupApiGatewayAccount("GatewayB", { provider: west }),
      setupApiGatewayAccount("GatewayC", { provider: east }),
      setupApiGatewayAccount("GatewayD", {}),
    ];
    await Promise.all(results.map(settle));

    expect(
      reads(registered)
        .map((r) => r.name)
        .sort(),
    ).toEqual([
      "GatewayAAPIGatewayAccount",
      "GatewayBAPIGatewayAccount",
      "GatewayDAPIGatewayAccount",
    ]);
  });

  it("keeps one setup per gateway and one shared role when no role is configured", async () => {
    const { setupApiGatewayAccount, registered } = await load("");

    const results = gateways.map((name) => setupApiGatewayAccount(name, {}));
    await Promise.all(results.map(settle));

    expect(reads(registered)).toHaveLength(1);
    expect(
      setups(registered)
        .map((r) => r.name)
        .sort(),
    ).toEqual([
      "GatewayAAPIGatewayAccountSetup",
      "GatewayBAPIGatewayAccountSetup",
      "GatewayCAPIGatewayAccountSetup",
    ]);
    expect(roles(registered).map((r) => r.name)).toEqual([
      "APIGatewayPushToCloudWatchLogsRole",
    ]);
  });

  it("registers only account names the helper already used before the cache", async () => {
    const { setupApiGatewayAccount, registered } = await load("");

    const results = gateways.map((name) => setupApiGatewayAccount(name, {}));
    await Promise.all(results.map(settle));

    const previousNames = new Set(
      gateways.flatMap((name) => [
        `${name}APIGatewayAccount`,
        `${name}APIGatewayAccountSetup`,
      ]),
    );
    for (const { name } of registered.filter((r) => r.type === ACCOUNT_TYPE)) {
      expect(previousNames).toContain(name);
    }
  });
});
