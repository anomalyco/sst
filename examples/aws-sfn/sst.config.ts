/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Step Functions Example
 *
 * This example demonstrates how to create a Step Functions workflow that:
 * 1. Invokes a Lambda function
 * 2. Extracts the status from the response
 * 3. Makes a choice based on the status (COMPLETED/FAILED)
 * 4. Handles retries for failed invocations
 *
 * The Lambda function returns a random status (COMPLETED or FAILED) to demonstrate
 * the workflow's decision-making capabilities.
 *
 * ```ts title="index.ts"
 * import { Handler } from "aws-lambda";
 *
 * type Event = {
 *   name: string;
 * };
 *
 * type Result = {
 *   status: "COMPLETED" | "FAILED";
 * };
 *
 * export const handler: Handler<Event, Result> = async (event) => {
 *   console.log(event);
 *   return {
 *     status: Math.random() < 0.5 ? "COMPLETED" : "FAILED",
 *   };
 * };
 * ```
 *
 * The Step Functions workflow is configured with:
 * - Automatic retries for Lambda invocation failures
 * - JSONPath-based choice state for status evaluation
 * - Proper error handling with Fail and Succeed states
 *
 * To test this workflow, you can use the AWS Console or the AWS CLI:
 *
 * ```bash
 * aws stepfunctions start-execution \
 *   --state-machine-arn <your-state-machine-arn> \
 *   --input '{"name": "test"}'
 * ```
 *
 */
export default $config({
  app(input) {
    return {
      name: "aws-sfn",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const myFunction = new sst.aws.Function("MyWorkflowFunction", {
      handler: "index.handler",
    });

    const invokeMyFunction = new sst.aws.sfn.LambdaInvoke("InvokeMyFunction", {
      Parameters: {
        FunctionName: myFunction.arn,
        Payload: {
          "name.$": "$.name",
        },
      },
      ResultPath: "$.result",
    });
    invokeMyFunction.markAsStart();

    invokeMyFunction.addRetry({
      ErrorEquals: [
        "States.TaskFailed",
        "Lambda.ServiceException",
        "Lambda.AWSLambdaException",
        "Lambda.SdkClientException",
      ],
      MaxAttempts: 10,
      BackoffRate: 2,
    });

    const done = new sst.aws.sfn.Succeed("Done");
    const failed = new sst.aws.sfn.Fail("Failed");

    const checkCompletion = new sst.aws.sfn.Choice("CheckCompletion", {
      QueryLanguage: "JSONPath",
    })
      .whenEquals("$.status", "COMPLETED", done)
      .whenEquals("$.status", "FAILED", failed)
      .otherwise(failed);

    const extractStatus = new sst.aws.sfn.Pass("ExtractStatus", {
      Parameters: {
        "status.$": "$.result.Payload.status",
      },
    });

    new sst.aws.StateMachine("MyWorkflow", {
      definition: invokeMyFunction.next(extractStatus).next(checkCompletion),
    });
  },
});
