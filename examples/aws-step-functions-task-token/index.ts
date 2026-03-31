import { SQSEvent } from "aws-lambda";
import { SFNClient, SendTaskSuccessCommand } from "@aws-sdk/client-sfn";

const sfn = new SFNClient();

export async function handler(e: SQSEvent) {
  // Parse the task token
  const { body } = e.Records[0];
  const { MyTaskToken } = JSON.parse(body);

  // Do some work
  await new Promise((resolve) => setTimeout(resolve, 1000));

  // Call `SendTaskSuccess` to mark the task done
  await sfn.send(
    new SendTaskSuccessCommand({
      taskToken: MyTaskToken,
      output: JSON.stringify({ result: "foo" }),
    })
  );

  return "ok";
}
