import type { Input } from "../input.js";
import { Component } from "../component.js";
import type { Link } from "../link.js";
import {
  LINK_NAME_PLACEHOLDER,
  type DeepInput,
} from "./helpers/wrangler-config.js";
import { binding, type CloudflareBinding } from "./binding.js";
import type { Unstable_RawConfig as RawConfig } from "wrangler";

export abstract class CloudflareComponent
  extends Component
  implements Link.Linkable
{
  protected readonly binding?: CloudflareBinding;
  protected abstract readonly type: Input<string>;
  protected devConfig?: DeepInput<Partial<RawConfig>>;
  protected readonly linkNamePlaceholder = LINK_NAME_PLACEHOLDER;

  protected abstract getLinkDefinition(): Link.Definition;

  public readonly getSSTLink = (): Link.Definition => {
    const link = this.getLinkDefinition();
    return {
      ...link,
      include: [
        ...(link.include ?? []),
        ...(this.binding ? [binding(this.binding)] : []),
        { type: "typescript.type", value: this.type },
        ...(this.devConfig
          ? [{ type: "cloudflare.dev", config: this.devConfig }]
          : []),
      ],
    };
  };
}
