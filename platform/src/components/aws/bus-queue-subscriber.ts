import { ComponentResourceOptions, Input, output } from "@pulumi/pulumi";
import { Component, transform } from "../component";
import { BusBaseSubscriberArgs, createRule } from "./bus-base-subscriber";
import { cloudwatch, sqs, iam } from "@pulumi/aws";
import { Queue } from "./queue";
import { parseArn } from "./helpers/arn";

export interface Args extends BusBaseSubscriberArgs {
  /**
   * The ARN of the SQS Queue.
   */
  queue: Input<string | Queue>;
}

/**
 * The `BusQueueSubscriber` component is internally used by the `Bus` component
 * to add subscriptions to [Amazon EventBridge Event Bus](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-event-bus.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribeQueue` method of the `Bus` component.
 */
export class BusQueueSubscriber extends Component {
  private readonly policy: Output<sqs.QueuePolicy>;
  private readonly rule: cloudwatch.EventRule;
  private readonly target: cloudwatch.EventTarget;

  constructor(name: string, args: Args, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const bus = output(args.bus);
    const busArn = bus.arn;
    const queueArn = output(args.queue).apply((queue) =>
      queue instanceof Queue ? queue.arn : output(queue),
    );

    // Detect cross-account scenario by comparing account IDs
    const isCrossAccount = output(busArn).apply((busArnStr) =>
      queueArn.apply((queueArnStr) => {
        const busParsed = parseArn(busArnStr);
        const queueParsed = parseArn(queueArnStr);
        const crossAccount = busParsed.account !== queueParsed.account;
        console.log("Cross-account detection:", {
          busAccount: busParsed.account,
          queueAccount: queueParsed.account,
          isCrossAccount: crossAccount,
        });
        return crossAccount;
      }),
    );

    // Create IAM role only for cross-account scenarios
    // This role allows EventBridge in the bus's account to send messages to the queue
    const targetRole = isCrossAccount.apply((crossAccount) => {
      if (!crossAccount) return undefined;

      // IAM role created in bus's account (using the provided provider)
      const role = new iam.Role(
        `${name}TargetRole`,
        {
          assumeRolePolicy: iam.assumeRolePolicyForPrincipal({
            Service: "events.amazonaws.com",
          }),
        },
        { provider: opts?.provider },
      );

      // Inline policy granting sqs:SendMessage to the target queue
      new iam.RolePolicy(
        `${name}TargetRolePolicy`,
        {
          role: role.id,
          policy: queueArn.apply((arn) =>
            JSON.stringify({
              Version: "2012-10-17",
              Statement: [
                {
                  Effect: "Allow",
                  Action: "sqs:SendMessage",
                  Resource: arn,
                },
              ],
            }),
          ),
        },
        { provider: opts?.provider },
      );

      return role;
    });

    const policy = createPolicy();
    const rule = createRule(name, bus.name, args, self);
    const target = createTarget();

    this.policy = policy;
    this.rule = rule;
    this.target = target;

    function createPolicy() {
      // For cross-account: create queue policy WITHOUT parent to force default provider
      // For same-account: use Queue.createPolicy with normal parent relationship
      return isCrossAccount.apply((crossAccount) => {
        if (crossAccount) {
          // Cross-account: Create policy directly with default provider (no parent)
          // This is CRITICAL - the policy must be in the queue's account, not bus's account
          // Get the role ARN for the queue policy principal
          const roleArn = targetRole.apply((role) => role?.arn);
          
          return new sqs.QueuePolicy(
            `${name}Policy`,
            {
              queueUrl: queueArn.apply((arn) => {
                // Parse SQS ARN: arn:aws:sqs:region:account-id:queue-name
                // Queue URL: https://sqs.{region}.amazonaws.com/{account-id}/{queue-name}
                const parsed = parseArn(arn);
                return `https://sqs.${parsed.region}.amazonaws.com/${parsed.account}/${parsed.resource}`;
              }),
              policy: iam.getPolicyDocumentOutput({
                statements: [
                  {
                    actions: ["sqs:SendMessage"],
                    resources: [queueArn],
                    principals: [
                      {
                        type: "AWS",
                        identifiers: [roleArn],
                      },
                    ],
                  },
                ],
              }).json,
            },
            { retainOnDelete: true }, // No parent, no provider = default provider
          );
        } else {
          // Same-account: Use normal Queue.createPolicy with parent
          return Queue.createPolicy(`${name}Policy`, queueArn, { parent: self });
        }
      });
    }

    function createTarget() {
      return new cloudwatch.EventTarget(
        ...transform(
          args?.transform?.target,
          `${name}Target`,
          {
            arn: queueArn,
            rule: rule.name,
            eventBusName: bus.name,
            // roleArn is required only for cross-account scenarios
            roleArn: targetRole?.arn,
          },
          { parent: self },
        ),
      );
    }
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The SQS Queue policy.
       */
      policy: this.policy,
      /**
       * The EventBus rule.
       */
      rule: this.rule,
      /**
       * The EventBus target.
       */
      target: this.target,
    };
  }
}

const __pulumiType = "sst:aws:BusQueueSubscriber";
// @ts-expect-error
BusQueueSubscriber.__pulumiType = __pulumiType;
