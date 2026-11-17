import { codedeploy, iam, lambda } from "@pulumi/aws";
import {
  all,
  ComponentResourceOptions,
  interpolate,
  Output,
  output,
} from "@pulumi/pulumi";
import { Component, Transform, transform } from "../component.js";
import { DurationMinutes, toSeconds } from "../duration.js";
import { VisibleError } from "../error.js";
import type { Input } from "../input.js";
import { Function, FunctionArgs } from "./function.js";
import { functionBuilder } from "./helpers/function-builder.js";
import { CodeDeployDeploymentWaiter } from "./providers/codedeploy-deployment-waiter.js";
import { CodeDeployLambdaDeployment } from "./providers/codedeploy-lambda-deployment.js";

const CODEDEPLOY_EVENT_MAP = {
  start: "DeploymentStart",
  success: "DeploymentSuccess",
  failure: "DeploymentFailure",
  stop: "DeploymentStop",
  rollback: "DeploymentRollback",
  ready: "DeploymentReady",
} as const;

type CodeDeployEvent = keyof typeof CODEDEPLOY_EVENT_MAP;

export interface FunctionRolloutArgs {
  function: Function;
  alias: Input<lambda.Alias>;
  /**
   * The rollout type when the function is updated.
   *
   * - `"canary"` — shifts a percentage of traffic to the new version, waits for the duration, then shifts 100%.
   * - `"linear"` — shifts a percentage of traffic to the new version every duration until 100%.
   * - `"all-at-once"` — shifts 100% of traffic to the new version immediately. Use `beforeTraffic` to run smoke tests that must pass before shifting.
   */
  type: Input<"canary" | "linear" | "all-at-once">;
  /**
   * The percentage of traffic to shift per step. Only used for `canary` and `linear`.
   * @default `10`
   */
  percentage?: Input<number>;
  /**
   * The time between each traffic shifting step. Only used for `canary` and `linear`.
   * @default `"10 minutes"`
   */
  duration?: Input<DurationMinutes>;
  /**
   * A list of CloudWatch alarm names. If any alarm enters `ALARM` state during the
   * deployment, traffic will roll back completely to the previous version.
   */
  alarms?: Input<Input<string>[]>;
  /**
   * A function to invoke before any traffic shifts to the new version. Can be
   * a handler path, `Function` instance, `FunctionArgs`, or ARN. If it reports
   * failure, the deployment is aborted and all traffic stays on the previous
   * version.
   *
   * @example
   *
   * ```ts title="sst.config.ts"
   * new sst.aws.Function("Function", {
   *   handler: "src/api.handler",
   *   rollout: {
   *     type: "all-at-once",
   *     beforeTraffic: "src/before-traffic.handler",
   *   },
   * });
   * ```
   *
   * For a complete example, see the [Lambda smoke test example](/docs/examples/#aws-lambda-smoke-test-function-url).
   *
   * If passing a `Function` reference, make sure to grant it the
   * `codedeploy:PutLifecycleEventHookExecutionStatus` permission.
   */
  beforeTraffic?: Input<string | FunctionArgs | Function>;
  /**
   * A function to invoke after all traffic has shifted to the new version. Can
   * be a handler path, `Function` instance, `FunctionArgs`, or ARN. If it
   * reports failure, the deployment is rolled back and traffic returns to the
   * previous version.
   *
   * @example
   *
   * ```ts title="sst.config.ts"
   * new sst.aws.Function("Function", {
   *   handler: "src/api.handler",
   *   rollout: {
   *     type: "all-at-once",
   *     afterTraffic: "src/after-traffic.handler",
   *   },
   * });
   * ```
   *
   * For a complete example, see the [Lambda smoke test example](/docs/examples/#aws-lambda-smoke-test-function-url).
   *
   * If passing a `Function` reference, make sure to grant it the
   * `codedeploy:PutLifecycleEventHookExecutionStatus` permission.
   */
  afterTraffic?: Input<string | FunctionArgs | Function>;
  /**
   * Whether SST should wait for the CodeDeploy deployment to complete.
   * @default `true` for `all-at-once`, `false` for `canary` and `linear`
   */
  wait?: Input<boolean>;
  /**
   * What to do when a new version is deployed while a previous rollout is still
   * in progress.
   *
   * - `"cancel"` — stop the existing rollout where it is, leaving traffic split
   *   between the old and new version, then start the new rollout.
   * - `"rollback"` — roll back the existing rollout so all traffic returns to the
   *   previous version, then start the new rollout.
   * - `"fail"` — error out and don't deploy.
   *
   * @default `"cancel"`
   */
  onConflict?: Input<"rollback" | "cancel" | "fail">;
  /**
   * Configure SNS notifications for deployment events.
   *
   * @example
   *
   * ```js
   * {
   *   rollout: {
   *     type: "canary",
   *     notifications: [
   *       {
   *         name: "OnFailure",
   *         events: ["failure", "rollback"],
   *         topic: myTopic.arn,
   *       },
   *     ],
   *   }
   * }
   * ```
   */
  notifications?: Input<
    Input<{
      /**
       * A name for this notification trigger.
       */
      name: Input<string>;
      /**
       * The deployment events to notify on.
       *
       * - `"start"` — deployment started.
       * - `"success"` — deployment completed successfully.
       * - `"failure"` — deployment failed.
       * - `"stop"` — deployment was stopped.
       * - `"rollback"` — deployment was rolled back.
       * - `"ready"` — traffic shifting is ready to proceed.
       */
      events: Input<Input<CodeDeployEvent>[]>;
      /**
       * The ARN of the SNS topic to notify.
       */
      topic: Input<string>;
    }>[]
  >;
  /**
   * [Transform](/docs/components#transform) how the rollout creates its underlying resources.
   */
  transform?: {
    /**
     * Transform the CodeDeploy Application resource.
     */
    application?: Transform<codedeploy.ApplicationArgs>;
    /**
     * Transform the CodeDeploy Deployment Group resource.
     */
    deploymentGroup?: Transform<codedeploy.DeploymentGroupArgs>;
    /**
     * Transform the IAM Role resource used by CodeDeploy.
     */
    role?: Transform<iam.RoleArgs>;
    /**
     * Transform the before-traffic Function.
     */
    beforeTrafficFunction?: Transform<FunctionArgs>;
    /**
     * Transform the after-traffic Function.
     */
    afterTrafficFunction?: Transform<FunctionArgs>;
  };
}

