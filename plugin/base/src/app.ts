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

export const app: App = {
  name: "test",
  stage: "test",
  protect: true,
  removal: "retain",
};

export const dev = process.env.SST_DEV === "true";

export const paths = {
  root: process.env.SST_PATH_ROOT!,
  artifacts: process.env.SST_PATH_ARTIFACTS!,
  working: process.env.SST_PATH_WORKING!,
};

export const command = "unknown";
