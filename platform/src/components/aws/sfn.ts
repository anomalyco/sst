import aws, { sfn } from "@pulumi/aws";
import { EventRuleArgs } from "@pulumi/aws/cloudwatch";
import { StateMachineArgs } from "@pulumi/aws/sfn";
import { ComponentResourceOptions, Output, output } from "@pulumi/pulumi";
import { Component, Transform, transform } from "../component.js";

import { Link } from "../link.js";

import type { CronArgs } from "./cron.js";
import { permission } from "./permission.js";
import { Chainable } from "./sfn-states";

const region = aws.config.region;

type SFNArgs = Partial<Omit<StateMachineArgs, "definition">> & {
  definition: Chainable;
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the EventBus resource.
     */
    stateMachine?: Transform<sfn.StateMachineArgs>;
  };
};

export class StateMachine extends Component implements Link.Linkable {
  private stateMachine: Output<sfn.StateMachine>;
  private startExecutionRole?: aws.iam.Role;

  static __pulumiType: string;

  constructor(
    private name: string,
    args: SFNArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    const parent = this;
    const role = args.roleArn
      ? aws.iam.Role.get(
          `${$app.name}-${$app.stage}-${name}SfnRole`,
          args.roleArn,
        )
      : new aws.iam.Role(`${name}SfnRole`, {
          name: `${$app.name}-${$app.stage}-${name}`,
          assumeRolePolicy: aws.iam.assumeRolePolicyForPrincipal({
            Service: `states.${region}.amazonaws.com`,
          }),
        });

    this.stateMachine = output(createStateMachine());
    createPermissions();

    function createPermissions() {
      new aws.iam.RolePolicy(`${name}SfnRolePolicy`, {
        role: role.id,
        policy: {
          Version: "2012-10-17",
          Statement: [
            {
              Effect: "Allow",
              Action: ["events:*"],
              Resource: "*",
            },
          ],
        },
      });
      args.definition.createPermissions(role, name);
    }

    function createStateMachine() {
      return new sfn.StateMachine(
        ...transform(
          args.transform?.stateMachine,
          `${$app.name}-${$app.stage}-${name}StateMachine`,
          {
            name: args.name,
            definition: $jsonStringify(args.definition.serializeToDefinition()),
            roleArn: role.arn,
          },
          { parent },
        ),
      );
    }
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        id: this.id,
        arn: this.arn,
      },
      include: [
        permission({
          actions: ["states:*"],
          resources: [
            this.stateMachine.arn,
            $interpolate`${this.stateMachine.arn.apply((arn) =>
              arn.replace("stateMachine", "execution"),
            )}:*`,
          ],
        }),
      ],
    };
  }

  /**
   * The State Machine ID.
   */
  public get id() {
    return this.stateMachine.id;
  }

  /**
   * The State Machine ARN.
   */
  public get arn() {
    return this.stateMachine.arn;
  }

  getStartExecutionRole(): aws.iam.Role {
    if (this.startExecutionRole) {
      return this.startExecutionRole;
    }
    const roleName = `${$app.name}-${$app.stage}-${this.name}-StartExecutionRole`;
    const policyName = `${$app.name}-${$app.stage}-${this.name}-StartExecutionRolePolicy`;
    this.startExecutionRole = new aws.iam.Role(roleName, {
      name: roleName,
      assumeRolePolicy: JSON.stringify({
        Version: "2012-10-17",
        Statement: [
          {
            Action: "sts:AssumeRole",
            Effect: "Allow",
            Sid: "",
            Principal: {
              Service: "events.amazonaws.com",
            },
          },
          {
            Action: "sts:AssumeRole",
            Effect: "Allow",
            Sid: "",
            Principal: {
              Service: "apigateway.amazonaws.com",
            },
          },
        ],
      }),
    });
    new aws.iam.RolePolicy(policyName, {
      name: policyName,
      role: this.startExecutionRole.id,
      policy: $jsonStringify({
        Version: "2012-10-17",
        Statement: [
          {
            Action: ["states:StartExecution"],
            Effect: "Allow",
            Resource: this.stateMachine.arn,
          },
        ],
      }),
    });
    return this.startExecutionRole;
  }

  public addCronTrigger(
    name: string,
    schedule: CronArgs["schedule"],
    input?: Record<string, unknown>,
  ): aws.cloudwatch.EventRule {
    const rule = new aws.cloudwatch.EventRule(name, {
      name: `${$app.name}-${$app.stage}-${name}`,
      description: $interpolate`Cron trigger for State Machine ${this.stateMachine.name}`,
      scheduleExpression: schedule,
    });
    new aws.cloudwatch.EventTarget(name, {
      rule: rule.name,
      arn: this.stateMachine.arn,
      roleArn: this.getStartExecutionRole().arn,
      input: $jsonStringify(input),
    });
    return rule;
  }

  public addEventBridgeTrigger(
    name: string,
    eventPattern: EventRuleArgs["eventPattern"],
  ): aws.cloudwatch.EventRule {
    const rule = new aws.cloudwatch.EventRule(name, {
      name: `${$app.name}-${$app.stage}-${name}`,
      description: $interpolate`Event trigger for State Machine ${this.stateMachine.name}`,
      eventPattern,
    });
    new aws.cloudwatch.EventTarget(name, {
      rule: rule.name,
      arn: this.stateMachine.arn,
      roleArn: this.getStartExecutionRole().arn,
    });
    return rule;
  }
}

const __pulumiType = "sst:aws:StateMachine";
StateMachine.__pulumiType = __pulumiType;
