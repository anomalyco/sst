import { dynamic } from "@pulumi/pulumi";
import { rpc } from "sst-plugin/internal/rpc";
import * as sst from "sst-plugin";

export interface DistributionDeploymentWaiterInputs {
  distributionId: sst.Input<string>;
  etag: sst.Input<string>;
  wait: sst.Input<boolean>;
}

export interface DistributionDeploymentWaiter {
  isDone: sst.Output<boolean>;
}

export class DistributionDeploymentWaiter extends dynamic.Resource {
  constructor(
    name: string,
    args: DistributionDeploymentWaiterInputs,
    opts?: sst.ComponentOptions,
  ) {
    super(
      new rpc.Provider("Aws.DistributionDeploymentWaiter"),
      `${name}.sst.aws.DistributionDeploymentWaiter`,
      args,
      opts,
    );
  }
}