export class FunctionRollout extends Component {
  private codedeployApp: codedeploy.Application;
  private codedeployRole: iam.Role;
  private deploymentGroup: Output<codedeploy.DeploymentGroup>;
  private codedeployDeployment: CodeDeployLambdaDeployment;
  private deploymentWaiter: CodeDeployDeploymentWaiter;

  constructor(
    name: string,
    args: FunctionRolloutArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);
    const parent = this;

    const strategy = output(args.type);
    const percentage = output(args.percentage ?? 10);
    const interval = output(args.duration ?? "10 minutes").apply((v) =>
      Math.round(toSeconds(v) / 60),
    );
    const rolloutWait = output(
      args.wait ?? strategy.apply((s) => s === "all-at-once"),
    );
    const onConflict = output(args.onConflict ?? "cancel");
    const notifications = output(args.notifications ?? []);
    const alarms = output(args.alarms ?? []);

    const beforeTrafficFnArn = buildTrafficFn(
      `${name}BeforeTraffic`,
      args.beforeTraffic,
      args.transform?.beforeTrafficFunction,
    );
    const afterTrafficFnArn = buildTrafficFn(
      `${name}AfterTraffic`,
      args.afterTraffic,
      args.transform?.afterTrafficFunction,
    );

    const invokeResources = [
      beforeTrafficFnArn?.arn,
      afterTrafficFnArn?.arn,
    ].filter((i) => i != null);

    const codedeployApp = createCodedeployApp();
    const deploymentConfigName = createDeploymentConfigName();
    const codedeployRole = createCodedeployRole();
    const deploymentGroup = createDeploymentGroup();
    const codedeployDeployment = createCodedeployDeployment();
    const deploymentWaiter = createDeploymentWaiter();

