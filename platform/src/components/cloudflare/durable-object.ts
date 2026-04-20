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
  /**
   * The storage backend for the initial migration.
   * @default `"sqlite"`
   */
  storage?: Input<"sqlite" | "kv">;
  /**
   * The migration tag to apply when the Durable Object is first linked into a worker.
   * @default `"v1"`
   */
  migrationTag?: Input<string>;
}

/**
 * Use the `DurableObject` component to register a
 * [Cloudflare Durable Object](https://developers.cloudflare.com/durable-objects/)
 * for a worker.
 *
 * Create the Durable Object and then link it to a `sst.cloudflare.Worker`. SST
 * adds the Durable Object binding and initial migration automatically.
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
    private args: DurableObjectArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);
  }

  /**
   * When you link a Durable Object to a worker, SST adds a Cloudflare Durable
   * Object namespace binding and the initial migration.
   *
   * @internal
   */
  getSSTLink() {
    return {
      properties: {
        className: this.args.className,
      },
      include: [
        binding({
          type: "durableObjectNamespaceBindings",
          properties: {
            className: this.args.className,
          },
        }),
        {
          type: "cloudflare.durableObject",
          className: this.args.className,
          migrationTag: this.args.migrationTag ?? "v1",
          storage: this.args.storage ?? "sqlite",
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

  /**
   * The storage backend for the Durable Object.
   */
  public get storage() {
    return output(this.args.storage ?? "sqlite");
  }

  /**
   * The migration tag used for the initial migration.
   */
  public get migrationTag() {
    return output(this.args.migrationTag ?? "v1");
  }
}

const __pulumiType = "sst:cloudflare:DurableObject";
// @ts-expect-error
DurableObject.__pulumiType = __pulumiType;
