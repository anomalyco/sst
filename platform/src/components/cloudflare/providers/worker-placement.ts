import { CustomResourceOptions, Input, Output, dynamic } from "@pulumi/pulumi";
import { cfFetch } from "../helpers/fetch.js";
import { DEFAULT_ACCOUNT_ID } from "../account-id.js";

interface Inputs {
  accountId: string;
  scriptName: string;
  mode?: string;
  region?: string;
  host?: string;
  hostname?: string;
}

export interface WorkerPlacementInputs {
  accountId?: Input<Inputs["accountId"]>;
  scriptName: Input<Inputs["scriptName"]>;
  mode?: Input<Inputs["mode"]>;
  region?: Input<Inputs["region"]>;
  host?: Input<Inputs["host"]>;
  hostname?: Input<Inputs["hostname"]>;
}

export interface WorkerPlacement {
  scriptName: Output<string>;
}

function buildPlacement(inputs: Inputs) {
  if (inputs.mode) return { mode: inputs.mode };
  if (inputs.region) return { region: inputs.region };
  if (inputs.host) return { host: inputs.host };
  if (inputs.hostname) return { hostname: inputs.hostname };
  return {};
}

class Provider implements dynamic.ResourceProvider {
  async create(inputs: Inputs): Promise<dynamic.CreateResult> {
    await this.patch(inputs, buildPlacement(inputs));
    return {
      id: inputs.scriptName,
      outs: { scriptName: inputs.scriptName },
    };
  }

  async update(
    id: string,
    olds: Inputs,
    news: Inputs,
  ): Promise<dynamic.UpdateResult> {
    await this.patch(news, buildPlacement(news));
    return {
      outs: { scriptName: news.scriptName },
    };
  }

  async delete(id: string, inputs: Inputs) {
    await this.patch(inputs, {});
  }

  async patch(inputs: Inputs, placement: Partial<Record<string, string>>) {
    const accountId = inputs.accountId || DEFAULT_ACCOUNT_ID;
    const formData = new FormData();
    formData.append(
      "settings",
      new Blob([JSON.stringify({ placement })], {
        type: "application/json",
      }),
    );
    await cfFetch(
      `/accounts/${accountId}/workers/scripts/${inputs.scriptName}/settings`,
      {
        method: "PATCH",
        body: formData,
      },
    );
  }
}

export class WorkerPlacement extends dynamic.Resource {
  constructor(
    name: string,
    args: WorkerPlacementInputs,
    opts?: CustomResourceOptions,
  ) {
    super(
      new Provider(),
      `${name}.sst.cloudflare.WorkerPlacement`,
      { ...args, scriptName: args.scriptName },
      opts,
    );
  }
}
