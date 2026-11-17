import { ComponentResourceOptions } from "@pulumi/pulumi";
import { CloudflareComponent } from "./component.js";

export interface AiArgs {}

/**
 * The `Ai` component lets you add a [Cloudflare Workers AI](https://developers.cloudflare.com/workers-ai/) binding to
 * your app.
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * const ai = new sst.cloudflare.Ai("MyAi");
 * ```
 *
 * #### Link to a worker
 *
 * You can link AI to a worker.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "./index.ts",
 *   link: [ai],
 *   url: true
 * });
 * ```
 *
 * Once linked, you can use the SDK to interact with the AI binding.
 *
 * ```ts title="index.ts" {3}
 * import { Resource } from "sst";
 *
 * const result = await Resource.MyAi.run("@cf/meta/llama-3-8b-instruct", {
 *   prompt: "What is the origin of the phrase 'Hello, World'"
 * });
 * ```
 */
export class Ai extends CloudflareComponent {
  protected readonly type = 'import("@cloudflare/workers-types").Ai';
  protected readonly binding: import("./binding.js").CloudflareBinding;

  constructor(name: string, args?: AiArgs, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);
    this.binding = { type: "ai" };
    this.devConfig = {
      ai: {
        binding: this.linkNamePlaceholder,
        remote: true,
      },
    };
  }

  /**
   * When you link an AI binding, it will be available to the worker and you can
   * interact with it using its [API methods](https://developers.cloudflare.com/workers-ai/).
   *
   * @example
   * ```ts title="index.ts" {3}
   * import { Resource } from "sst";
   *
   * const result = await Resource.MyAi.run("@cf/meta/llama-3-8b-instruct", {
   *   prompt: "What is the origin of the phrase 'Hello, World'"
   * });
   * ```
   *
   * @internal
   */
  protected getLinkDefinition() {
    return {
      properties: {},
    };
  }
}

const __pulumiType = "sst:cloudflare:Ai";
// @ts-expect-error
Ai.__pulumiType = __pulumiType;
