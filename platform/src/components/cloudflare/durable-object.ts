import { ComponentResourceOptions, output, type Output } from "@pulumi/pulumi";
import { Component } from "../component.js";
import { Link } from "../link.js";
import type { Input } from "../input.js";
import { binding } from "./binding.js";

export interface DurableObjectArgs {
  /**
   * The name of the class in your worker handler file that extends `DurableObject`.
   *
   * @example
   * ```js
   * {
   *   className: "Counter"
   * }
   * ```
   */
  className: Input<string>;
}

/**
 * Use the `DurableObject` component to register a
 * [Cloudflare Durable Object](https://developers.cloudflare.com/durable-objects/)
 * for a worker.
 *
 * Create the Durable Object and then link it to a `sst.cloudflare.Worker`. SST
 * adds the Durable Object binding automatically. The `className` must match the
 * exported Durable Object class name in your worker code.
 *
 * Durable Objects require migrations on the worker. Use the Worker's
 * `migrations` field like Wrangler's top-level `migrations` config: keep the
 * full ordered history there, and SST uses it to build the Cloudflare migration
 * payload for the Worker.
 *
 * On the first deploy, add the class with `newSqliteClasses`. If you later
 * rename the class, keep the original migration and add a new migration with a
 * unique tag and `renamedClasses`.
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
 *   migrations: [{
 *     tag: "v1",
 *     newSqliteClasses: [counter.className],
 *   }],
 *   url: true,
 * });
 * ```
 *
 * To rename a deployed class from `Counter` to `CounterV2`, update `className`
 * and the exported class name, then append a migration.
 *
 * ```ts title="sst.config.ts"
 * const counter = new sst.cloudflare.DurableObject("Counter", {
 *   className: "CounterV2",
 * });
 *
 * new sst.cloudflare.Worker("Api", {
 *   handler: "src/worker.ts",
 *   link: [counter],
 *   migrations: [
 *     {
 *       tag: "v1",
 *       newSqliteClasses: ["Counter"],
 *     },
 *     {
 *       tag: "v2",
 *       renamedClasses: [{ from: "Counter", to: counter.className }],
 *     },
 *   ],
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
  /**
   * The exported Durable Object class name.
   */
  public readonly className: Output<string>;

  constructor(
    name: string,
    args: DurableObjectArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);
    this.className = output(args.className);
  }

  /**
   * When you link a Durable Object to a worker, SST adds a Cloudflare Durable
   * Object namespace binding.
   *
   * @internal
   */
  public getSSTLink() {
    const properties = {
      className: this.className,
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
}

const __pulumiType = "sst:cloudflare:DurableObject";
// @ts-expect-error
DurableObject.__pulumiType = __pulumiType;
