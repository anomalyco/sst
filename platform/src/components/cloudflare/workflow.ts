import { ComponentResourceOptions } from "@pulumi/pulumi";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component.js";
import type { Input } from "../input.js";
import { Link } from "../link.js";
import { DEFAULT_ACCOUNT_ID } from "./account-id.js";
import { binding } from "./binding.js";

export interface WorkflowArgs {
  /**
   * The exported workflow class name from the worker script.
   */
  className: Input<string>;
  /**
   * The name of the worker script that defines the workflow.
   */
  scriptName: Input<string>;
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
     * Transform the Workflow resource.
     */
    workflow?: Transform<cloudflare.WorkflowArgs>;
  };
}

/**
 * The `Workflow` component lets you register a [Cloudflare Workflow](https://developers.cloudflare.com/workflows/) from a Worker script.
 *
 * Once created, you can [link it](/docs/linking/) to other workers. SST adds a workflow
 * binding so you can trigger and inspect workflow instances through `Resource`.
 *
 * @example
 *
 * #### Register a Workflow
 *
 * First create the Worker script that exports your workflow class.
 *
 * ```ts title="src/workflow.ts"
 * import { WorkflowEntrypoint } from "cloudflare:workers";
 *
 * export class SignupWorkflow extends WorkflowEntrypoint {
 *   async run(event, step) {
 *     await step.do("send welcome email", async () => {
 *       console.log("send email to", event.payload.userId);
 *     });
 *   }
 * }
 * ```
 *
 * Then register it in your app.
 *
 * ```ts title="sst.config.ts"
 * const workflowWorker = new sst.cloudflare.Worker("WorkflowWorker", {
 *   handler: "src/workflow.ts",
 * });
 *
 * const workflow = new sst.cloudflare.Workflow("SignupWorkflow", {
 *   className: "SignupWorkflow",
 *   scriptName: workflowWorker.scriptName,
 * });
 * ```
 *
 * #### Link to a worker
 *
 * Link the workflow to another worker so it can trigger instances.
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
 *     const instance = await Resource.SignupWorkflow.create({
 *       params: { userId: "123" },
 *     });
 *
 *     return Response.json({ id: instance.id });
 *   },
 * };
 * ```
 */
export class Workflow extends Component implements Link.Linkable {
  private workflow: cloudflare.Workflow;

  constructor(
    name: string,
    args: WorkflowArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    const parent = this;
    const workflow = createWorkflow();

    this.workflow = workflow;

    function createWorkflow() {
      return new cloudflare.Workflow(
        ...transform(
          args.transform?.workflow,
          `${name}Workflow`,
          {
            accountId: DEFAULT_ACCOUNT_ID,
            className: args.className,
            scriptName: args.scriptName,
            workflowName: args.workflowName ?? "",
          },
          { parent },
        ),
      );
    }
  }

  /**
   * When you link a workflow to a worker, it will be available through the Cloudflare
   * [Workflow binding API](https://developers.cloudflare.com/workflows/build/workers-api/#workflow).
   *
   * @example
   * ```ts title="src/api.ts"
   * import { Resource } from "sst";
   *
   * const instance = await Resource.SignupWorkflow.create({
   *   params: { userId: "123" },
   * });
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
   * The generated ID of the workflow.
   */
  public get id() {
    return this.workflow.id;
  }

  /**
   * The exported workflow class name.
   */
  public get className() {
    return this.workflow.className;
  }

  /**
   * The script name that defines the workflow.
   */
  public get scriptName() {
    return this.workflow.scriptName;
  }

  /**
   * The generated workflow name.
   */
  public get workflowName() {
    return this.workflow.workflowName;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
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
