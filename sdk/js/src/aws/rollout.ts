import { Context } from "aws-lambda";
import { aws } from "./client.js";

/**
 * The `rollout` client SDK is available through the following.
 *
 * @example
 * ```js title="src/before-traffic.ts"
 * import { rollout } from "sst/aws/rollout";
 * ```
 */
export namespace rollout {
  export interface Event {
    /**
     * The CodeDeploy deployment ID.
     */
    deploymentId: string;
    /**
     * The lifecycle event hook execution ID. Pass this to `rollout.report` to
     * report the status of the hook.
     */
    lifecycleEventHookExecutionId: string;
  }

  /**
   * Creates a typed handler for a CodeDeploy lifecycle hook.
   *
   * @example
   * ```ts title="src/before-traffic.ts"
   * import { Resource } from "sst";
   * import { rollout } from "sst/aws/rollout";
   *
   * export const handler = rollout.handler(async (event) => {
   *   const resp = await fetch(Resource.Function.latestUrl);
   *   await rollout.report(event, resp.ok ? "Succeeded" : "Failed");
   * });
   * ```
   */
  export function handler(
    cb: (event: Event, context: Context) => Promise<void>,
  ) {
    return async (rawEvent: RawEvent, context: Context) => {
      if (!rawEvent.DeploymentId) {
        throw new Error(
          "Missing DeploymentId in event. This handler must be invoked by a CodeDeploy lifecycle hook.",
        );
      }
      if (!rawEvent.LifecycleEventHookExecutionId) {
        throw new Error(
          "Missing LifecycleEventHookExecutionId in event. This handler must be invoked by a CodeDeploy lifecycle hook.",
        );
      }
      await cb(
        {
          deploymentId: rawEvent.DeploymentId,
          lifecycleEventHookExecutionId:
            rawEvent.LifecycleEventHookExecutionId,
        },
        context,
      );
    };
  }

  /**
   * Reports the status of a CodeDeploy lifecycle hook back to CodeDeploy.
   *
   * @example
   * ```ts
   * await rollout.report(event, "Succeeded");
   * ```
   */
  export async function report(
    event: Event,
    status: "Succeeded" | "Failed",
    options?: { aws?: aws.Options },
  ) {
    const res = await aws.fetch(
      "codedeploy",
      "/",
      {
        method: "POST",
        headers: {
          "X-Amz-Target":
            "CodeDeploy_20141006.PutLifecycleEventHookExecutionStatus",
          "Content-Type": "application/x-amz-json-1.1",
        },
        body: JSON.stringify({
          deploymentId: event.deploymentId,
          lifecycleEventHookExecutionId: event.lifecycleEventHookExecutionId,
          status,
        }),
      },
      options,
    );
    if (!res.ok) {
      const body = await res.text();
      throw new Error(
        `Failed to report lifecycle status: ${res.status} ${body}`,
      );
    }
  }

  interface RawEvent {
    DeploymentId: string;
    LifecycleEventHookExecutionId: string;
  }
}
