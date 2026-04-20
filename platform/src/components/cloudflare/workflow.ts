import { all, ComponentResourceOptions, Output } from "@pulumi/pulumi";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component.js";
import type { Input } from "../input.js";
import { Link } from "../link.js";
import { DEFAULT_ACCOUNT_ID } from "./account-id.js";
import { binding } from "./binding.js";
import { WorkerArgs } from "./worker.js";
import { WorkerBuilder, workerBuilder } from "./helpers/worker-builder.js";

export interface WorkflowArgs {
  /**
   * The exported workflow class name from the worker script.
   */
  className: Input<string>;
  /**
   * The worker handler that defines the workflow.
   *
   * Pass a file path for the simple case, or full worker props if you need to
   * customize the worker.
   */
  handler: Input<string | WorkerArgs>;
  /**
   * The workflow name. If not provided, Cloudflare will auto-generate one.
   */
  workflowName?: Input<string>;
  /**
   * [Transform](/docs/components/#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Worker resource.
     */
    worker?: Transform<WorkerArgs>;
    /**
     * Transform the Workflow resource.
     */
    workflow?: Transform<cloudflare.WorkflowArgs>;
  };
}

/**
 * Use the `Workflow` component to register a
 * [Cloudflare Workflow](https://developers.cloudflare.com/workflows/) and the
 * worker that defines it.
 *
 * Link it to another worker to start workflow instances through `Resource`.
 *
 * @example
 *
 * #### Create a workflow
 *
 * Start with a worker file that exports your workflow class.
 *
 * ```ts title="src/workflow.ts"
 * import { WorkflowEntrypoint } from "cloudflare:workers";
 *
 * export default {
 *   async fetch() {
 *     return new Response("Workflow worker is ready.");
 *   },
 * };
 *
 * export class SignupWorkflow extends WorkflowEntrypoint {
 *   async run(_event, step) {
 *     await step.do("log workflow run", async () => {
 *       console.log("Cloudflare Workflow ran.");
 *     });
 *   }
 * }
 * ```
 *
 * Then register it in your app.
 *
 * ```ts title="sst.config.ts"
 * const workflow = new sst.cloudflare.Workflow("SignupWorkflow", {
 *   handler: "src/workflow.ts",
 *   className: "SignupWorkflow",
 * });
 * ```
 *
 * #### Customize the worker
 *
 * ```ts title="sst.config.ts"
 * const bucket = new sst.cloudflare.Bucket("MyBucket");
 *
 * const workflow = new sst.cloudflare.Workflow("SignupWorkflow", {
 *   className: "SignupWorkflow",
 *   handler: {
 *     handler: "src/workflow.ts",
 *     link: [bucket],
 *   },
 * });
 * ```
 *
 * #### Link it to another worker
 *
 * Link the workflow to another worker so it can start instances.
 *
 * ```ts title="sst.config.ts"
 * new sst.cloudflare.Worker("Api", {
 *   handler: "src/api.ts",
 *   link: [workflow],
 *   url: true,
 * });
 * ```
 *
 * Then call it from your worker.
 *
 * ```ts title="src/api.ts"
 * import { Resource } from "sst";
 *
 * export default {
 *   async fetch() {
 *     const instance = await Resource.SignupWorkflow.create();
 *
 *     return Response.json({ id: instance.id });
 *   },
 * };
 * ```
 */
export class Workflow extends Component implements Link.Linkable {
  private worker: WorkerBuilder;
  private workflow: Output<cloudflare.Workflow>;

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
      return workerBuilder(
        `${name}Worker`,
        args.handler,
        args.transform?.worker,
        { parent },
      );
    }

    function createWorkflow() {
      return all([worker, args.className, args.workflowName]).apply(
        ([worker, className, workflowName]) =>
          new cloudflare.Workflow(
            ...transform(
              args.transform?.workflow,
              `${name}Workflow`,
              {
                accountId: DEFAULT_ACCOUNT_ID,
                className,
                scriptName: worker.script.scriptName,
                workflowName: workflowName ?? "",
              },
              { parent },
            ),
          ),
      );
    }
  }

  /**
   * When you link a workflow to a worker, SST adds a Cloudflare
   * [Workflow binding](https://developers.cloudflare.com/workflows/build/workers-api/#workflow).
   *
   * @example
   * ```ts title="src/api.ts"
   * import { Resource } from "sst";
   *
   * const instance = await Resource.SignupWorkflow.create();
   * ```
   *
   * @internal
   */
  getSSTLink() {
    return {
      properties: {
        className: this.className,
        scriptName: this.scriptName,
        workflowName: this.workflowName,
      },
      include: [
        binding({
          type: "workflowBindings",
          properties: {
            className: this.className,
            scriptName: this.scriptName,
            workflowName: this.workflowName,
          },
        }),
      ],
    };
  }

  /**
   * The workflow ID.
   */
  public get id() {
    return this.workflow.apply((workflow) => workflow.id);
  }

  /**
   * The exported workflow class name.
   */
  public get className() {
    return this.workflow.apply((workflow) => workflow.className);
  }

  /**
   * The script name for the workflow worker.
   */
  public get scriptName() {
    return this.worker.apply((worker) => worker.script.scriptName);
  }

  /**
   * The workflow name.
   */
  public get workflowName() {
    return this.workflow.apply((workflow) => workflow.workflowName);
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    const self = this;
    return {
      /**
       * The Cloudflare Worker.
       */
      get worker() {
        return self.worker.apply((worker) => worker.getWorker());
      },
      /**
       * The Cloudflare workflow.
       */
      workflow: this.workflow,
    };
  }
}

const __pulumiType = "sst:cloudflare:Workflow";
// @ts-expect-error
Workflow.__pulumiType = __pulumiType;
