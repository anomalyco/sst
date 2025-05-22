import { dynamic } from "@pulumi/pulumi";
import { rpc } from "sst-plugin/internal/rpc";
import * as sst from "sst-plugin";

export interface HostedZoneLookupInputs {
  domain: sst.Input<string>;
}

export interface HostedZoneLookup {
  zoneId: sst.Output<string>;
}

export class HostedZoneLookup extends dynamic.Resource {
  constructor(
    name: string,
    args: HostedZoneLookupInputs,
    opts?: sst.ComponentOptions,
  ) {
    super(
      new rpc.Provider("Aws.HostedZoneLookup"),
      `${name}.sst.aws.HostedZoneLookup`,
      { ...args, zoneId: undefined },
      opts,
    );
  }
}
