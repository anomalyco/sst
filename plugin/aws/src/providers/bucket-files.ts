import { dynamic } from "@pulumi/pulumi";
import { rpc } from "sst-plugin/internal/rpc";
import * as sst from "sst-plugin";

export interface BucketFile {
  source: string;
  key: string;
  cacheControl?: string;
  contentType: string;
  hash?: string;
}

export interface BucketFilesInputs {
  bucketName: sst.Input<string>;
  files: sst.Input<BucketFile[]>;
  purge: sst.Input<boolean>;
  region: sst.Input<string>;
}

export class BucketFiles extends dynamic.Resource {
  constructor(
    name: string,
    args: BucketFilesInputs,
    opts?: sst.ComponentOptions,
  ) {
    super(
      new rpc.Provider("Aws.BucketFiles"),
      `${name}.sst.aws.BucketFiles`,
      args,
      opts,
    );
  }
}
