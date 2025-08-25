import { dynamic } from "@pulumi/pulumi";
import { rpc } from "sst-plugin/internal/rpc";
import * as sst from "sst-plugin";

export interface FunctionEnvironmentUpdateInputs {
  /**
   * The name of the function to update.
   */
  functionName: sst.Input<string>;
  /**
   * The environment variables to update.
   */
  environment: sst.Input<Record<string, sst.Input<string>>>;
  /**
   * The region of the function to update.
   */
  region: sst.Input<string>;
}

/**
 * The `FunctionEnvironmentUpdate` component is internally used by the `Function` component
 * to update the environment variables of a function.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addEnvironment` method of the `Function` component.
 */
export class FunctionEnvironmentUpdate extends dynamic.Resource {
  constructor(
    name: string,
    args: FunctionEnvironmentUpdateInputs,
    opts?: sst.ComponentOptions,
  ) {
    super(
      new rpc.Provider("Aws.FunctionEnvironmentUpdate"),
      `${name}.sst.aws.FunctionEnvironmentUpdate`,
      args,
      opts,
    );
  }
}
