import { dynamic } from "@pulumi/pulumi";
import { rpc } from "sst-plugin/internal/rpc";
import * as sst from "sst-plugin";

export interface OriginAccessControlInputs {
  name: sst.Input<string>;
}

export class OriginAccessControl extends dynamic.Resource {
  constructor(
    name: string,
    args: OriginAccessControlInputs,
    opts?: sst.ComponentOptions,
  ) {
    super(
      new rpc.Provider("Aws.OriginAccessControl"),
      `${name}.sst.aws.OriginAccessControl`,
      args,
      opts,
    );
  }
}
