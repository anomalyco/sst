import { all, ComponentResourceOptions, Output } from "@pulumi/pulumi";
import * as cf from "@pulumi/cloudflare";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component";
import { WorkerArgs } from "./worker";
import { DEFAULT_ACCOUNT_ID } from "./account-id.js";
import { Input } from "../input.js";
import { WorkerBuilder, workerBuilder } from "./helpers/worker-builder";
import { VisibleError } from "../error";

export interface CronArgs {
  /**
   * The worker that'll be executed when the cron job runs.
   * @deprecated Use `worker` instead.
   *
   * @example
   *
   * ```ts
   * {
   *   job: "src/cron.ts"
   * }
   * ```
   *
   * You can pass in the full worker props.
   *
   * ```ts
   * {
   *   job: {
   *     handler: "src/cron.ts",
   *     link: [bucket]
   *   }
   * }
   * ```
   */
  job?: Input<string | WorkerArgs>;
  /**
   * The worker that'll be executed when the cron job runs.
   *
   * @example
   *
   * ```ts
   * {
   *   worker: "src/cron.ts"
   * }
   * ```
   *
   * You can pass in the full worker props.
   *
   * ```ts
   * {
   *   worker: {
   *     handler: "src/cron.ts",
   *     link: [bucket]
   *   }
   * }
   * ```
   */
  worker?: Input<string | WorkerArgs>;
  /**
   * The schedule for the cron job.
   *
   * :::note
   * The cron job continues to run even after you exit `sst dev`.
   * :::
   *
   * @example
   *
   * You can use a [cron expression](https://developers.cloudflare.com/workers/configuration/cron-triggers/#supported-cron-expressions).
   *
   * ```ts
   * {
   *   schedules: ["* * * * *"]
   *   // schedules: ["*\/30 * * * *"]
   *   // schedules: ["45 * * * *"]
   *   // schedules: ["0 17 * * sun"]
   *   // schedules: ["10 7 * * mon-fri"]
   *   // schedules: ["0 15 1 * *"]
   *   // schedules: ["59 23 LW * *"]
   * }
   * ```
   */
  schedules: Input<string[]>;
  /**
   * [Transform](/docs/components/#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Worker Cron Trigger resource.
     */
    trigger?: Transform<cloudflare.WorkerCronTriggerArgs>;
  };
}

/**
 * The `Cron` component lets you add cron jobs to your app using Cloudflare.
 * It uses [Cloudflare Cron Triggers](https://developers.cloudflare.com/workers/configuration/cron-triggers/).
 *
 * @example
 * #### Minimal example
 *
 * Create a worker file that exposes a `scheduled` handler:
 *
 * ```ts title="cron.ts"
 * export default {
 *   async scheduled() {
 *     console.log("Running on a schedule");
 *   },
 * };
 * ```
 *
 * Pass in a `schedules` and a `worker` that'll be executed.
 *
 * ```ts title="sst.config.ts"
 * new sst.cloudflare.Cron("MyCronJob", {
 *   worker: "cron.ts",
 *   schedules: ["* * * * *"]
 * });
 * ```
 *
 * #### Customize the worker
 *
 * ```js title="sst.config.ts"
 * new sst.cloudflare.Cron("MyCronJob", {
 *   schedules: ["* * * * *"],
 *   worker: {
 *     handler: "cron.ts",
 *     link: [bucket]
 *   }
 * });
 * ```
 */
export class Cron extends Component {
  private worker: WorkerBuilder;
  private trigger: Output<cf.WorkerCronTrigger>;

  constructor(name: string, args: CronArgs, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const parent = this;

    const workerArgs = normalizeWorker();
    const worker = createWorker();
    const trigger = createTrigger();

    this.worker = worker;
    this.trigger = trigger;

    function normalizeWorker() {
      if (args.job && args.worker)
        throw new VisibleError(
          `You cannot provide both "job" and "worker" in the "${name}" Cron component. The "job" property has been deprecated. Use "worker" instead.`,
        );
      return args.worker ?? args.job;
    }

    function createWorker() {
      if (!workerArgs)
        throw new VisibleError(
          `You must provide a "worker" for the "${name}" Cron component.`,
        );
      return workerBuilder(`${name}Handler`, workerArgs);
    }

    function createTrigger() {
      return all([args.schedules]).apply(([schedules]) => {
        return new cloudflare.WorkersCronTrigger(
          ...transform(
            args.transform?.trigger,
            `${name}Trigger`,
            {
              accountId: DEFAULT_ACCOUNT_ID,
              scriptName: worker.script.scriptName,
              schedules: schedules.map((s) => ({ cron: s })),
            },
            { parent },
          ),
        );
      });
    }
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Cloudflare Worker.
       */
      worker: this.worker.script,
      /**
       * The Cloudflare Worker Cron Trigger.
       */
      trigger: this.trigger,
    };
  }
}

const __pulumiType = "sst:cloudflare:Cron";
// @ts-expect-error
Cron.__pulumiType = __pulumiType;
