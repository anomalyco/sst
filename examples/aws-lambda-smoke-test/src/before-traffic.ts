import { LambdaClient, InvokeCommand } from "@aws-sdk/client-lambda";
import { Resource } from "sst";
import { rollout } from "sst/aws/rollout";

const lambda = new LambdaClient();

export const handler = rollout.handler(async (event) => {
  const status = await validate();
  await rollout.report(event, status);
});

async function validate(): Promise<"Succeeded" | "Failed"> {
  try {
    const resp = await lambda.send(
      new InvokeCommand({
        FunctionName: Resource.Function.name,
        Qualifier: Resource.Function.latestQualifier,
        Payload: JSON.stringify({ health: true }),
      }),
    );

    if (resp.FunctionError) {
      console.log("Invocation failed:", resp.FunctionError);
      return "Failed";
    }

    const payload = new TextDecoder().decode(resp.Payload);
    console.log("Health check passed:", payload);
    return "Succeeded";
  } catch (err) {
    console.error("Validation failed:", err);
    return "Failed";
  }
}
