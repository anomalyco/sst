import { ComponentResourceOptions, output } from "@pulumi/pulumi";
import { Component } from "../component.js";
import type { Input } from "../input.js";
import { Link } from "../link.js";
import { binding } from "./binding.js";

export interface DurableObjectArgs {
  /**
   * The exported Durable Object class name.
   */
  className: Input<string>;
}

/**
 * Use the `DurableObject` component to register a
 * [Cloudflare Durable Object](https://developers.cloudflare.com/durable-objects/)
 * for a worker.
 *
 * Create the Durable Object and then link it to a `sst.cloudflare.Worker`. SST
 * adds the Durable Object binding automatically. Configure migrations on the
 * worker, similar to Wrangler.
 *
 * @example
 *
 * ```ts title="sst.config.ts"
 * const counter = new sst.cloudflare.DurableObject("Counter", {
 *   className: "Counter",
 * });
 *
 * new sst.cloudflare.Worker("Api", {
 *   handler: "src/worker.ts",
 *   link: [counter],
 *   durableObjectMigrations: [{
 *     tag: "v1",
 *     newSqliteClasses: ["Counter"],
 *   }],
 *   url: true,
 * });
 * ```
 *
 * ```ts title="src/worker.ts"
 * import { Resource } from "sst";
 * import { DurableObject } from "cloudflare:workers";
 *
 * export default {
 *   async fetch() {
 *     const stub = Resource.Counter.getByName("global");
 *     return stub.fetch("https://counter/");
 *   },
 * };
 *
 * export class Counter extends DurableObject {
 *   async fetch() {
 *     return new Response("hello from the durable object");
 *   }
 * }
 * ```
 */
export class DurableObject extends Component implements Link.Linkable {
  constructor(
    name: string,
    private readonly args: DurableObjectArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);
  }

  /**
   * When you link a Durable Object to a worker, SST adds a Cloudflare Durable
   * Object namespace binding.
   *
   * @internal
   */
  public getSSTLink() {
    const properties = {
      className: this.args.className,
    };

    return {
      properties,
      include: [
        binding({
          type: "durableObjectNamespaceBindings",
          properties,
        }),
        {
          type: "cloudflare.durableObject",
          ...properties,
        },
      ],
    };
  }

  /**
   * The exported Durable Object class name.
   */
  public get className() {
    return output(this.args.className);
  }
}

const __pulumiType = "sst:cloudflare:DurableObject";
// @ts-expect-error
DurableObject.__pulumiType = __pulumiType;
