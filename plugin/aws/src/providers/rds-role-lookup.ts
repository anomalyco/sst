import { dynamic } from "@pulumi/pulumi";
import { rpc } from "sst-plugin/internal/rpc";
import * as sst from "sst-plugin";

export interface RdsRoleLookupInputs {
  name: sst.Input<string>;
}

export class RdsRoleLookup extends dynamic.Resource {
  constructor(
    name: string,
    args: RdsRoleLookupInputs,
    opts?: sst.ComponentOptions,
  ) {
    super(
      new rpc.Provider("Aws.RdsRoleLookup"),
      `${name}.sst.aws.RdsRoleLookup`,
      args,
      opts,
    );
  }
}