    this.codedeployApp = codedeployApp;
    this.codedeployRole = codedeployRole;
    this.deploymentGroup = deploymentGroup;
    this.codedeployDeployment = codedeployDeployment;
    this.deploymentWaiter = deploymentWaiter;

    all([deploymentWaiter.status, deploymentWaiter.deploymentId]).apply(
      ([status, deploymentId]) => {
        if (status === "Failed" || status === "Stopped") {
          throw new VisibleError(
            `Rollout for "${name}" failed. Update your function code and deploy again.\n\nFor more details, check the CodeDeploy deployment in the AWS console: ${deploymentId}`,
          );
        }
      },
    );

    function createCodedeployApp() {
      return new codedeploy.Application(
        ...transform(
          args.transform?.application,
          `${name}CodeDeployApp`,
          {
            computePlatform: "Lambda",
          },
          { parent },
        ),
      );
    }

    function createDeploymentConfigName() {
      const builtInConfigName = all([strategy, percentage, interval]).apply(
        ([strategy, percentage, interval]): string | undefined => {
          if (strategy === "all-at-once")
            return "CodeDeployDefault.LambdaAllAtOnce";
          return resolveBuiltInLambdaConfig(strategy, percentage, interval);
        },
      );

      const customDeployConfig = all([strategy, percentage, interval]).apply(
        ([strategy, percentage, interval]) => {
          if (strategy === "all-at-once") return;
          if (resolveBuiltInLambdaConfig(strategy, percentage, interval))
            return;
          return new codedeploy.DeploymentConfig(
            `${name}DeployConfig`,
            {
              computePlatform: "Lambda",
              trafficRoutingConfig: {
                type:
                  strategy === "canary" ? "TimeBasedCanary" : "TimeBasedLinear",
                timeBasedCanary:
                  strategy === "canary" ? { interval, percentage } : undefined,
                timeBasedLinear:
                  strategy === "linear" ? { interval, percentage } : undefined,
              },
            },
            { parent },
          );
        },
      );

      const customConfigName = customDeployConfig.apply(
        (config) => config?.deploymentConfigName ?? "",
      );
      const deploymentConfigName = all([
        builtInConfigName,
        customConfigName,
      ]).apply(([builtIn, custom]) => builtIn ?? custom);

      return deploymentConfigName;
    }

    function resolveBuiltInLambdaConfig(
      strategy: string,
      percentage: number,
      interval: number,
    ): string | undefined {
      if (strategy === "canary" && percentage === 10) {
        const map: Record<number, string> = {
          5: "CodeDeployDefault.LambdaCanary10Percent5Minutes",
          10: "CodeDeployDefault.LambdaCanary10Percent10Minutes",
          15: "CodeDeployDefault.LambdaCanary10Percent15Minutes",
          30: "CodeDeployDefault.LambdaCanary10Percent30Minutes",
        };
        return map[interval];
      }
      if (strategy === "linear" && percentage === 10) {
        const map: Record<number, string> = {
          1: "CodeDeployDefault.LambdaLinear10PercentEvery1Minute",
          2: "CodeDeployDefault.LambdaLinear10PercentEvery2Minutes",
          3: "CodeDeployDefault.LambdaLinear10PercentEvery3Minutes",
          10: "CodeDeployDefault.LambdaLinear10PercentEvery10Minutes",
        };
        return map[interval];
      }
      return undefined;
    }

    function buildTrafficFn(
      name: string,
      input: Input<string | FunctionArgs | Function> | undefined,
      transform: Transform<FunctionArgs> | undefined,
    ) {
      if (!input) return undefined;
      return functionBuilder(
        name,
        input,
        {
          link: [args.function],
          permissions: [
            {
              actions: ["codedeploy:PutLifecycleEventHookExecutionStatus"],
              resources: ["*"],
            },
          ],
          _skipHint: true,
        },
        transform,
        { parent },
      );
    }

