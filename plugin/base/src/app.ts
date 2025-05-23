import { VisibleError } from "./error.js";

export interface App
  extends Readonly<{
    /**
     * The name of the current app.
     */
    name: string;
    /**
     * The stage currently being run. You can use this to conditionally deploy resources based
     * on the stage.
     *
     * For example, to deploy a bucket only in the `dev` stage:
     *
     * ```ts title="sst.config.ts"
     * if ($app.stage === "dev") {
     *   new sst.aws.Bucket("MyBucket");
     * }
     * ```
     */
    stage: string;
    /**
     * The removal policy for the current stage. If `removal` was not set in the `sst.config.ts`, this will be return its default value, `retain`.
     */
    removal: "remove" | "retain" | "retain-all";
    /**
     * If true, prevents `sst remove` from being executed on this stage
     */
    protect: boolean;
  }> {}

if (!process.env.SST_ENVIRONMENT)
  throw new VisibleError(
    "sst code must be run through the `sst` cli, not directly.",
  );

const parsed = JSON.parse(process.env.SST_ENVIRONMENT);
console.log(parsed);
export const app: App = parsed.app;

export const dev = parsed.dev;

export const path = parsed.path;

export const command: string = parsed.command;

export const version: Record<string, number> = parsed.version;
