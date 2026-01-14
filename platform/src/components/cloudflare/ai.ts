import { ComponentResourceOptions } from "@pulumi/pulumi";
import { Component, Transform } from "../component";
import { Link } from "../link";
import { binding } from "./binding";

export interface AiArgs {
  transform?: {
    binding?: Transform<{ name: string }>;
  };
}

export class Ai extends Component implements Link.Linkable {
  constructor(name: string, args?: AiArgs, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);
    this.name = name;
  }

  getSSTLink() {
    return {
      properties: {},
      include: [
        binding({
          type: "aiBinding",
          properties: {
            name: this.name,
          },
        }),
      ],
    };
  }

  name: string;
}

const __pulumiType = "sst:cloudflare:Ai";
// @ts-expect-error
Ai.__pulumiType = __pulumiType;