    function createCodedeployRole() {
      return new iam.Role(
        ...transform(
          args.transform?.role,
          `${name}CodeDeployRole`,
          {
            assumeRolePolicy: iam.assumeRolePolicyForPrincipal({
              Service: "codedeploy.amazonaws.com",
            }),
            inlinePolicies: [
              {
                name: "CodeDeployLambdaPolicy",
                policy: all([notifications]).apply(
                  ([notifications]) =>
                    iam.getPolicyDocumentOutput({
                      statements: [
                        {
                          actions: ["lambda:GetAlias", "lambda:UpdateAlias"],
                          resources: [
                            args.function.arn,
                            interpolate`${args.function.arn}:*`,
                          ],
                        },
                        ...(invokeResources.length > 0
                          ? [
                              {
                                actions: ["lambda:InvokeFunction"],
                                resources: invokeResources,
                              },
                            ]
                          : []),
                        {
                          actions: ["cloudwatch:DescribeAlarms"],
                          resources: ["*"],
                        },
                        ...(notifications.length > 0
                          ? [
                              {
                                actions: ["sns:Publish"],
                                resources: notifications.map(
                                  (item) => item.topic,
                                ),
                              },
                            ]
                          : []),
                      ],
                    }).json,
                ),
              },
            ],
          },
          { parent },
        ),
      );
    }

    function createDeploymentGroup() {
      return all([alarms]).apply(
        ([alarms]) =>
          new codedeploy.DeploymentGroup(
            ...transform(
              args.transform?.deploymentGroup,
              `${name}DeploymentGroup`,
              {
                deploymentGroupName: "",
                appName: codedeployApp.name,
                serviceRoleArn: codedeployRole.arn,
                deploymentConfigName,
                deploymentStyle: {
                  deploymentType: "BLUE_GREEN",
                  deploymentOption: "WITH_TRAFFIC_CONTROL",
                },
                autoRollbackConfiguration: {
                  enabled: true,
                  events: ["DEPLOYMENT_FAILURE", "DEPLOYMENT_STOP_ON_ALARM"],
                },
                alarmConfiguration:
                  alarms.length > 0
                    ? {
                        enabled: true,
                        alarms,
                      }
                    : undefined,
                triggerConfigurations: all([notifications]).apply(([n]) =>
                  n.map((notification) => ({
                    triggerEvents: notification.events.map(
                      (e) => CODEDEPLOY_EVENT_MAP[e],
                    ),
                    triggerName: notification.name,
                    triggerTargetArn: notification.topic,
                  })),
                ),
              },
              { parent },
            ),
          ),
      );
    }

    function createCodedeployDeployment() {
      return new CodeDeployLambdaDeployment(
        `${name}RolloutDeployment`,
        {
          applicationName: codedeployApp.name,
          deploymentGroupName: deploymentGroup.deploymentGroupName,
          functionName: args.function.name,
          aliasName: output(args.alias).apply((a) => a.name),
          targetVersion: args.function.nodes.function.version,
          onConflict,
          beforeTrafficFnArn: beforeTrafficFnArn?.arn,
          afterTrafficFnArn: afterTrafficFnArn?.arn,
        },
        { parent },
      );
    }

    function createDeploymentWaiter() {
      return new CodeDeployDeploymentWaiter(
        `${name}RolloutWaiter`,
        {
          deploymentId: codedeployDeployment.deploymentId,
          wait: rolloutWait,
          trigger: Date.now().toString(),
        },
        { parent },
      );
    }
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The CodeDeploy Application.
       */
      application: this.codedeployApp,
      /**
       * The IAM Role used by CodeDeploy.
       */
      role: this.codedeployRole,
      /**
       * The CodeDeploy Deployment Group.
       */
      deploymentGroup: this.deploymentGroup,
      /**
       * The CodeDeploy deployment.
       */
      deployment: this.codedeployDeployment,
      /**
       * The deployment waiter.
       */
      waiter: this.deploymentWaiter,
    };
  }
}
const __pulumiType = "sst:aws:FunctionRollout";
// @ts-expect-error
FunctionRollout.__pulumiType = __pulumiType;
