import { jsonStringify, output, type Input } from "@pulumi/pulumi";
import type * as cf from "@pulumi/cloudflare";
import type { Link } from "../../link.js";
import type { CloudflareBinding } from "../binding.js";

export function buildWorkerBinding(
  link: Link.Linkable,
  name: Input<string>,
): cf.types.input.WorkersScriptBinding {
  const item = link.getSSTLink();
  const cloudflareBinding = item.include?.find(
    (include) => include.type === "cloudflare.binding",
  ) as
    | { type: "cloudflare.binding"; binding: CloudflareBinding }
    | undefined;

  return cloudflareBinding
    ? {
        ...cloudflareBinding.binding,
        name,
      }
    : {
        type: "secret_text",
        name: output(name).apply((name) => `SST_RESOURCE_${name}`),
        text: jsonStringify(item.properties),
      };
}
