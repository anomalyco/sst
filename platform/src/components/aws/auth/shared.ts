import { ComponentResourceOptions } from "@pulumi/pulumi";
import { Dynamo } from "../dynamo";

export function createAuthTable(authName: string, opts?: ComponentResourceOptions) {
  return new Dynamo(
    `${authName}Storage`,
    {
      fields: { pk: "string", sk: "string" },
      primaryIndex: { hashKey: "pk", rangeKey: "sk" },
      ttl: "expiry",
    },
    opts,
  );
}
