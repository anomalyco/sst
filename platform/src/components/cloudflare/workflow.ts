import { ComponentResourceOptions } from "@pulumi/pulumi";
import * as cf from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component";
import { Link } from "../link";
import { Input } from "../input";
import { binding } from "./binding";
import { DEFAULT_ACCOUNT_ID } from "./account-id";
import { Worker, WorkerArgs } from "./worker";

export interface WorkflowArgs {
  /**
   * Path to the handler file that exports the class that extends `WorkflowEntrypoint`.
   *
   * The handler path is relative to the root of your repo or the `sst.config.ts`.
   *
   * @example
   *
   * ```js
   * {
   *   handler: "src/workflow.ts"
   * }
   * ```
   */
  handler: Input<string>;
  /**
   * The name of the class in your handler file that extends `WorkflowEntrypoint`.
   *
   * Cloudflare Workflows are defined as classes that extend the
   * [`WorkflowEntrypoint`](https://developers.cloudflare.com/workflows/build/workers-api/#workflowentrypoint)
   * class. You must specify the name of the class so SST can bind to it.
   *
   * :::caution
   * The class must be exported as a **named export**, not as `export default`.
   * :::
   *
   * @example
   *
   * Given this handler file:
   *
   * ```ts title="src/workflow.ts"
   * import { WorkflowEntrypoint } from "cloudflare:workers";
   *
   * export class OrderProcessor extends WorkflowEntrypoint<Env, Params> {
   *   async run(event, step) {
   *     // ...
   *   }
   * }
   * ```
   *
   * You would set:
   *
   * ```js
   * {
   *   className: "OrderProcessor"
   * }
   * ```
   */
  className: Input<string>;
  /**
   * [Link resources](/docs/linking/) to your workflow. This will:
   *
   * 1. Make them available on `this.env` inside your workflow class.
   * 2. Allow you to access them in your workflow using the [SDK](/docs/reference/sdk/).
   *
   * @example
   *
   * Takes a list of components to link to the workflow.
   *
   * ```js
   * {
   *   link: [bucket, db]
   * }
   * ```
   */
  link?: Input<any[]>;
  /**
   * Key-value pairs that are set as environment variables on the workflow's Worker.
   *
   * They can be accessed in your workflow through `this.env.<key>`.
   *
   * @example
   *
   * ```js
   * {
   *   environment: {
   *     DEBUG: "true"
   *   }
   * }
   * ```
   */
  environment?: Input<Record<string, Input<string>>>;
  /**
   * [Transform](/docs/components/#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the underlying Worker that hosts the workflow class.
     */
    worker?: Transform<WorkerArgs>;
    /**
     * Transform the Workflow resource.
     */
    workflow?: Transform<cf.WorkflowArgs>;
  };
}

