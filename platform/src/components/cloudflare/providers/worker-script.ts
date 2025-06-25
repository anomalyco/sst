import { CustomResourceOptions, Output, dynamic } from "@pulumi/pulumi";
import { rpc } from "../../rpc/rpc.js";
import { DEFAULT_ACCOUNT_ID } from "../account-id.js";
import { WorkersScriptArgs } from "@pulumi/cloudflare";
import { Input } from "../../input.js";

export interface WorkerScriptInputs extends Omit<WorkersScriptArgs, "content"> {
  content: Input<{
    filename: Input<string>;
    hash: Input<string>;
  }>;
}

export interface WorkerScript {
  scriptName: Output<string>;
}

export class WorkerScript extends dynamic.Resource {
  constructor(
    name: string,
    args: WorkerScriptInputs,
    opts?: CustomResourceOptions,
  ) {
    super(
      new rpc.Provider("Cloudflare.WorkerScript"),
      `${name}.sst.cloudflare.WorkerScript`,
      {
        ...args,
        accountId: DEFAULT_ACCOUNT_ID,
        apiToken:
          $app.providers?.cloudflare?.apiToken ||
          process.env.CLOUDFLARE_API_TOKEN!,
      },
      {
        ...opts,
        replaceOnChanges: ["scriptName"],
      },
    );
  }
}
