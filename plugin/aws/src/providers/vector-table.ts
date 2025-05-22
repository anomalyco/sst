import { dynamic } from "@pulumi/pulumi";
import { rpc } from "sst-plugin/internal/rpc";
import * as sst from "sst-plugin";

export interface PostgresTableInputs {
  clusterArn: sst.Input<string>;
  secretArn: sst.Input<string>;
  databaseName: sst.Input<string>;
  tableName: sst.Input<string>;
  dimension: sst.Input<number>;
}

export class VectorTable extends dynamic.Resource {
  constructor(
    name: string,
    args: PostgresTableInputs,
    opts?: sst.ComponentOptions,
  ) {
    super(
      new rpc.Provider("Aws.VectorTable"),
      `${name}.sst.aws.VectorTable`,
      args,
      opts,
    );
  }
}