/**
 * The `Workflow` component lets you add [Cloudflare Workflows](https://developers.cloudflare.com/workflows/)
 * to your app.
 *
 * :::caution
 * Workflow `console.log` output doesn't stream in real time during `sst dev`.
 * :::
 *
 * A Workflow is a durable, multi-step function that runs on Cloudflare Workers. You
 * define it as a class that extends `WorkflowEntrypoint` and pass it to this component.
 * SST will deploy it as a Worker script, register it as a Workflow, and wire up the
 * binding so you can trigger it from any Worker you link it to.
 *
 * @example
 * #### Minimal example
 *
 * Define a workflow class:
 *
 * ```ts title="src/workflow.ts"
 * import { WorkflowEntrypoint, WorkflowEvent, WorkflowStep } from "cloudflare:workers";
 *
 * export class OrderProcessor extends WorkflowEntrypoint<Env, { orderId: string }> {
 *   async run(event: WorkflowEvent<{ orderId: string }>, step: WorkflowStep) {
 *     await step.do("validate", async () => {});
 *     await step.sleep("cooldown", "10 seconds");
 *     await step.do("charge", async () => {});
 *   }
 * }
 * ```
 *
 * Then register it with SST:
 *
 * ```ts title="sst.config.ts"
 * const processor = new sst.cloudflare.Workflow("OrderProcessor", {
 *   handler: "src/workflow.ts",
 *   className: "OrderProcessor",
 * });
 * ```
 *
 * #### Trigger from a Worker
 *
 * Link the workflow to a Worker to get a native Cloudflare
 * [`Workflow`](https://developers.cloudflare.com/workflows/build/workers-api/#workflow)
 * binding:
 *
 * ```ts title="sst.config.ts" {4}
 * new sst.cloudflare.Worker("Api", {
 *   handler: "src/api.ts",
 *   url: true,
 *   link: [processor],
 * });
 * ```
 *
 * Then invoke it from your worker code:
 *
 * ```ts title="src/api.ts"
 * import { Resource } from "sst";
 *
 * export default {
 *   async fetch(req: Request) {
 *     const instance = await Resource.OrderProcessor.create({
 *       params: { orderId: "ord_123" },
 *     });
 *     return Response.json({ id: instance.id });
 *   },
 * };
 * ```
 *
 * #### Give the workflow access to other resources
 *
 * ```ts title="sst.config.ts"
 * const bucket = new sst.cloudflare.Bucket("Orders");
 *
 * const processor = new sst.cloudflare.Workflow("OrderProcessor", {
 *   handler: "src/workflow.ts",
 *   className: "OrderProcessor",
 *   link: [bucket],
 * });
 * ```
 *
 */
export class Workflow extends Component implements Link.Linkable {
  private worker: Worker;
  private workflow: cf.Workflow;

  constructor(
    name: string,
    args: WorkflowArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    const parent = this;

    const worker = createWorker();
    const workflow = createWorkflow();

    this.worker = worker;
    this.workflow = workflow;

    function createWorker() {
      return new Worker(
        ...transform(
          args.transform?.worker,
          `${name}Script`,
          {
            handler: args.handler,
            link: args.link,
            environment: args.environment,
          },
          { parent },
        ),
      );
    }

    function createWorkflow() {
      return new cf.Workflow(
        ...transform(
          args.transform?.workflow,
          `${name}Workflow`,
          {
            accountId: DEFAULT_ACCOUNT_ID,
            workflowName: "",
            className: args.className,
            scriptName: worker.nodes.worker.scriptName,
          },
          { parent },
        ),
      );
    }
  }

  /**
   * The name of the Cloudflare Workflow.
   */
  public get workflowName() {
    return this.workflow.workflowName;
  }

  /**
   * The name of the workflow class.
   */
  public get className() {
    return this.workflow.className;
  }

  /**
   * The name of the Worker script that hosts the workflow class.
   */
  public get scriptName() {
    return this.workflow.scriptName;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Cloudflare Worker that hosts the workflow class.
       */
      worker: this.worker,
      /**
       * The Cloudflare Workflow resource.
       */
      workflow: this.workflow,
    };
  }

  /**
   * When you link a workflow to a worker, it automatically creates a
   * [Workflow binding](https://developers.cloudflare.com/workflows/build/workers-api/#call-workflows-from-workers)
   * on the worker. This lets you trigger, inspect, and manage workflow instances
   * from your worker code.
   *
   * @example
   * ```ts title="src/api.ts"
   * import { Resource } from "sst";
   *
   * const instance = await Resource.MyWorkflow.create({ params: { hello: "world" } });
   * ```
   *
   * @internal
   */
  getSSTLink() {
    return {
      properties: {
        workflowName: this.workflow.workflowName,
        className: this.workflow.className,
        scriptName: this.workflow.scriptName,
      },
      include: [
        binding({
          type: "workflowBindings",
          properties: {
            workflowName: this.workflow.workflowName,
            className: this.workflow.className,
            scriptName: this.workflow.scriptName,
          },
        }),
      ],
    };
  }
}

const __pulumiType = "sst:cloudflare:Workflow";
// @ts-expect-error
Workflow.__pulumiType = __pulumiType;
