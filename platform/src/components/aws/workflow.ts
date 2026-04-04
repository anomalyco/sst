import { ComponentResourceOptions, Input, output } from "@pulumi/pulumi";
import { cloudwatch, iam, lambda } from "@pulumi/aws";
import { Duration, DurationDays, DurationMinutes } from "../duration.js";
import { Component, Transform } from "../component.js";
import { RETENTION } from "./logging.js";
import { Function, FunctionArgs } from "./function.js";

export interface WorkflowArgs
  extends Omit<
    FunctionArgs,
    | "concurrency"
    | "durable"
    | "injections"
    | "live"
    | "logging"
    | "retries"
    | "streaming"
    | "timeout"
    | "transform"
    | "url"
    | "versioning"
    | "_skipHint"
    | "_skipMetadata"
  > {
  /**
   * Number of days to retain the workflow execution state.
   *
   * @default `"14 days"`
   */
  retention?: Input<DurationDays>;
  /**
   * Configure timeout limits for the workflow.
   */
  timeout?: Input<{
    /**
     * Maximum execution time for the workflow across all durable invocations.
     *
     * @default `"15 minutes"`
     */
    global?: Input<Duration>;
    /**
     * Maximum execution time for each underlying Lambda invocation.
     *
     * @default `"20 seconds"`
     */
    invocation?: Input<DurationMinutes>;
  }>;
  /**
   * Configure the workflow logs in CloudWatch. Or pass in `false` to disable writing logs.
   * The log format is always set to `json`.
   *
   * @default `{retention: "1 month", format: "json"}`
   */
  logging?:
    | false
    | {
        /**
         * The duration the workflow logs are kept in CloudWatch.
         *
         * Not applicable when an existing log group is provided.
         *
         * @default `1 month`
         * @example
         * ```js
         * {
         *   logging: {
         *     retention: "forever"
         *   }
         * }
         * ```
         */
        retention?: Input<keyof typeof RETENTION>;
        /**
         * Assigns the given CloudWatch log group name to the workflow. This allows you to
         * pass in a previously created log group.
         *
         * By default, the workflow creates a new log group when it's created.
         *
         * @default Creates a log group
         * @example
         * ```js
         * {
         *   logging: {
         *     logGroup: "/existing/log-group"
         *   }
         * }
         * ```
         */
        logGroup?: Input<string>;
      };
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying resources.
   */
  transform?: {
    /**
     * Transform the Lambda Function resource.
     */
    function?: Transform<lambda.FunctionArgs>;
    /**
     * Transform the IAM Role resource.
     */
    role?: Transform<iam.RoleArgs>;
    /**
     * Transform the CloudWatch LogGroup resource.
     */
    logGroup?: Transform<cloudwatch.LogGroupArgs>;
  };
}

/**
 * The `Workflow` component lets you add serverless workflows to your app using
 * [AWS Lambda Durable Functions](https://docs.aws.amazon.com/lambda/latest/dg/durable-functions.html).
 *
 * It is a thin wrapper around the [`Function`](/docs/component/aws/function) component
 * with durable execution enabled.
 *
 * @example
 * ```ts title="sst.config.ts"
 * new sst.aws.Workflow("MyWorkflow", {
 *   handler: "src/workflow.handler",
 * });
 * ```
 */
export class Workflow extends Component {
  private readonly fn: Function;

  constructor(
    name: string,
    args: WorkflowArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    const {
      timeout: workflowTimeout,
      retention,
      logging: workflowLogging,
      transform: workflowTransform,
      ...functionArgs
    } = args;
    const invocationTimeout =
      workflowTimeout === undefined
        ? undefined
        : (output(workflowTimeout).apply(
            (timeout) => timeout?.invocation,
          ) as FunctionArgs["timeout"]);
    const globalTimeout =
      workflowTimeout === undefined
        ? undefined
        : (output(workflowTimeout).apply(
            (timeout) => timeout?.global,
          ) as Input<Duration>);
    const logging =
      workflowLogging === undefined
        ? undefined
        : (output(workflowLogging).apply((logging) => {
            if (logging === false) return false;
            return {
              ...logging,
              format: "json" as const,
            };
          }) as FunctionArgs["logging"]);

    this.fn = new Function(
      name,
      {
        ...functionArgs,
        timeout: invocationTimeout,
        durable: {
          timeout: globalTimeout,
          retention,
        },
        logging,
        transform: workflowTransform && {
          function: workflowTransform.function,
          role: workflowTransform.role,
          logGroup: workflowTransform.logGroup,
        },
      },
      { parent: this },
    );

    this.registerOutputs({
      name: this.name,
      arn: this.arn,
    });
  }

  /** @internal */
  public getFunction() {
    return this.fn;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return this.fn.nodes;
  }

  /**
   * The name of the Lambda function backing the workflow.
   */
  public get name() {
    return this.fn.name;
  }

  /**
   * The ARN of the Lambda function backing the workflow.
   */
  public get arn() {
    return this.fn.arn;
  }

  /**
   * Add environment variables lazily to the workflow after it is created.
   */
  public addEnvironment(environment: Input<Record<string, Input<string>>>) {
    return this.fn.addEnvironment(environment);
  }

  /** @internal */
  public getSSTLink() {
    const link = this.fn.getSSTLink();
    return {
      properties: {
        name: link.properties.name,
      },
      include: link.include,
    };
  }
}

const __pulumiType = "sst:aws:Workflow";
// @ts-expect-error
Workflow.__pulumiType = __pulumiType;
