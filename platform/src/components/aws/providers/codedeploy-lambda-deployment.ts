import { CustomResourceOptions, Input, Output, dynamic } from "@pulumi/pulumi";
import { rpc } from "../../rpc/rpc.js";

export interface CodeDeployLambdaDeploymentInputs {
  applicationName: Input<string>;
  deploymentGroupName: Input<string>;
  functionName: Input<string>;
  aliasName: Input<string>;
  targetVersion: Input<string>;
  onConflict: Input<"rollback" | "cancel" | "fail">;
  beforeTrafficFnArn?: Input<string>;
  afterTrafficFnArn?: Input<string>;
}

export interface CodeDeployLambdaDeployment {
  deploymentId: Output<string>;
}

export class CodeDeployLambdaDeployment extends dynamic.Resource {
  constructor(
    name: string,
    args: CodeDeployLambdaDeploymentInputs,
    opts?: CustomResourceOptions,
  ) {
    super(
      new rpc.Provider("Aws.CodeDeployLambdaDeployment"),
      `${name}.sst.aws.CodeDeployLambdaDeployment`,
      { ...args, deploymentId: undefined },
      opts,
    );
  }
}
