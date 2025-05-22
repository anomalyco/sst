import { dynamic } from "@pulumi/pulumi";
import { rpc } from "sst-plugin/internal/rpc";
import * as sst from "sst-plugin";

export interface KvRoutesUpdateInputs {
  store: sst.Input<string>;
  key: sst.Input<string>;
  entry: sst.Input<string>;
  namespace: sst.Input<string>;
}

export class KvRoutesUpdate extends dynamic.Resource {
  constructor(
    name: string,
    args: KvRoutesUpdateInputs,
    opts?: sst.ComponentOptions,
  ) {
    super(
      new rpc.Provider("Aws.KvRoutesUpdate"),
      `${name}.sst.aws.KvRoutesUpdate`,
      args,
      opts,
    );
  }
}
