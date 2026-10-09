import { CustomResourceOptions, Input, Output, dynamic } from "@pulumi/pulumi";
import { rpc } from "../../rpc/rpc.js";

export interface CodeDeployDeploymentWaiterInputs {
  deploymentId: Input<string>;
  wait: Input<boolean>;
  trigger: Input<string>;
}

export interface CodeDeployDeploymentWaiter {
  deploymentId: Output<string>;
  status: Output<string>;
}

export class CodeDeployDeploymentWaiter extends dynamic.Resource {
  constructor(
    name: string,
    args: CodeDeployDeploymentWaiterInputs,
    opts?: CustomResourceOptions,
  ) {
    super(
      new rpc.Provider("Aws.CodeDeployDeploymentWaiter"),
      `${name}.sst.aws.CodeDeployDeploymentWaiter`,
      { ...args, status: undefined },
      opts,
    );
  }
}
