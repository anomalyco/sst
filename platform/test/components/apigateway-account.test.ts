import { beforeAll, describe, expect, it } from "vitest";
import * as pulumi from "@pulumi/pulumi";

// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
};
// @ts-ignore
global.$util = pulumi;

const ACCOUNT_TYPE = "aws:apigateway/account:Account";
const registered: { type: string; name: string }[] = [];

pulumi.runtime.setMocks(
  {
    newResource: function (args: pulumi.runtime.MockResourceArgs): {
      id: string;
      state: any;
    } {
      registered.push({ type: args.type, name: args.name });
      if (args.type === ACCOUNT_TYPE) {
        return {
          id: "APIGatewayAccount",
          state: {
            ...args.inputs,
            cloudwatchRoleArn: "arn:aws:iam::111111111111:role/existing",
          },
        };
      }
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

describe("setupApiGatewayAccount", () => {
  let setupApiGatewayAccount: typeof import("../../src/components/aws/helpers/apigateway-account").setupApiGatewayAccount;
  let aws: typeof import("@pulumi/aws");

  function resolveOutput<T>(value: pulumi.Output<T>) {
    return new Promise<T>((resolve) => {
      value.apply((resolved) => {
        resolve(resolved);
        return resolved;
      });
    });
  }

  function accountReads() {
    return registered.filter((r) => r.type === ACCOUNT_TYPE);
  }

  beforeAll(async () => {
    setupApiGatewayAccount = (
      await import("../../src/components/aws/helpers/apigateway-account")
    ).setupApiGatewayAccount;
    aws = await import("@pulumi/aws");
  });

  it("returns the same account for every gateway on the same provider", async () => {
    const first = setupApiGatewayAccount("GatewayA", {});
    const second = setupApiGatewayAccount("GatewayB", {});

    expect(second).toBe(first);

    await resolveOutput(first);
    await resolveOutput(second);
    expect(accountReads()).toHaveLength(1);
    expect(accountReads()[0].name).toBe("APIGatewayAccount");
  });

  it("creates one uniquely named read per distinct provider", async () => {
    const east = new aws.Provider("east", { region: "us-east-1" });
    const west = new aws.Provider("west", { region: "us-west-2" });

    const eastAccount = setupApiGatewayAccount("GatewayC", { provider: east });
    const westAccount = setupApiGatewayAccount("GatewayD", { provider: west });
    const eastAgain = setupApiGatewayAccount("GatewayE", { provider: east });

    expect(eastAgain).toBe(eastAccount);
    expect(westAccount).not.toBe(eastAccount);

    await resolveOutput(eastAccount);
    await resolveOutput(westAccount);
    const reads = accountReads();
    expect(reads).toHaveLength(3);
    expect(new Set(reads.map((r) => r.name)).size).toBe(3);
  });
});
