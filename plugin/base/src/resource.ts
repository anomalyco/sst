import { CustomResource, Output } from "@pulumi/pulumi";

class DynamicResource extends CustomResource {
  declare outputs: Output<Record<string, any>>;
  constructor(
    public readonly type: string,
    name: string,
    source: any,
    inputs: {},
  ) {
    super(
      "sst:index:Dynamic",
      name,
      {
        type,
        source,
        inputs,
        outputs: undefined,
      },
      {
        pluginDownloadURL: "github://api.github.com/sst/pulumi-sst",
        version: "0.0.3",
      },
    );
  }
}

interface Definition<Inputs = any, Outputs = any> {
  type: string;
  create: (inputs: Inputs) => Promise<{
    id: string;
    outputs: Outputs;
  }>;
  update?: (
    name: string,
    state: { inputs: Inputs; outputs: Outputs },
    inputs: Inputs,
  ) => Promise<Outputs>;
  delete?: (
    name: string,
    state: { inputs: Inputs; outputs: Outputs },
  ) => Promise<void>;
}

export const definitions: Record<string, Definition> = {};

export function resource<Inputs, Outputs>(
  def: Definition<Inputs, Outputs>,
): new (
  name: string,
  inputs: Inputs,
) => {
  id: Output<string>;
  type: string;
} & {
  [K in keyof Outputs]: Output<Outputs[K]>;
} {
  definitions[def.type] = def;
  return class {
    constructor(name: string, inputs: Inputs) {
      // @ts-ignore
      const res = new DynamicResource(def.type, name, def["__source"], inputs);
      return new Proxy(res, {
        get(target, prop) {
          if (prop in target) {
            return (target as any)[prop];
          }
          return res.outputs[prop as string];
        },
      });
    }
    static definition = def;
  } as any;
}

resource({
  type: "Test",
  async create(inputs: { test: string }) {
    return {
      id: "test",
      outputs: {
        updated: Date.now(),
      },
    };
  },
  async update(id, olds, news) {
    return {
      updated: Date.now(),
    };
  },
});
