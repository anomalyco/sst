import { dynamic } from "@pulumi/pulumi";
import { rpc } from "sst-plugin/internal/rpc";
import * as sst from "sst-plugin";

export interface DistributionInvalidationInputs {
  distributionId: sst.Input<string>;
  paths: sst.Input<string[]>;
  wait: sst.Input<boolean>;
  version: sst.Input<string>;
}

export class DistributionInvalidation extends dynamic.Resource {
  constructor(
    name: string,
    args: DistributionInvalidationInputs,
    opts?: sst.ComponentOptions,
  ) {
    super(
      new rpc.Provider("Aws.DistributionInvalidation"),
      `${name}.sst.aws.DistributionInvalidation`,
      args,
      opts,
    );
  }
}
