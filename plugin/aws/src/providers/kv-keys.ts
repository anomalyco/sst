import { dynamic } from "@pulumi/pulumi";
import { rpc } from "sst-plugin/internal/rpc";
import * as sst from "sst-plugin";

export interface KvKeysInputs {
  store: sst.Input<string>;
  namespace: sst.Input<string>;
  entries: sst.Input<Record<string, sst.Input<string>>>;
  purge: sst.Input<boolean>;
}

export class KvKeys extends dynamic.Resource {
  constructor(name: string, args: KvKeysInputs, opts?: sst.ComponentOptions) {
    super(new rpc.Provider("Aws.KvKeys"), `${name}.sst.aws.KvKeys`, args, opts);
  }
}
